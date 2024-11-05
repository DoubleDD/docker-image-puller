package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type Manifest struct {
	SchemaVersion int    `json:"schemaVersion"`
	MediaType     string `json:"mediaType"`
	Config        struct {
		Digest string `json:"digest"`
		Size   int    `json:"size"`
	} `json:"config"`
	Layers []struct {
		MediaType string `json:"mediaType"`
		Size      int    `json:"size"`
		Digest    string `json:"digest"`
	} `json:"layers"`
}

type MergeRequest struct {
	Manifest Manifest `json:"manifest"`
	Layers   []string `json:"layers"`
}

// 新增：获取registry认证token的响应结构
type TokenResponse struct {
	Token string `json:"token"`
}

// 新增：解析镜像地址的函数
func parseImageAddress(imageAddress string) (registry, repository, tag string, err error) {
	// 移除可能存在的协议前缀
	imageAddress = strings.TrimPrefix(imageAddress, "http://")
	imageAddress = strings.TrimPrefix(imageAddress, "https://")

	// 分离tag
	parts := strings.Split(imageAddress, ":")
	if len(parts) > 2 {
		return "", "", "", fmt.Errorf("invalid image address format")
	}

	address := parts[0]
	if len(parts) == 2 {
		tag = parts[1]
	} else {
		tag = "latest"
	}

	// 分离registry和repository
	addressParts := strings.Split(address, "/")
	if len(addressParts) < 2 {
		return "", "", "", fmt.Errorf("invalid image address format")
	}

	// 判断第一部分是否包含域名特征（包含'.'或':'）
	if strings.Contains(addressParts[0], ".") || strings.Contains(addressParts[0], ":") {
		registry = addressParts[0]
		repository = strings.Join(addressParts[1:], "/")
	} else {
		// 默认使用Docker Hub
		registry = "registry.hub.docker.com"
		repository = imageAddress
	}

	return registry, repository, tag, nil
}

// DockerRegistryClient 是用于与 Docker Registry API 交互的客户端结构体
type DockerRegistryClient struct {
	Username string
	Password string
	Registry string
	jwtCache map[string]jwtCacheItem // 新增：JWT缓存
	mutex    sync.RWMutex            // 新增：用于并发安全
}

type jwtCacheItem struct {
	token      string
	expireTime time.Time
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
	return func(c *DockerRegistryClient) {
		c.Username = "kedong@yunlizhihui"
		c.Password = "kedong@123"
	}
}

type jwtPayload struct {
	Exp int64 `json:"exp"`
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
	// 获取 JWT Token
	token, err := client.getJWT(repo, tag)
	if err != nil {
		return nil, err
	}

	// Step 3: 使用 Token 获取 manifest
	manifest, err := client.fetchManifest(repo, tag, token)
	if err != nil {
		return nil, err
	}

	return manifest, nil
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

// 使用 token 获取 manifest
func (client *DockerRegistryClient) fetchManifest(repo, tag, token string) (map[string]interface{}, error) {
	url := fmt.Sprintf("https://%s/v2/%s/manifests/%s", client.Registry, repo, tag)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")

	resp, err := http.DefaultClient.Do(req)
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

// 获取镜像 manifest 的 API 端点
func getManifestHandler(c *gin.Context) {
	image := c.Query("image")
	registry, repo, tag, err := parseImageAddress(image)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	username := c.Query("username")
	password := c.Query("password")

	client := NewDockerRegistryClient(registry, WithAuth(username, password))
	manifest, err := client.GetManifest(repo, tag)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, manifest)
}

func main() {
	r := gin.Default()
	dip := r.Group("dip")
	{
		dip.GET("/api/docker/manifest", getManifestHandler)
	}

	r.Run(":8888") // 启动服务
}
