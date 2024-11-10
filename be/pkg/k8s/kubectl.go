package k8s

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// PodInfo 定义了 Pod 信息的结构体
type NamespaceImages struct {
	Namespace string
	Images    []string
}

func GetImages() map[string][]string {
	kubectl := "/Users/kedong/.local/bin/kubectl"
	jsonpath := "{range .items[*]}{.metadata.namespace}{\"\\t\"}{range .spec.containers[*]}{.image}{\",\"}{end}{\"\\n\"}{end} | sort | uniq"

	// 执行 kubectl 命令获取所有 Pod 的 JSON 数据
	cmd := exec.Command(kubectl, "get", "pods", "--all-namespaces", "-o", fmt.Sprintf("jsonpath=%s", jsonpath))
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		fmt.Printf("命令执行出错: %s\n", err)
		return nil
	}
	outputStr := out.String()

	// 创建一个 map 来存储结果
	result := make(map[string][]string)

	// 分割输出字符串为行
	lines := strings.Split(outputStr, "\n")

	for _, line := range lines {
		niPair := strings.Split(line, "\t")
		if len(niPair) == 2 {
			namespace := niPair[0]
			image := niPair[1]

			// 分割 images 字符串为切片
			images := strings.Split(image, ",")

			// 去除每个 image 字符串的前后空格
			for i := range images {
				images[i] = strings.TrimSpace(images[i])
			}

			// 将 images 添加到对应的 namespace 中，并去重
			for _, img := range images {
				if !contains(result[namespace], img) && img != "" {
					result[namespace] = append(result[namespace], img)
				}
			}
		}
	}
	return result
}

// 辅助函数：检查切片中是否包含某个元素
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
