package minio

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func (c *S3Client) ListBuckets() ([]Bucket, error) {
	// MinIO API要求使用斜杠结尾
	endpoint := strings.TrimSuffix(c.Endpoint, "/") + "/"
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	date := time.Now().UTC().Format("20060102T150405Z")
	req.Header.Set("x-amz-date", date)
	req.Header.Set("x-amz-content-sha256", emptyPayloadHash)
	req.Header.Set("Host", req.URL.Host)

	c.signRequest(req, date)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list buckets failed: status=%d body=%s", resp.StatusCode, string(body))
	}

	var result ListBucketsResult
	if err := xml.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}
	return result.Buckets.Bucket, nil
}

func (c *S3Client) ListObjects(prefix, delimiter string) (*ListObjectsResult, error) {
	url := fmt.Sprintf("%s/%s", strings.TrimSuffix(c.Endpoint, "/"), c.Bucket)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	if prefix != "" {
		q.Set("prefix", prefix)
	}
	if delimiter != "" {
		q.Set("delimiter", delimiter)
	}
	req.URL.RawQuery = q.Encode()

	date := time.Now().UTC().Format("20060102T150405Z")
	req.Header.Set("x-amz-date", date)
	req.Header.Set("x-amz-content-sha256", emptyPayloadHash)
	req.Header.Set("Host", req.URL.Host)

	c.signRequest(req, date)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list objects failed: status=%d body=%s", resp.StatusCode, string(body))
	}

	var result ListObjectsResult
	if err := xml.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *S3Client) PutObject(key string, data []byte) error {
	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/%s/%s", c.Endpoint, c.Bucket, key), bytes.NewReader(data))
	if err != nil {
		return err
	}

	date := time.Now().UTC().Format("20060102T150405Z")
	req.Header.Set("x-amz-date", date)
	req.Header.Set("x-amz-content-sha256", hashBytes(data))

	c.signRequest(req, date)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("put object failed: %s", resp.Status)
	}

	return nil
}

func (c *S3Client) GetObject(key string) ([]byte, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/%s/%s", c.Endpoint, c.Bucket, key), nil)
	if err != nil {
		return nil, err
	}

	date := time.Now().UTC().Format("20060102T150405Z")
	req.Header.Set("x-amz-date", date)
	req.Header.Set("x-amz-content-sha256", emptyPayloadHash)

	c.signRequest(req, date)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get object failed: %s", resp.Status)
	}

	var buf bytes.Buffer
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
