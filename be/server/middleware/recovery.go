package middleware

import (
	"docker-image-handler/utils"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 打印错误日志（可以根据需要自定义日志记录方式）
				fmt.Printf("Panic: %v\n", err)

				// 返回统一的错误响应
				utils.NewResponse(c, http.StatusInternalServerError, "服务器内部错误", nil)
			}
		}()
		c.Next()
	}
}
