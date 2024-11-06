package main

import (
	"docker-image-handler/api"
	"docker-image-handler/config"
	"log"
)

func main() {
	// 加载配置
	cfg := config.Load()

	// 初始化路由
	r := api.SetupRouter()

	// 启动服务
	log.Printf("Server starting on port %s", cfg.Port)
	r.Run(":" + cfg.Port)

}
