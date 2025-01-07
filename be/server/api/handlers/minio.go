package handlers

import (
	"docker-image-handler/config"
	"docker-image-handler/pkg/minio"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func MinioBuckets(c *gin.Context) {
	minioConfig := config.Load().Minio
	// 初始化 MinIO 客户端
	client := minio.NewS3Client(minioConfig.Endpoint, minioConfig.Username, minioConfig.Password, "", "")

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
	minioConfig := config.Load().Minio
	// 初始化 MinIO 客户端
	client := minio.NewS3Client(minioConfig.Endpoint, minioConfig.Username, minioConfig.Password, "", bucket)

	// 获取 Bucket 列表
	result, err := client.ListObjects(prefix, "/")
	if err != nil {
		fmt.Println("Error listing files:", err)
	} else {
		fmt.Println("files:", result)
	}
	c.JSON(http.StatusOK, result)
}
