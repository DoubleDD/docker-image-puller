package api

import (
	"docker-image-handler/api/handlers"
	"docker-image-handler/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	// 使用中间件
	r.Use(middleware.CORS())

	// API路由组
	dip := r.Group("dip")
	{
		dip.GET("/api/docker/manifest", handlers.GetManifest)
		dip.GET("/api/docker/images", handlers.GetDockerImages)
		dip.GET("/api/docker/blob/download", handlers.ImageLayerBlobDownload)
		dip.GET("/api/docker/blob/chunk/pre", handlers.UploadBlobChunkInitiate)
		dip.POST("/api/docker/blob/chunk", handlers.UploadBlobChunk)
		dip.POST("/api/docker/merge", handlers.MergeImage)
	}
	return r
}
