package k8s

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type NamespaceImages struct {
	Namespace   string
	Deployments []Deployment
}
type Deployment struct {
	ImageName string
	Name      string
}

func GetImagesBack(namespace string) ([]NamespaceImages, error) {
	var result []NamespaceImages
	var nsList []string
	if namespace != "" {
		nsList = append(nsList, namespace)
	} else {
		// 获取namespace
		outStr, _, err := ExecCmd("kubectl", "get", "namespaces", "-o", fmt.Sprintf("jsonpath='%s'", `{range .items[*]}{.metadata.name}{"\n"}{end}`))
		if err != nil {
			fmt.Printf("命令执行出错: %s\n", err)
			return result, err
		}
		nsList = strings.Split(outStr, "\n")
	}
	for _, ns := range nsList {
		ns = strings.TrimSpace(ns)
		if ns == "" {
			continue
		}
		// 1、获取命名空间下所有的deployment
		outStr, _, err := ExecCmd("kubectl", "get", "deployment", "-n", ns, "-o", fmt.Sprintf("jsonpath='%s'", `{range .items[*]}{.metadata.name}{"\n"}{end}`))
		if err != nil {
			fmt.Printf("命令执行出错: %s\n", err)
			return result, err
		}
		deployments := strings.Split(outStr, "\n")

		var deploymentList []Deployment
		jsonpath := `{range .spec.template.spec.%s[*]}{.image}{"\n"}{end}`
		for _, deployment := range deployments {
			deployment = strings.TrimSpace(deployment)
			if deployment == "" {
				continue
			}

			// 2、获取deployment下的initContainers的镜像地址
			imagesStr, _, err := ExecCmd("kubectl", "get", "deployment", deployment, "-n", ns, "-o", fmt.Sprintf("jsonpath='%s'", fmt.Sprintf(jsonpath, "initContainers")))
			if err != nil {
				fmt.Printf("获取deployment下的initContainers的镜像地址,命令执行出错: %s\n", err)
				continue
			}
			if imagesStr == "" {
				containsStr, _, err := ExecCmd("kubectl", "get", "deployment", deployment, "-n", ns, "-o", fmt.Sprintf("jsonpath='%s'", fmt.Sprintf(jsonpath, "containers")))
				if err != nil {
					fmt.Printf("获取deployment下的initContainers的镜像地址,命令执行出错: %s\n", err)
					continue
				}
				imagesStr = containsStr
			}

			if imagesStr == "" {
				continue
			}
			deploymentList = append(deploymentList, Deployment{
				Name:      deployment,
				ImageName: imagesStr,
			})
		}
		if len(deploymentList) > 0 {
			result = append(result, NamespaceImages{
				Namespace:   ns,
				Deployments: deploymentList,
			})
		}
	}
	return result, nil
}

// 滚动更新服务
func RolloutDeploymentBackup(name string, namespace string) string {
	if namespace == "" {
		namespace = "default"
	}
	out, _, err := ExecCmd("kubectl", "rollout", "restart", "deployment", name, "-n", namespace)
	if err != nil {
		return err.Error()
	}
	fmt.Println(out)
	return out
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

// 执行命令
func ExecCmd(name string, args ...string) (string, string, error) {
	cmd := exec.Command(name, args...)
	// 继承当前环境变量
	cmd.Env = append(os.Environ(), "PATH="+os.Getenv("PATH"))
	fmt.Println(cmd)

	var out bytes.Buffer
	var stderror bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderror
	err := cmd.Run()
	if err != nil {
		fmt.Printf("命令执行出错: %s\n", err)
		return "", "", err
	}
	outputStr := out.String()
	outputStr = outputStr[1 : len(outputStr)-1]
	outputStr = strings.TrimSpace(outputStr)
	fmt.Println(outputStr)
	return outputStr, stderror.String(), nil
}
