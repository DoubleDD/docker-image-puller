package minio

import (
	"context"
	"docker-image-handler/config"
	"fmt"
	"io"
	"log"
	"sync"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Tool struct {
	client *minio.Client
	ctx    context.Context
}

var (
	minioClientInstance *Tool
	minioClientOnce     sync.Once
)

func MinioClient() *Tool {
	minioClientOnce.Do(func() {
		load := config.Load()
		endpoint := load.Minio.Endpoint
		accessKeyID := load.Minio.Username
		secretAccessKey := load.Minio.Password

		// Initialize minio client object.
		client, err := minio.New(endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
			Secure: false,
		})
		if err != nil {
			log.Fatalln(err)
		}

		minioClientInstance = &Tool{
			client: client,
			ctx:    context.Background(),
		}
		log.Printf("%#v\n", minioClientInstance) // minioClient is now set up
	})

	return minioClientInstance
}

func (c *Tool) ListBuckets() ([]minio.BucketInfo, error) {
	buckets, err := c.client.ListBuckets(c.ctx)
	if err != nil {
		return nil, err
	}
	return buckets, nil
}

func (c *Tool) ListObjects(bucket, prefix string) ([]minio.ObjectInfo, error) {
	objectCh := c.client.ListObjects(c.ctx, bucket, minio.ListObjectsOptions{
		WithVersions: false,
		WithMetadata: false,
		Prefix:       prefix,
		Recursive:    false,
		MaxKeys:      0,
		StartAfter:   "",
		UseV1:        false,
	})
	var result []minio.ObjectInfo
	for object := range objectCh {
		if object.Err != nil {
			fmt.Println(object.Err)
			return nil, object.Err
		}
		result = append(result, object)
	}
	return result, nil
}

func (c *Tool) GetMetadata(bucket, object string) (minio.ObjectInfo, error) {
	return c.client.StatObject(c.ctx, bucket, object, minio.StatObjectOptions{})
}

func (c *Tool) UploadFile(bucket, object, fileName, fileContentType string, fileSize int64, fileReader io.Reader) (map[string]any, error) {
	uploadInfo, err := c.client.PutObject(
		c.ctx,
		bucket,
		object,
		fileReader,
		fileSize,
		minio.PutObjectOptions{ContentType: fileContentType},
	)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"message":    "file uploaded successfully",
		"bucket":     bucket,
		"object":     object,
		"file":       fileName,
		"uploadInfo": uploadInfo,
	}, nil
}
