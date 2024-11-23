package handlers

import (
	"docker-image-handler/pkg/registry"
	"docker-image-handler/pkg/utils"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func Status(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// GetManifest 获取镜像 manifest 的 API 端点
func GetManifest(c *gin.Context) {
	image := c.Query("image")
	reg, repo, tag, err := utils.ParseImageAddress(image)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	username := c.Query("username")
	password := c.Query("password")

	client := registry.NewDockerRegistryClient(reg, registry.WithAuth(username, password))
	manifest, err := client.GetManifest(repo, tag)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, manifest)
}

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

	// 转成 SSE 协议
	utils.HttpToSse(c)

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
