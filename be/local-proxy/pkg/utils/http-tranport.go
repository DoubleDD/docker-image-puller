package utils

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// LoggingRoundTripper 自定义的 RoundTripper，用于拦截请求和响应
type LoggingRoundTripper struct {
	Transport http.RoundTripper
}

// RoundTrip 实现 RoundTripper 接口的 RoundTrip 方法
func (lrt *LoggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()

	// 打印请求日志
	log.Println("=====> Starting request")
	curl := GenerateCurlCommand(req)
	fmt.Println(curl)
	fmt.Println("======================================================================")

	// 打印请求行和请求头
	fmt.Printf("%s %s %s\n", req.Method, req.URL.Scheme+"://"+req.URL.Host+req.URL.Path+"?"+req.URL.RawQuery, req.Proto)
	for name, values := range req.Header {
		for _, value := range values {
			fmt.Printf("%s: %s\n", name, value)
		}
	}
	fmt.Println()

	// 调用原始 Transport（如果没有则使用默认 Transport）
	transport := lrt.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	// 发送请求
	resp, err := transport.RoundTrip(req)

	// 打印响应行和响应头
	if err != nil {
		log.Printf("<===== Request failed: %v", err)
	} else {
		fmt.Printf("%s %d %s\n", resp.Proto, resp.StatusCode, http.StatusText(resp.StatusCode))
		for name, values := range resp.Header {
			for _, value := range values {
				fmt.Printf("%s: %s\n", name, value)
			}
		}
		fmt.Println("======================================================================")

		log.Printf("<===== Completed request in %v\n", time.Since(start))
	}

	return resp, err
}

func HttpToSse(c *gin.Context) {
	// 设置SSE响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")
}
