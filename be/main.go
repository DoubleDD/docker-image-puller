package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	dip := r.Group("dip")
	{
		dip.GET("/api/docker/manifest", getManifestHandler)
		dip.GET("/api/docker/blob/download", startBlobDownloadHandler)       // 新增
		dip.GET("/api/docker/blob/progress", getBlobDownloadProgressHandler) // 新增
	}

	r.Run(":8888") // 启动服务
}

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

// 下载任务的状态结构
type DownloadTask struct {
	TaskID   string                 `json:"taskId"`
	Status   string                 `json:"status"` // pending, downloading, completed, failed
	Progress map[string]LayerStatus `json:"progress"`
	Error    string                 `json:"error,omitempty"`
	mutex    sync.RWMutex
}

type LayerStatus struct {
	Size       int64   `json:"size"`
	Downloaded int64   `json:"downloaded"`
	Percentage float64 `json:"percentage"`
	Status     string  `json:"status"`
}

// 全局任务管理器
var (
	downloadTasks = make(map[string]*DownloadTask)
	tasksMutex    sync.RWMutex
)

type BlobDownloadRequest struct {
	Image  string `json:"image"`  // 镜像地址
	Digest string `json:"digest"` // 层的digest
	Size   int64  `json:"size"`   // 层的大小
}

func startBlobDownloadHandler(c *gin.Context) {
	image := c.Query("image")
	digest := c.Query("digest")
	// 获取查询参数 "size" 并尝试转换为 int64
	sizeStr := c.Query("size")
	// 将字符串转换为 int64，设置基数为 10，位数为 64
	size, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil {
		// 处理转换错误，例如返回400响应
		c.JSON(400, gin.H{"error": "Invalid size parameter"})
		return
	}

	registry, repo, tag, err := parseImageAddress(image)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	username := c.Query("username")
	password := c.Query("password")

	// 创建下载任务
	taskID := fmt.Sprintf("%d", time.Now().UnixNano())

	// 设置SSE响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	// 发送任务ID
	c.SSEvent("taskId", taskID)
	c.Writer.Flush()

	client := NewDockerRegistryClient(registry, WithAuth(username, password))

	// 开始下载
	err = client.downloadBlobWithSSE(repo, tag, digest, size, c)
	if err != nil {
		c.SSEvent("error", err.Error())
		c.Writer.Flush()
		return
	}
}

// 修改 DockerRegistryClient 的下载方法以支持SSE
func (client *DockerRegistryClient) downloadBlobWithSSE(repo, tag, digest string, size int64, c *gin.Context) error {
	httpClient, err := client.newAuthenticatedClient(repo, tag)
	if err != nil {
		return err
	}

	req, err := httpClient.newRequest("GET", fmt.Sprintf("/blobs/%s", digest), nil)
	if err != nil {
		return err
	}

	resp, err := httpClient.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download blob: %d", resp.StatusCode)
	}

	// 使用 sendfile 进行零拷贝传输
	if f, ok := resp.Body.(*os.File); ok {
		// 这行代码使用 gin 框架的 DataFromReader 方法来实现零拷贝传输
		// - http.StatusOK: 设置 HTTP 响应状态码为 200
		// - resp.ContentLength: 设置响应内容的长度
		// - resp.Header.Get("Content-Type"): 设置响应的 Content-Type 头部
		// - f: 直接从文件对象读取数据并写入响应
		// - nil: 不使用额外的 headers
		c.DataFromReader(http.StatusOK, resp.ContentLength, resp.Header.Get("Content-Type"), f, nil)
	} else {
		// 使用缓冲读取并报告进度
		buffer := make([]byte, 32*1024) // 32KB 缓冲区
		var downloaded int64

		for {
			n, err := resp.Body.Read(buffer)
			if n > 0 {
				downloaded += int64(n)
				percentage := float64(downloaded) / float64(size) * 100

				// 发送进度事件
				c.SSEvent("progress", gin.H{
					"digest":     digest,
					"downloaded": downloaded,
					"total":      size,
					"percentage": percentage,
				})
				c.Writer.Flush()

				// 发送数据块
				c.SSEvent("data", base64.StdEncoding.EncodeToString(buffer[:n]))
				c.Writer.Flush()
			}

			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
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

// 查询下载进度的处理函数
func getBlobDownloadProgressHandler(c *gin.Context) {
	taskID := c.Query("taskId")

	tasksMutex.RLock()
	task, exists := downloadTasks[taskID]
	tasksMutex.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	task.mutex.RLock()
	defer task.mutex.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"taskId":   task.TaskID,
		"status":   task.Status,
		"progress": task.Progress,
		"error":    task.Error,
	})
}

// DockerRegistryClient 的新方法
func (client *DockerRegistryClient) downloadBlob(repo, tag, digest string, task *DownloadTask) error {
	httpClient, err := client.newAuthenticatedClient(repo, tag)
	if err != nil {
		return err
	}

	req, err := httpClient.newRequest("GET", fmt.Sprintf("/blobs/%s", digest), nil)
	if err != nil {
		return err
	}

	resp, err := httpClient.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download blob: %d", resp.StatusCode)
	}

	// 创建临时文件保存数据
	tmpDir := "./tmp"
	os.MkdirAll(tmpDir, 0755)
	fileName := filepath.Join(tmpDir, strings.Replace(digest, ":", "_", -1))

	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	// 使用io.TeeReader来同时写入文件并计算下载进度
	reader := io.TeeReader(resp.Body, &WriteCounter{
		Total: resp.ContentLength,
		OnProgress: func(current int64) {
			task.mutex.Lock()
			layerStatus := task.Progress[digest]
			layerStatus.Downloaded = current
			layerStatus.Percentage = float64(current) / float64(resp.ContentLength) * 100
			layerStatus.Status = "downloading"
			task.Progress[digest] = layerStatus
			task.mutex.Unlock()
		},
	})

	_, err = io.Copy(file, reader)
	if err != nil {
		return err
	}

	// 更新最终状态
	task.mutex.Lock()
	layerStatus := task.Progress[digest]
	layerStatus.Status = "completed"
	layerStatus.Downloaded = resp.ContentLength
	layerStatus.Percentage = 100
	task.Progress[digest] = layerStatus
	task.mutex.Unlock()

	return nil
}

// 用于跟踪写入进度的辅助结构
type WriteCounter struct {
	Total      int64
	OnProgress func(int64)
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	if wc.OnProgress != nil {
		wc.OnProgress(int64(n))
	}
	return n, nil
}

// DownloadTask 的辅助方法
func (task *DownloadTask) setError(err string) {
	task.mutex.Lock()
	task.Status = "failed"
	task.Error = err
	task.mutex.Unlock()
}

// 新增：HTTP 客户端结构体
type httpClient struct {
	client  *http.Client
	token   string
	baseURL string
}

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
