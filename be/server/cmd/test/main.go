package main

import (
	"docker-image-handler/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"log"
)

func main() {
	minioClient()
}
func minioClient() *minio.Client {
	load := config.Load()
	endpoint := load.Minio.Endpoint
	accessKeyID := load.Minio.Username
	secretAccessKey := load.Minio.Password

	// Initialize minio client object.
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: false,
	})
	if err != nil {
		log.Fatalln(err)
		return nil
	}

	log.Printf("%#v\n", minioClient) // minioClient is now set up
	return minioClient
}
