package handlers

import (
	"docker-image-handler/pkg/minio"
	"fmt"
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
	} else {
		fmt.Println("files:", result)
	}
	c.JSON(http.StatusOK, result)
}
