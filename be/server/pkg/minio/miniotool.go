package minio

import (
	"context"
	"docker-image-handler/config"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"log"
	"sync"
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

func (c *Tool) ListObjects(bucket, prefix string) ([]string, error) {
	objectCh := c.client.ListObjects(c.ctx, bucket, minio.ListObjectsOptions{
		WithVersions: false,
		WithMetadata: false,
		Prefix:       prefix,
		Recursive:    false,
		MaxKeys:      0,
		StartAfter:   "",
		UseV1:        false,
	})
	var result []string
	for object := range objectCh {
		if object.Err != nil {
			fmt.Println(object.Err)
			return nil, object.Err
		}
		result = append(result, object.Key)
	}
	return result, nil
}
