package api

import (
	"docker-image-handler/api/handlers"
	"docker-image-handler/middleware"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// 自定义日志格式
	r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// 自定义日志格式
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
			param.ClientIP,
			param.TimeStamp.Format(time.RFC1123),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	}))

	// 使用中间件
	r.Use(middleware.CORS())

	// 全局异常处理
	r.Use(middleware.RecoveryMiddleware())

	// API路由组
	dip := r.Group("dip")
	{
		dip.GET("/api/dp/logs", handlers.Logs)
		dip.GET("/api/dp/containers", handlers.Containers)
		dip.GET("/api/dp/pods", handlers.Pods)
		dip.GET("/api/dp/rollout", handlers.Rollout)
		dip.GET("/api/docker/images", handlers.GetDockerImages)
		dip.GET("/api/docker/blob/chunk/pre", handlers.UploadBlobChunkInitiate)
		dip.POST("/api/docker/blob/chunk", handlers.UploadBlobChunk)
		dip.POST("/api/docker/merge", handlers.MergeImage)
		dip.GET("/api/docker/pull", handlers.PullImage)
		dip.POST("/api/docker/images", handlers.PostDockerImages)
	}
	return r
}
