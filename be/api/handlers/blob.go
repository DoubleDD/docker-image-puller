package handlers

import (
	"docker-image-handler/pkg/registry"
	"docker-image-handler/pkg/utils"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ImageLayerBlobDownload 镜像文件下载
func ImageLayerBlobDownload(c *gin.Context) {
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

	reg, repo, tag, err := utils.ParseImageAddress(image)
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

	client := registry.NewDockerRegistryClient(reg, registry.WithAuth(username, password))

	// 开始下载
	err = client.DownloadBlobWithSSE(repo, tag, digest, size, c)
	if err != nil {
		c.SSEvent("error", err.Error())
		c.Writer.Flush()
		return
	}
}

// UploadBlobChunkInitiate 上传预检
func UploadBlobChunkInitiate(c *gin.Context) {
	chunkMd5 := c.Request.Header.Get("md5")
	digest := c.Request.Header.Get("digest")
	no := c.Request.Header.Get("no")
	fmt.Println("预检参数：", no, chunkMd5, digest)

	if chunkMd5 == "" {
		c.Status(http.StatusNoContent)
		return
	}
	// 根据md5校验文件是否已存在，如果存在，就不用上传，直接返回200
	tmpDir := filepath.Join(utils.UserHomeTmpDir(), strings.Replace(digest, ":", "_", -1))
	fileName := filepath.Join(tmpDir, "chunk-"+no)

	if utils.CheckFileMd5(fileName, chunkMd5) {
		// md5 一样，说明文件已存在，不用上传了
		c.Header("Connection", "Close")
		c.Status(http.StatusForbidden)
		return
	} else {
		// 不一样
		c.Status(http.StatusNoContent)
		return
	}
}

// UploadBlobChunk 新增：处理分块上传的处理器
func UploadBlobChunk(c *gin.Context) {
	digest := c.PostForm("digest")
	no := c.PostForm("no")
	chunkData := c.PostForm("chunk")

	// Base64解码
	data, err := base64.StdEncoding.DecodeString(chunkData)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid base64 data"})
		return
	}

	tmpDir := filepath.Join(utils.UserHomeTmpDir(), strings.Replace(digest, ":", "_", -1))
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create temp directory"})
		return
	}

	// 创建或追加到文件
	fileName := filepath.Join(tmpDir, "chunk-"+no)
	err = utils.CreateFileWithData(fileName, data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err, "msg": "Failed to save file"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}
