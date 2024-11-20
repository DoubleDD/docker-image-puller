package handlers

import (
	"docker-image-handler/pkg/k8s"
	"docker-image-handler/pkg/utils"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Rollout(c *gin.Context) {
	name := c.Query("name")
	ns := c.Query("ns")
	outStr := k8s.RolloutDeployment(name, ns)

	c.JSON(http.StatusOK, gin.H{"msg": outStr})
}

func Logs(c *gin.Context) {
	podName := c.Query("p")
	containerName := c.Query("c")
	ns := c.Query("ns")
	if podName == "" || ns == "" {
		c.String(http.StatusBadRequest, "ns, pod  parameters are required")
		return
	}

	// 转成 SSE 协议
	utils.HttpToSse(c)

	err := k8s.GetLogs(ns, podName, containerName, func(msg string) {
		c.SSEvent("message", msg)
		c.Writer.Flush()
	})
	if err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("Error opening stream: %v", err))
		return
	}
}

func Pods(c *gin.Context) {
	deployment := c.Query("dp")
	ns := c.Query("ns")
	pods, err := k8s.GetPodsForDeployment(ns, deployment)
	if err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("获取pod异常: %v", err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": "ok", "list": pods})
}
