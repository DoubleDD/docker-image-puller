package utils

import (
	"fmt"
	"net/http"
	"strings"
)

// 新增：生成 curl 命令的辅助函数
func GenerateCurlCommand(req *http.Request) string {
	// 基础命令
	cmd := fmt.Sprintf("curl -X %s", req.Method)

	// 添加所有请求头
	for key, values := range req.Header {
		for _, value := range values {
			cmd += fmt.Sprintf(" \\\n  -H '%s: %s'", key, value)
		}
	}

	// 添加完整 URL
	fullURL := req.URL.String()
	if !strings.HasPrefix(fullURL, "http") {
		if strings.HasPrefix(fullURL, "/") {
			fullURL = fmt.Sprintf("%s://%s%s", req.URL.Scheme, req.URL.Host, fullURL)
		} else {
			fullURL = fmt.Sprintf("%s://%s/%s", req.URL.Scheme, req.URL.Host, fullURL)
		}
	}
	cmd += fmt.Sprintf(" \\\n  '%s'", fullURL)

	// 添加一些有用的 curl 选项
	cmd += " \\\n  -v" // 添加详细输出
	cmd += " \\\n  -L" // 跟随重定向

	return cmd
}
