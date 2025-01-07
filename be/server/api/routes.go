package api

import (
	"docker-image-handler/api/handlers"
	"docker-image-handler/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// 使用中间件
	r.Use(middleware.CORS())

	// 全局异常处理
	r.Use(middleware.RecoveryMiddleware())

	// API路由组
	dip := r.Group("dip")
	{
		dip.GET("/api/docker/images", handlers.GetDockerImages)
		dip.GET("/api/docker/pull", handlers.PullImage)
		dip.POST("/api/docker/images", handlers.PostDockerImages)
		dip.POST("/api/service/update", handlers.UpdateService)

		dip.GET("/minio/buckets", handlers.MinioBuckets)
		dip.GET("/minio/files", handlers.MinioFiles)
	}
	return r
}
