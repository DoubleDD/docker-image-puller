package main

import (
	"archive/tar"
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

	// 使用 CORS 中间件
	r.Use(CORSMiddleware())

	dip := r.Group("dip")
	{
		dip.GET("/api/docker/manifest", getManifestHandler)
		dip.GET("/api/docker/blob/download", startBlobDownloadHandler) // 镜像层数据下载接口
		dip.POST("/api/docker/blob/chunk", uploadBlobChunkHandler)     // 镜像层数据分块上传接口
		dip.POST("/api/docker/merge", mergeImageHandler)               // 新增合并接口
	}

	r.Run(":7152") // 启动服务
}

// 添加 CORS 中间件
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
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
	Image    string   `json:"image"`
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
}

type LayerStatus struct {
	Size       int64   `json:"size"`
	Downloaded int64   `json:"downloaded"`
	Percentage float64 `json:"percentage"`
	Status     string  `json:"status"`
}

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

// 新增：在 DockerRegistryClient 中添���创建认证客户端的方法
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

// 新增：处理分块上传的处理器
func uploadBlobChunkHandler(c *gin.Context) {
	digest := c.PostForm("digest")
	chunkData := c.PostForm("chunk")

	// Base64解码
	data, err := base64.StdEncoding.DecodeString(chunkData)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid base64 data"})
		return
	}

	// 确保临时目录存在
	tmpDir := "./tmp"
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create temp directory"})
		return
	}

	// 创建或追加到文件
	fileName := filepath.Join(tmpDir, strings.Replace(digest, ":", "_", -1))
	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file"})
		return
	}
	defer file.Close()

	// 写入数据
	if _, err := file.Write(data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func mergeImageHandler(c *gin.Context) {
	var req MergeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// 创建临时目录用于存放最终的镜像文件
	tmpDir := "./tmp"
	outputFile := filepath.Join(tmpDir, fmt.Sprintf("image_%d.tar", time.Now().UnixNano()))

	// 创建tar文件
	tf, err := os.Create(outputFile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create output file"})
		return
	}
	defer tf.Close()

	tw := tar.NewWriter(tf)
	defer tw.Close()

	// 写入manifest.json
	manifestList := []map[string]interface{}{
		{
			"Config":   req.Manifest.Config.Digest[7:] + ".json", // 移除 "sha256:" 前缀
			"RepoTags": []string{req.Image},
			"Layers":   make([]string, len(req.Manifest.Layers)),
		},
	}

	// 准备层文件名列表
	for i, layer := range req.Manifest.Layers {
		manifestList[0]["Layers"].([]string)[i] = layer.Digest[7:] + "/layer.tar"
	}

	// 写入 manifest.json
	manifestJson, err := json.Marshal(manifestList)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal manifest"})
		return
	}

	err = addFileToTar(tw, "manifest.json", manifestJson)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add manifest to tar"})
		return
	}

	// 写入配置文件
	configFile := filepath.Join(tmpDir, strings.Replace(req.Manifest.Config.Digest, ":", "_", -1))
	configData, err := os.ReadFile(configFile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read config file"})
		return
	}

	err = addFileToTar(tw, req.Manifest.Config.Digest[7:]+".json", configData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add config to tar"})
		return
	}

	// 处理每一层
	for _, layer := range req.Manifest.Layers {
		layerFile := filepath.Join(tmpDir, strings.Replace(layer.Digest, ":", "_", -1))

		// 创建层目录
		layerDir := layer.Digest[7:]

		// 添加层数据
		layerData, err := os.ReadFile(layerFile)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to read layer %s", layer.Digest)})
			return
		}

		err = addFileToTar(tw, filepath.Join(layerDir, "layer.tar"), layerData)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to add layer %s to tar", layer.Digest)})
			return
		}
	}

	// 完成tar文件写入
	if err := tw.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to finalize tar file"})
		return
	}

	// 延迟删除临时文件
	defer func() {
		// 删除配置文件
		configFile := filepath.Join(tmpDir, strings.Replace(req.Manifest.Config.Digest, ":", "_", -1))
		os.Remove(configFile)

		// 删除每一层的临时文件
		for _, layer := range req.Manifest.Layers {
			layerFile := filepath.Join(tmpDir, strings.Replace(layer.Digest, ":", "_", -1))
			os.Remove(layerFile)
		}

	}()

	c.JSON(http.StatusOK, gin.H{"ok": "Done!"})
}

// 辅助函数：添加文件到tar
func addFileToTar(tw *tar.Writer, name string, data []byte) error {
	header := &tar.Header{
		Name:    name,
		Mode:    0644,
		Size:    int64(len(data)),
		ModTime: time.Now(),
	}

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	if _, err := tw.Write(data); err != nil {
		return err
	}

	return nil
}
