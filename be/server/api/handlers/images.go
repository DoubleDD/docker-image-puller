package handlers

import (
	"docker-image-handler/pkg/docker"
	"docker-image-handler/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetDockerImages(c *gin.Context) {
	images, err := docker.GetImages()
	if err != nil {
		c.JSON(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, images)
}

func PullImage(c *gin.Context) {
	oldImage := c.Query("oldImage")
	newImage := c.Query("newImage")
	utils.HttpToSse(c)
	err := docker.PullImage(
		oldImage,
		newImage,
		func(msg string) {
			c.SSEvent("message", msg)
			c.Writer.Flush()
		},
		func() {
			c.SSEvent("done", "Done!")
			c.Writer.Flush()
		},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": "ok"})
}
