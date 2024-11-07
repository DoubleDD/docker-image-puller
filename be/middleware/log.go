package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// 自定义日志中间件
func CustomLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 记录请求开始时间
		start := time.Now()

		// 处理请求
		c.Next()

		// 记录请求结束时间
		end := time.Now()
		latency := end.Sub(start)

		// 获取请求信息
		clientIP := c.ClientIP()
		method := c.Request.Method
		path := c.Request.URL.Path
		statusCode := c.Writer.Status()

		// 记录日志
		fmt.Printf("[GIN] %v | %3d | %13v | %15s | %-7s %s\n",
			end.Format("2006/01/02 - 15:04:05"),
			statusCode,
			latency,
			clientIP,
			method,
			path,
		)
	}
}
