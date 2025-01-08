package handlers

import (
	"docker-image-handler/pkg/docker"
	"docker-image-handler/utils"
	"fmt"
	"io"
	"log"
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

// 定义消息结构
type Message struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

func PullImage(c *gin.Context) {
	oldImage := c.Query("oldImage")
	newImage := c.Query("newImage")
	utils.HttpToSse(c)

	// 创建一个通道用于发送消息
	messageChan := make(chan Message)

	// 创建一个 done channel 用于同步
	done := make(chan struct{})

	go func() {
		// 确保在 goroutine 结束时发送完成信号
		defer close(done)

		err := docker.PullImage(
			oldImage,
			newImage,
			func(msg string) {
				messageChan <- Message{
					Event: "message",
					Data:  msg,
				}
			},
			func() {
				messageChan <- Message{
					Event: "done",
					Data:  "Done!",
				}
			},
		)

		if err != nil {
			log.Print(err)
		}
	}()

	// 在主函数结束时等待 goroutine 完成并关闭 messageChan
	defer func() {
		<-done // 等待 goroutine 完成
		close(messageChan)
	}()

	c.Stream(func(w io.Writer) bool {
		msg, ok := <-messageChan
		if !ok {
			return false
		}

		// data, _ := json.Marshal(msg.Data)
		// 直接写入 writer
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", msg.Event, msg.Data)

		return msg.Event != "done"
	})
}

func PostDockerImages(c *gin.Context) {
	ns := c.PostForm("ns")
	name := c.PostForm("name")
	oldImage := c.PostForm("oldImage")
	newImage := c.PostForm("newImage")
	if ns == "" {
		ns = "Default"
	}

	err := docker.AddImage(ns, name, oldImage, newImage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": "ok"})
}
