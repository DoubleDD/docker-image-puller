package main

import (
	"docker-image-handler/pkg/k8s"
	"encoding/json"
	"fmt"
)

func main() {
	getLogs()

	// utils.MergeFiles()
	// k8s.RolloutDeployment("chat2db", "")
}

func getLogs() {
	k8s.GetLogs("default", "chat2db-c49957b9-jwt9p", "", func(msg string) {
		fmt.Println(msg)
	})
}

func getImages() {
	ns, _ := k8s.GetImages("")
	// 使用 json.MarshalIndent 格式化 JSON
	formattedJSON, err := json.MarshalIndent(ns, "", "    ")
	if err != nil {
		fmt.Println("Error formatting JSON:", err)
		return
	}

	// 打印格式化后的 JSON
	fmt.Println(string(formattedJSON))
}
