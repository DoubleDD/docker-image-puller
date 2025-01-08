package handlers

import (
	"docker-image-handler/pkg/minio"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func MinioBuckets(c *gin.Context) {
	// 初始化 MinIO 客户端
	client := minio.MinioClient()

	// 获取 Bucket 列表
	buckets, err := client.ListBuckets()
	if err != nil {
		fmt.Println("Error listing buckets:", err)
	} else {
		fmt.Println("Buckets:", buckets)
	}
	c.JSON(http.StatusOK, buckets)
}

func MinioFiles(c *gin.Context) {
	bucket := c.Query("bucket")
	prefix := c.Query("prefix")
	// 初始化 MinIO 客户端
	client := minio.MinioClient()

	// 获取 Bucket 列表
	result, err := client.ListObjects(bucket, prefix)
	if err != nil {
		fmt.Println("Error listing files:", err)
	}
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.JSON(http.StatusOK, result)
}

func MinioFileMetadata(c *gin.Context) {
	bucket := c.Query("bucket")
	object := c.Query("object")
	// 初始化 MinIO 客户端
	client := minio.MinioClient()

	// 获取 Bucket 列表
	result, err := client.GetMetadata(bucket, object)
	if err != nil {
		fmt.Println("Error listing files:", err)
	}
	c.JSON(http.StatusOK, result)
}

func MinioFileUpload(c *gin.Context) {
	bucket := c.Query("bucket")
	object := c.Query("object")
	// 获取上传的文件
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "file is required",
		})
		return
	}

	// 打印参数和文件信息
	log.Printf("Bucket: %s, Object: %s, File: %s", bucket, object, file.Filename)
	// 打开上传的文件
	fileReader, err := file.Open()
	if err != nil {
		log.Printf("Failed to open file: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to open file",
		})
		return
	}
	defer fileReader.Close()

	// 初始化 MinIO 客户端
	client := minio.MinioClient()

	result, err := client.UploadFile(bucket, object, file.Filename, file.Header.Get("Content-Type"), file.Size, fileReader)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to upload file to MinIO",
		})
		return
	}
	c.JSON(http.StatusOK, result)
}
