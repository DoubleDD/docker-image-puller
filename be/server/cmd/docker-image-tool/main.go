package main

import (
	"docker-image-handler/api"
	"docker-image-handler/config"
	"docker-image-handler/pkg/linux"
	"log"
	"os"
)

func main() {
	if len(os.Args) == 1 {
		server()
	} else {
		switch os.Args[1] {
		case "install":
			//安装服务
			linux.InstallService()
		case "uninstall":
			//安装服务
			linux.UninstallService()
		default:

		}
	}
}
func server() {
	// 加载配置
	cfg := config.Load()

	// 初始化路由
	r := api.SetupRouter()

	// 启动服务
	log.Printf("Server starting on port %s", cfg.Port)
	err := r.Run(":" + cfg.Port)
	if err != nil {
		return
	}

}
