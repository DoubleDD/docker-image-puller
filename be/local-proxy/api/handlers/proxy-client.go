package handlers

import (
	"docker-image-handler/config"
	"docker-image-handler/pkg/registry"
	"docker-image-handler/pkg/utils"
	"fmt"
	"net/http"
	"path/filepath"
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
	reg, ns, repo, tag, err := utils.ParseImageAddress(image)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	platform := c.Query("platform")
	username := c.Query("username")
	password := c.Query("password")
	if username == "" {
		cfg := config.Load()
		username = cfg.DefaultUsername
		password = cfg.DefaultPassword
	}

	client := registry.NewDockerRegistryClient(reg, registry.WithAuth(username, password))
	manifest, err := client.GetManifest(ns+"/"+repo, tag, platform)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, manifest)
}

// ImageLayerBlobDownload 镜像文件下载
func ImageLayerBlobDownload(c *gin.Context) {
	dataType := c.Query("type")
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
	if dataType == "" {
		dataType = ".tar"
	}
	reg, ns, repo, tag, err := utils.ParseImageAddress(image)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	username := c.Query("username")
	password := c.Query("password")
	if username == "" {
		cfg := config.Load()
		username = cfg.DefaultUsername
		password = cfg.DefaultPassword
	}
	// 创建下载任务
	taskID := fmt.Sprintf("%d", time.Now().UnixNano())

	// 转成 SSE 协议
	utils.HttpToSse(c)

	// 发送任务ID
	c.SSEvent("taskId", taskID)
	c.Writer.Flush()

	client := registry.NewDockerRegistryClient(reg, registry.WithAuth(username, password))

	// 开始下载
	_, err = client.DownloadBlobWithSSE(ns, repo, tag, digest, dataType, size, c)
	if err != nil {
		c.SSEvent("error", err.Error())
		c.Writer.Flush()
		return
	}
}

func MergeImageLayers(c *gin.Context) {
	image := c.Query("image")
	// imageLayersPath := filepath.Join(utils.UserHomeTmpDir(), image)
	// imageLayersPath 是镜像所有层的文件内容，层文件名为：{digest}/all。现在将所有的层合并层一个完整的离线镜像文件，合并完后的效果应该和"docker save"命令的效果是一样的，可以使用 docker load 命令加载到docker引擎中，也可以直接使用skopeo工具将其推送到私有仓库中

	// 获取镜像的manifest
	reg, ns, repo, tag, err := utils.ParseImageAddress(image)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	username := c.Query("username")
	password := c.Query("password")
	if username == "" {
		cfg := config.Load()
		username = cfg.DefaultUsername
		password = cfg.DefaultPassword
	}

	// 转成 SSE 协议
	utils.HttpToSse(c)

	client := registry.NewDockerRegistryClient(reg, registry.WithAuth(username, password))
	manifest, err := client.GetManifest(ns+"/"+repo, tag, "")
	if err != nil {
		c.SSEvent("error", err.Error())
		c.Writer.Flush()
		return
	}

	c.SSEvent("manifest", manifest)
	c.Writer.Flush()

	// 下载镜像的层内容
	if manifest.Layers != nil && len(*manifest.Layers) > 0 {
		// layers 文件列表
		layerFileMap := make(map[string]string)
		for i, layer := range *manifest.Layers {
			fmt.Printf("下载镜像层：%d  %s\n", i, image)
			filename, err := client.DownloadBlobWithSSE(ns, repo, tag, layer.Digest, ".tar", layer.Size, c)
			if err != nil {
				c.SSEvent("error", gin.H{
					"message": "下载镜像层失败",
					"error":   err,
				})
				c.Writer.Flush()
				return
			} else {
				layerFileMap[layer.Digest] = filename
			}
		}
		// 下载config的内容
		filename, err := client.DownloadBlobWithSSE(ns, repo, tag, manifest.Config.Digest, ".json", manifest.Config.Size, c)
		if err != nil {
			c.SSEvent("error", gin.H{
				"message": "下载镜像Config失败",
				"error":   err,
			})
			c.Writer.Flush()
			return
		} else {
			layerFileMap[manifest.Config.Digest] = filename
		}

		// 镜像输出目录
		dest := filepath.Join(utils.UserHomeTmpDir(), repo+"_"+tag)
		// 将层文件合并成完整的镜像文件
		msg, err := registry.MergeImageLayers(image, manifest, layerFileMap, dest)
		if err != nil {
			c.SSEvent("error", gin.H{
				"message": msg,
				"error":   err,
			})
			c.Writer.Flush()
			return
		}
		c.SSEvent("done", "Done!")
		c.Writer.Flush()
	}
}
