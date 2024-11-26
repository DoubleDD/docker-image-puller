package handlers

import (
	"docker-image-handler/pkg/k8s"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetDockerImages(c *gin.Context) {
	ns := c.Query("namespace")
	images, err := k8s.GetImagesNew(ns)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, images)
}
