package docker

import (
	"docker-image-handler/config"
	"docker-image-handler/pkg/utils"
	"fmt"
	"os"
	"os/exec"
)

func PushImage(tarFile, image, registry string) error {
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

	go func() {
		fmt.Printf("步骤3: 推送镜像\n [%s]...\n", newTag)
		pushCmd := exec.Command("docker", "push", newTag)
		if err := pushCmd.Run(); err != nil {
			fmt.Printf("❌ 镜像推送失败: %v\n", err)
		}
		fmt.Printf("✅ 镜像推送成功\n")
	}()

	fmt.Printf("🎉 镜像处理任务完成!\n")

	return nil
}
