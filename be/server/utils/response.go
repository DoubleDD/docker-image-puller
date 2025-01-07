package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code    int         `json:"code"`    // 状态码
	Message string      `json:"message"` // 描述信息
	Data    interface{} `json:"data"`    // 数据
}

// NewResponse 封装一个工具函数，便于返回统一格式
func NewResponse(c *gin.Context, code int, message string, data interface{}) {
	c.JSON(code, Response{
		Code:    code,
		Message: message,
		Data:    data,
	})
	c.Abort() // 确保请求不会继续处理
}
func Success(c *gin.Context, data interface{}) {
	NewResponse(c, http.StatusOK, "请求成功", data)
}

func Fail(c *gin.Context, code int, message string) {
	if code == 0 {
		code = http.StatusBadRequest
	}
	NewResponse(c, code, message, nil)
}
