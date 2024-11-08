package handlers

import (
	"docker-image-handler/pkg/k8s"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetDockerImages(c *gin.Context) {
	images := k8s.GetImages()
	c.JSON(http.StatusOK, images)
}
