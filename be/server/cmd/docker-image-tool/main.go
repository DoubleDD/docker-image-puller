package main

import (
	"docker-image-handler/api"
	"docker-image-handler/config"
	"docker-image-handler/utils"
	"log"
	"net/http"
)

func main() {
	// 加载配置
	cfg := config.Load()

	// 设置 DefaultClient 的 Transport 为我们的 LoggingRoundTripper
	http.DefaultClient.Transport = &utils.LoggingRoundTripper{}

	// 初始化路由
	r := api.SetupRouter()

	// 启动服务
	log.Printf("Server starting on port %s", cfg.Port)
	err := r.Run(":" + cfg.Port)
	if err != nil {
		return
	}

}
