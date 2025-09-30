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

type Digest struct {
	MediaType string `json:"mediaType,omitempty"`
	Size      int64  `json:"size"`
	Digest    string `json:"digest"`
	Platform  struct {
		Architecture string `json:"architecture,omitempty"`
		Os           string `json:"os,omitempty"`
	} `json:"platform,omitempty"`
}

type Manifest struct {
	SchemaVersion int       `json:"schemaVersion"`
	MediaType     string    `json:"mediaType"`
	Config        *Digest   `json:"config,omitempty"`    // 可能为零值（空结构体）
	Layers        *[]Digest `json:"layers,omitempty"`    // 可能为nil或空切片
	Manifests     *[]Digest `json:"manifests,omitempty"` // 可能为nil或空切片
}

type MergeRequest struct {
	Manifest Manifest `json:"manifest"`
	Image    string   `json:"image"`
	Registry string   `json:"registry"`
}
