package handlers

import (
	"docker-image-handler/pkg/registry"
	"docker-image-handler/pkg/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// 获取镜像 manifest 的 API 端点
func GetManifest(c *gin.Context) {
	image := c.Query("image")
	reg, repo, tag, err := utils.ParseImageAddress(image)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	username := c.Query("username")
	password := c.Query("password")

	client := registry.NewDockerRegistryClient(reg, registry.WithAuth(username, password))
	manifest, err := client.GetManifest(repo, tag)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, manifest)
}
