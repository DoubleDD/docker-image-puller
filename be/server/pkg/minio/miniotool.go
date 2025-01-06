package minio

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// MinIOClient 是一个 MinIO 客户端
type MinIOClient struct {
	Endpoint  string // MinIO 服务器地址
	AccessKey string // Access Key
	SecretKey string // Secret Key
}

// NewMinIOClient 创建一个新的 MinIO 客户端
func NewMinIOClient(endpoint, accessKey, secretKey string) *MinIOClient {
	return &MinIOClient{
		Endpoint:  endpoint,
		AccessKey: accessKey,
		SecretKey: secretKey,
	}
}

// generateSignature 生成签名
func (c *MinIOClient) generateSignature(stringToSign string) string {
	h := hmac.New(sha256.New, []byte(c.SecretKey))
	h.Write([]byte(stringToSign))
	return hex.EncodeToString(h.Sum(nil))
}

// getAmzDate 获取当前时间（AWS 格式）
func (c *MinIOClient) getAmzDate() string {
	return time.Now().UTC().Format("20060102T150405Z")
}

// makeRequest 发送 HTTP 请求
func (c *MinIOClient) makeRequest(method, url string, body io.Reader, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	// 设置请求头
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	return client.Do(req)
}

// GetFileMetadata 获取文件的元数据
func (c *MinIOClient) GetFileMetadata(bucketName, objectName string) (map[string]string, error) {
	method := "HEAD"
	amzDate := c.getAmzDate()
	stringToSign := fmt.Sprintf("%s\n\n\n%s\n/%s/%s", method, amzDate, bucketName, objectName)
	signature := c.generateSignature(stringToSign)

	url := fmt.Sprintf("%s/%s/%s", c.Endpoint, bucketName, objectName)
	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s:%s", c.AccessKey, signature),
		"Date":          amzDate,
	}

	resp, err := c.makeRequest(method, url, nil, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to get file metadata: " + resp.Status)
	}

	metadata := make(map[string]string)
	for key, values := range resp.Header {
		if strings.HasPrefix(key, "X-Amz-Meta-") {
			metadata[key] = values[0]
		}
	}

	return metadata, nil
}

// ListFiles 列出指定目录下的文件列表
func (c *MinIOClient) ListFiles(bucketName, prefix string) ([]string, error) {
	method := "GET"
	amzDate := c.getAmzDate()
	stringToSign := fmt.Sprintf("%s\n\n\n%s\n/%s/?prefix=%s", method, amzDate, bucketName, prefix)
	signature := c.generateSignature(stringToSign)

	url := fmt.Sprintf("%s/%s/?prefix=%s", c.Endpoint, bucketName, prefix)
	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s:%s", c.AccessKey, signature),
		"Date":          amzDate,
	}

	resp, err := c.makeRequest(method, url, nil, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to list files: " + resp.Status)
	}

	var result struct {
		Contents []struct {
			Key string `json:"key"`
		} `json:"contents"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	files := make([]string, 0, len(result.Contents))
	for _, item := range result.Contents {
		files = append(files, item.Key)
	}

	return files, nil
}

// UploadFile 上传文件到 MinIO
func (c *MinIOClient) UploadFile(bucketName, objectName, filePath string) error {
	fileContent, err := io.ReadAll(bytes.NewReader([]byte("Hello, MinIO!")))
	if err != nil {
		return err
	}

	method := "PUT"
	contentType := "application/octet-stream"
	amzDate := c.getAmzDate()
	stringToSign := fmt.Sprintf("%s\n\n%s\n%s\n/%s/%s", method, contentType, amzDate, bucketName, objectName)
	signature := c.generateSignature(stringToSign)

	url := fmt.Sprintf("%s/%s/%s", c.Endpoint, bucketName, objectName)
	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s:%s", c.AccessKey, signature),
		"Date":          amzDate,
		"Content-Type":  contentType,
	}

	resp, err := c.makeRequest(method, url, bytes.NewReader(fileContent), headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("failed to upload file: " + resp.Status)
	}

	return nil
}

// ListBuckets 获取 Bucket 列表
func (c *MinIOClient) ListBuckets() ([]string, error) {
	method := "GET"
	amzDate := c.getAmzDate()
	stringToSign := fmt.Sprintf("%s\n\n\n%s\n/", method, amzDate)
	signature := c.generateSignature(stringToSign)

	url := fmt.Sprintf("%s/", c.Endpoint)
	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s:%s", c.AccessKey, signature),
		"Date":          amzDate,
	}

	resp, err := c.makeRequest(method, url, nil, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to list buckets: " + resp.Status)
	}

	var result struct {
		Buckets []struct {
			Name string `json:"name"`
		} `json:"buckets"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	buckets := make([]string, 0, len(result.Buckets))
	for _, bucket := range result.Buckets {
		buckets = append(buckets, bucket.Name)
	}

	return buckets, nil
}
