package utils

import (
	"docker-image-handler/config"
	"fmt"
	"strings"
)

// 新增：解析镜像地址的函数
func ParseImageAddress(imageAddress string) (registry, repository, tag string, err error) {
	cfg := config.Load()

	// 移除可能存在的协议前缀
	imageAddress = strings.TrimPrefix(imageAddress, "http://")
	imageAddress = strings.TrimPrefix(imageAddress, "https://")

	// 分离tag
	parts := strings.Split(imageAddress, ":")
	if len(parts) > 2 {
		return "", "", "", fmt.Errorf("invalid image address format")
	}

	address := parts[0]
	if len(parts) == 2 {
		tag = parts[1]
	} else {
		tag = "latest"
	}

	// 分离registry和repository
	addressParts := strings.Split(address, "/")
	if len(addressParts) < 2 {
		// 默认使用Docker Hub
		registry = cfg.DefaultPullRegistry
		repository = imageAddress
	} else {
		// 判断第一部分是否包含域名特征（包含'.'或':'）
		if strings.Contains(addressParts[0], ".") || strings.Contains(addressParts[0], ":") {
			registry = addressParts[0]
			repository = strings.Join(addressParts[1:], "/")
		} else {
			registry = cfg.DefaultPullRegistry
			repository = imageAddress
		}
	}

	return registry, repository, tag, nil
}
