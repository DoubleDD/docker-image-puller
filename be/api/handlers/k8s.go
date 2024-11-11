package handlers

import (
	"docker-image-handler/pkg/k8s"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Rollout(c *gin.Context) {
	name := c.Query("name")
	ns := c.Query("ns")
	outStr := k8s.RolloutDeployment(name, ns)

	c.JSON(http.StatusOK, gin.H{"msg": outStr})
}
