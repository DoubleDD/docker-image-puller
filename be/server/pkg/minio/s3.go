package minio

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

type S3Client struct {
	AccessKey string
	SecretKey string
	Region    string
	Bucket    string
	Endpoint  string
}
type Bucket struct {
	Name         string `xml:"Name" json:"name"`
	CreationDate string `xml:"CreationDate" json:"creation_date"`
}
type ListBucketsResult struct {
	XMLName xml.Name `xml:"ListAllMyBucketsResult"`
	Buckets struct {
		Bucket []Bucket `xml:"Bucket"`
	} `xml:"Buckets"`
}

type ListObjectsResult struct {
	XMLName   xml.Name `xml:"ListBucketResult"`
	Name      string   `xml:"Name"`
	Prefix    string   `xml:"Prefix"`
	Delimiter string   `xml:"Delimiter"`
	Contents  []struct {
		Key          string    `xml:"Key"`
		LastModified time.Time `xml:"LastModified"`
		Size         int64     `xml:"Size"`
	} `xml:"Contents"`
	CommonPrefixes []struct {
		Prefix string `xml:"Prefix"`
	} `xml:"CommonPrefixes"`
}

func NewS3Client(endpoint, accessKey, secretKey, region, bucket string) *S3Client {
	return &S3Client{
		AccessKey: accessKey,
		SecretKey: secretKey,
		Region:    region,
		Bucket:    bucket,
		Endpoint:  endpoint,
	}
}

func (c *S3Client) signRequest(req *http.Request, date string) {
	scope := fmt.Sprintf("%s/%s/s3/aws4_request", date[:8], c.Region)
	credential := fmt.Sprintf("%s/%s", c.AccessKey, scope)

	headers := make(map[string]string)
	for k, v := range req.Header {
		headers[strings.ToLower(k)] = strings.TrimSpace(v[0])
	}

	signedHeaders := make([]string, 0)
	for k := range headers {
		signedHeaders = append(signedHeaders, k)
	}
	sort.Strings(signedHeaders)

	// 规范化查询字符串
	query := req.URL.Query()
	queryKeys := make([]string, 0)
	for k := range query {
		queryKeys = append(queryKeys, k)
	}
	sort.Strings(queryKeys)

	canonicalQuery := ""
	for i, k := range queryKeys {
		if i > 0 {
			canonicalQuery += "&"
		}
		canonicalQuery += fmt.Sprintf("%s=%s", k, query.Get(k))
	}

	// 添加查询参数到规范请求
	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n",
		req.Method,
		req.URL.Path,
		canonicalQuery)

	for _, k := range signedHeaders {
		canonicalRequest += fmt.Sprintf("%s:%s\n", k, headers[k])
	}
	canonicalRequest += "\n" + strings.Join(signedHeaders, ";") + "\n" + headers["x-amz-content-sha256"]

	stringToSign := fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s\n%s",
		date,
		scope,
		hashString(canonicalRequest))

	signingKey := hmacSHA256([]byte("AWS4"+c.SecretKey), []byte(date[:8]))
	signingKey = hmacSHA256(signingKey, []byte(c.Region))
	signingKey = hmacSHA256(signingKey, []byte("s3"))
	signingKey = hmacSHA256(signingKey, []byte("aws4_request"))

	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	auth := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s,SignedHeaders=%s,Signature=%s",
		credential,
		strings.Join(signedHeaders, ";"),
		signature)

	req.Header.Set("Authorization", auth)
}

const emptyPayloadHash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

func hashBytes(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func hashString(s string) string {
	return hashBytes([]byte(s))
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}
