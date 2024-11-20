package handlers

import (
	"docker-image-handler/pkg/utils"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

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
