package docker

import (
	"bufio"
	"docker-image-handler/config"
	"docker-image-handler/pkg/k8s"
	"docker-image-handler/utils"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

func PushImage(tarFile, image, registry string, push bool) error {
	cfg := config.Load()

	fmt.Printf("开始处理镜像推送任务...\n")
	fmt.Printf("镜像文件: %s\n镜像名称: %s\n目标仓库: %s\n", tarFile, image, registry)

	if registry == "" {
		registry = cfg.DefaultPushRegistry
		fmt.Printf("使用默认镜像仓库: %s\n", registry)
	}

	fmt.Printf("步骤1: 加载镜像文件...\n")
	loadCmd := exec.Command("docker", "load", "-i", tarFile)
	if err := loadCmd.Run(); err != nil {
		fmt.Printf("❌ 镜像加载失败: %v\n", err)
		return err
	}
	fmt.Printf("✅ 镜像加载成功\n")

	_, repository, tag, _ := utils.ParseImageAddress(image)
	newTag := fmt.Sprintf("%s/%s:%s", registry, repository, tag)
	fmt.Printf("步骤2: 标记镜像\n [%s] -> [%s]...\n", image, newTag)

	tagCmd := exec.Command("docker", "tag", image, newTag)
	if err := tagCmd.Run(); err != nil {
		fmt.Printf("❌ 镜像标记失败: %v\n", err)
		return err
	}
	fmt.Printf("✅ 镜像标记成功\n")

	defer func() {
		fmt.Printf("步骤4: 清理临时文件 [%s]...\n", tarFile)
		if err := os.Remove(tarFile); err != nil {
			fmt.Printf("⚠️ 临时文件清理失败: %v\n", err)
			return
		}
		fmt.Printf("✅ 临时文件清理成功\n")
	}()

	if push {
		fmt.Printf("步骤3: 推送镜像\n [%s]...\n", newTag)
		pushCmd := exec.Command("docker", "push", newTag)
		if err := pushCmd.Run(); err != nil {
			fmt.Printf("❌ 镜像推送失败: %v\n", err)
			return err
		}
		fmt.Printf("✅ 镜像推送成功\n")
	}

	fmt.Printf("🎉 镜像处理任务完成!\n")

	return nil
}

// PullImage 拉镜像
func PullImage(oldImage, newImage string, msgFn func(string), doneFn func()) error {
	msgFn(fmt.Sprintf("1. docker pull %s", oldImage))
	if err := execCmd(msgFn, "docker", "pull", oldImage); err != nil {
		msgFn(fmt.Sprintf("❌ 拉镜像失败: %v", err))
		doneFn()
		return err
	}
	msgFn("✅ 拉镜像成功")

	msgFn(fmt.Sprintf("\n2. docker tag %s %s", oldImage, newImage))
	err := execCmd(msgFn, "docker", "tag", oldImage, newImage)
	if err != nil {
		msgFn(fmt.Sprintf("❌ 镜像打Tag失败: %v", err))
		doneFn()
		return err
	}
	msgFn("✅ 镜像打Tag成功")

	msgFn(fmt.Sprintf("\n3. docker push %s", newImage))
	if err := execCmd(msgFn, "docker", "push", newImage); err != nil {
		msgFn(fmt.Sprintf("❌ 上传镜像失败: %v", err))
		doneFn()
		return err
	}
	msgFn("✅ 上传镜像成功\n\n🎉 🎉 🎉 OH YEAH ALL DONE!")
	doneFn()
	return nil
}

func GetImages() ([]k8s.NamespaceImages, error) {
	var data []k8s.NamespaceImages
	// 读取文件内容
	filePath := "images.json"
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return data, fmt.Errorf("failed to read file: %w", err)
	}

	// 解析 JSON 数据
	if len(fileData) > 0 { // 如果文件不为空，则解析
		if err := json.Unmarshal(fileData, &data); err != nil {
			return data, fmt.Errorf("failed to unmarshal JSON: %w", err)
		}
	}
	return data, nil
}

func AddImage(ns, name, oldImage, newImage string) error {
	data, err := GetImages()
	if err != nil {
		return err
	}

	if hasNs(ns, data) {
		for i := 0; i < len(data); i++ {
			if ns == data[i].Namespace { // 直接通过索引访问切片元素
				ni := k8s.Deployment{
					Name:      name,
					ImageName: oldImage,
					NewImage:  newImage,
				}
				// 直接修改切片中的元素
				data[i].Deployments = append(data[i].Deployments, ni)
				break
			}
		}
	} else {
		image := k8s.NamespaceImages{
			Namespace: ns,
			Deployments: []k8s.Deployment{
				{
					Name:      name,
					ImageName: oldImage,
					NewImage:  newImage,
				},
			},
		}
		data = append(data, image)
	}
	// 将更新后的数据编码为 JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	filePath := "images.json"
	// 写回文件
	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}

func hasNs(ns string, images []k8s.NamespaceImages) bool {
	for i := 0; i < len(images); i++ {
		item := images[i]
		if ns == item.Namespace {

			return true
		}
	}
	return false
}

func execCmd(stdOutFn func(string), name string, args ...string) error {
	cmd := exec.Command(name, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("❌ 创建标准输出管道失败: %v\n", err)
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Printf("❌ 创建标准错误管道失败: %v\n", err)
		return err
	}

	// 启动命令
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动命令失败: %v", err)
	}

	// 读取标准输出
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			stdOutFn(">>> " + scanner.Text())
		}
	}()

	// 读取标准错误
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			stdOutFn(">>> " + scanner.Text())
		}
	}()

	if err := cmd.Wait(); err != nil {
		return err
	}
	return nil
}
