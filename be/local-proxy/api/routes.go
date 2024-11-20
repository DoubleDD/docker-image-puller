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

	// API路由组
	proxy := r.Group("proxy")
	{
		proxy.GET("/api/docker/manifest", handlers.GetManifest)
		proxy.GET("/api/docker/blob/download", handlers.ImageLayerBlobDownload)
	}
	return r
}
