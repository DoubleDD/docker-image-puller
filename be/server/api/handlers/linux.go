package handlers

import (
	"docker-image-handler/pkg/linux"

	"github.com/gin-gonic/gin"
)

func UpdateService(c *gin.Context) {
	linux.UpdateService()
}
