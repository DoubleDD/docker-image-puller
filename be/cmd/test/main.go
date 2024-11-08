package main

import (
	"docker-image-handler/pkg/k8s"
)

func main() {
	k8s.GetImages()
	// utils.MergeFiles()
}
