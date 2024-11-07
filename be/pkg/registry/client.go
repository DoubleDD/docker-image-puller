package registry

import (
	"crypto/md5"
	"docker-image-handler/config"
	"docker-image-handler/pkg/utils"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// 新增：创建带有认证的请求
func (c *httpClient) newRequest(method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	return req, nil
}

// NewDockerRegistryClient 创建新的 DockerRegistryClient 实例
func NewDockerRegistryClient(registry string, opts ...func(*DockerRegistryClient)) *DockerRegistryClient {
	client := &DockerRegistryClient{
		Registry: registry,
		jwtCache: make(map[string]jwtCacheItem),
	}

	// 应用所有选项
	for _, opt := range opts {
		opt(client)
	}

	return client
}

// 提供设置认证信息的选项函数
func WithAuth(username, password string) func(*DockerRegistryClient) {
	if username == "" || password == "" {
		return DefaultAuth()
	} else {
		return func(c *DockerRegistryClient) {
			c.Username = username
			c.Password = password
		}
	}
}

func DefaultAuth() func(*DockerRegistryClient) {
	cfg := config.Load()
	return func(c *DockerRegistryClient) {
		c.Username = cfg.DefaultUsername
		c.Password = cfg.DefaultPassword
	}
}

func parseJWTExpireTime(token string) (time.Time, error) {
	// 获取 JWT 的第二部分（payload）
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return time.Time{}, fmt.Errorf("invalid JWT format")
	}

	// 解码 base64
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}, err
	}

	// 解析 JSON
	var claims jwtPayload
	if err := json.Unmarshal(payload, &claims); err != nil {
		return time.Time{}, err
	}

	return time.Unix(claims.Exp, 0), nil
}

// SSE协议下载文件，持续输出
func (client *DockerRegistryClient) DownloadBlobWithSSE(repo, tag, digest string, size int64, c *gin.Context) error {
	httpClient, err := client.newAuthenticatedClient(repo, tag)
	if err != nil {
		return err
	}

	req, err := httpClient.newRequest("GET", fmt.Sprintf("/blobs/%s", digest), nil)
	if err != nil {
		return err
	}

	// 打印 curl 命令
	curlCmd := generateCurlCommand(req)
	fmt.Printf("\nCURL command for replay:\n%s\n\n", curlCmd)

	resp, err := httpClient.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download blob: %d", resp.StatusCode)
	}

	maxBufferSize := 2 * 1024 * 1024

	// 使用缓冲读取并报告进度
	buffer := make([]byte, maxBufferSize) // 1MB 缓冲区
	var downloaded, chunkNumber, totalRead int64

	for {
		n, err := resp.Body.Read(buffer[totalRead:])
		if n > 0 {
			totalRead += int64(n)
			downloaded += int64(n)
			// 计算进度
			percentage := float64(downloaded) / float64(size) * 100
			// 发送进度事件
			c.SSEvent("progress", gin.H{
				"size":       n,
				"digest":     digest,
				"downloaded": downloaded,
				"total":      size,
				"percentage": percentage,
			})
			c.Writer.Flush()
		}
		// 如果达到 5MB 或文件已读取完，则发送数据块
		if totalRead >= int64(maxBufferSize) || (err == io.EOF && totalRead > 0) {
			md5Hash, _ := utils.DataHash(buffer[:totalRead], md5.New)
			// 发送数据块
			c.SSEvent("data", gin.H{
				"no":   chunkNumber,
				"md5":  md5Hash,
				"data": base64.StdEncoding.EncodeToString(buffer[:totalRead]),
			})
			c.Writer.Flush()
			totalRead = 0 // 重置缓冲区
			chunkNumber++
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	// 发送完成事件
	c.SSEvent("complete", gin.H{
		"digest": digest,
		"size":   size,
	})
	c.Writer.Flush()

	return nil
}

// 新增：生成 curl 命令的辅助函数
func generateCurlCommand(req *http.Request) string {
	// 基础命令
	cmd := fmt.Sprintf("curl -X %s", req.Method)

	// 添加所有请求头
	for key, values := range req.Header {
		for _, value := range values {
			cmd += fmt.Sprintf(" \\\n  -H '%s: %s'", key, value)
		}
	}

	// 添加完整 URL
	fullURL := req.URL.String()
	if !strings.HasPrefix(fullURL, "http") {
		if strings.HasPrefix(fullURL, "/") {
			fullURL = fmt.Sprintf("%s://%s%s", req.URL.Scheme, req.URL.Host, fullURL)
		} else {
			fullURL = fmt.Sprintf("%s://%s/%s", req.URL.Scheme, req.URL.Host, fullURL)
		}
	}
	cmd += fmt.Sprintf(" \\\n  '%s'", fullURL)

	// 添加一些有用的 curl 选项
	cmd += " \\\n  -v" // 添加详细输出
	cmd += " \\\n  -L" // 跟随重定向

	return cmd
}

func (client *DockerRegistryClient) getJWT(repo, tag string) (string, error) {
	cacheKey := fmt.Sprintf("%s:%s", repo, tag)

	// 检查缓存
	client.mutex.RLock()
	if item, exists := client.jwtCache[cacheKey]; exists && time.Now().Before(item.expireTime) {
		client.mutex.RUnlock()
		return item.token, nil
	}
	client.mutex.RUnlock()

	// 缓存不存在或已过期，重新获取token
	authURL, err := client.getAuthURL(repo, tag)
	if err != nil {
		return "", err
	}

	token, err := client.getToken(authURL)
	if err != nil {
		return "", err
	}

	// 从 JWT 中解析过期时间
	expireTime, err := parseJWTExpireTime(token)
	if err != nil {
		// 如果解析失败，使用默认的1小时过期时间
		expireTime = time.Now().Add(1 * time.Hour)
	}

	// 将新token存入缓存
	client.mutex.Lock()
	client.jwtCache[cacheKey] = jwtCacheItem{
		token:      token,
		expireTime: expireTime,
	}
	client.mutex.Unlock()

	return token, nil
}

// GetManifest 获取镜像的 manifest
func (client *DockerRegistryClient) GetManifest(repo, tag string) (map[string]interface{}, error) {
	return client.fetchManifest(repo, tag)
}

// 获取 authentication URL
func (client *DockerRegistryClient) getAuthURL(repo, tag string) (string, error) {
	url := fmt.Sprintf("https://%s/v2/%s/manifests/%s", client.Registry, repo, tag)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		return "", errors.New("expected 401 Unauthorized response to extract auth URL")
	}

	authHeader := resp.Header.Get("WWW-Authenticate")
	if authHeader == "" {
		return "", errors.New("WWW-Authenticate header not found")
	}

	// 使用更可靠的解析方法
	parts := strings.Split(strings.TrimPrefix(authHeader, "Bearer "), ",")
	params := make(map[string]string)

	for _, part := range parts {
		if strings.Contains(part, "=") {
			kv := strings.SplitN(part, "=", 2)
			key := strings.TrimSpace(kv[0])
			// 移除首尾的引号
			value := strings.Trim(strings.TrimSpace(kv[1]), `"`)
			params[key] = value
		}
	}

	realm := params["realm"]
	service := params["service"]
	scope := params["scope"]

	if realm == "" {
		return "", errors.New("realm not found in auth header")
	}

	return fmt.Sprintf("%s?service=%s&scope=%s", realm, service, scope), nil
}

// 获取 JWT token
func (client *DockerRegistryClient) getToken(authURL string) (string, error) {
	req, err := http.NewRequest("GET", authURL, nil)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(client.Username, client.Password)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("failed to obtain token")
	}

	var tokenResponse TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return "", err
	}

	return tokenResponse.Token, nil
}

// 新增：在 DockerRegistryClient 中添加创建认证客户端的方法
func (client *DockerRegistryClient) newAuthenticatedClient(repo, tag string) (*httpClient, error) {
	token, err := client.getJWT(repo, tag)
	if err != nil {
		return nil, err
	}

	return &httpClient{
		client:  http.DefaultClient,
		token:   token,
		baseURL: fmt.Sprintf("https://%s/v2/%s", client.Registry, repo),
	}, nil
}

// 使用 token 获取 manifest
func (client *DockerRegistryClient) fetchManifest(repo, tag string) (map[string]interface{}, error) {
	httpClient, err := client.newAuthenticatedClient(repo, tag)
	if err != nil {
		return nil, err
	}

	req, err := httpClient.newRequest("GET", fmt.Sprintf("/manifests/%s", tag), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")

	resp, err := httpClient.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to fetch manifest")
	}

	var manifest map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, err
	}

	return manifest, nil
}

// ... 其他Registry相关方法
