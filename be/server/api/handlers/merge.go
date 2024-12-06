package handlers

import (
	"archive/tar"
	"docker-image-handler/internal/models"
	"docker-image-handler/pkg/docker"
	"docker-image-handler/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func MergeImage(c *gin.Context) {
	var req models.MergeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// 创建临时目录用于存放最终的镜像文件
	tmpDir := utils.UserHomeTmpDir()
	outputFile := filepath.Join(tmpDir, fmt.Sprintf("%s_%s.tar", req.Image, time.Now().Format("2006-01-02_15:04:05")))

	// 创建tar文件
	tf, err := utils.CreateFile(outputFile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "Failed to create output file", "error": err})
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

	// 合并配置文件
	configDir := filepath.Join(tmpDir, strings.Replace(req.Manifest.Config.Digest, ":", "_", -1))
	configFile := filepath.Join(configDir, "all")
	utils.MergeFiles(configDir, configFile)

	// 写入配置文件
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
		// 合并层文件
		layerSourceDir := filepath.Join(tmpDir, strings.Replace(layer.Digest, ":", "_", -1))
		layerFile := filepath.Join(layerSourceDir, "all")
		utils.MergeFiles(layerSourceDir, layerFile)

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
	// defer func() {
	// 	// 删除配置文件
	// 	configFile := filepath.Join(tmpDir, strings.Replace(req.Manifest.Config.Digest, ":", "_", -1))
	// 	os.RemoveAll(configFile)
	//
	// 	// 删除每一层的临时文件
	// 	for _, layer := range req.Manifest.Layers {
	// 		layerFile := filepath.Join(tmpDir, strings.Replace(layer.Digest, ":", "_", -1))
	// 		os.RemoveAll(layerFile)
	// 	}
	//
	// }()

	// 新开一个协程用来执行 docker tag、docker push等操作
	err = docker.PushImage(outputFile, req.Image, req.Registry, req.Push)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "合并镜像失败", "error": err})
		return
	}

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
