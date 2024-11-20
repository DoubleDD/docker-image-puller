package registry

import (
	"net/http"
	"sync"
	"time"
)

// DockerRegistryClient 是用于与 Docker Registry API 交互的客户端结构体
type DockerRegistryClient struct {
	Username string
	Password string
	Registry string
	jwtCache map[string]jwtCacheItem // 新增：JWT缓存
	mutex    sync.RWMutex            // 新增：用于并发安全
}

type jwtCacheItem struct {
	token      string
	expireTime time.Time
}

type jwtPayload struct {
	Exp int64 `json:"exp"`
}

// TokenResponse 新增：获取registry认证token的响应结构
type TokenResponse struct {
	Token string `json:"token"`
}

// 新增：HTTP 客户端结构体
type httpClient struct {
	client  *http.Client
	token   string
	baseURL string
}
