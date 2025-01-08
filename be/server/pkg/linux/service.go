package linux

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

// InstallService 安装服务
func InstallService() error {
	log.Println("Starting service installation...")

	// 获取当前可执行文件的路径
	log.Println("Getting executable path...")
	exePath, err := os.Executable()
	if err != nil {
		log.Printf("Failed to get executable path: %v\n", err)
		return fmt.Errorf("failed to get executable path: %v", err)
	}
	log.Printf("Executable path: %s\n", exePath)

	binDir := "/opt/docker-tools"
	targetPath := filepath.Join(binDir, "docker-tools")

	// 确保 /opt/docker-tools 目录存在
	log.Printf("Ensuring directory exists: %s\n", binDir)
	if err := os.MkdirAll(binDir, 0755); err != nil {
		log.Printf("Failed to create directory %s: %v\n", binDir, err)
		return fmt.Errorf("failed to create /opt/docker-tools directory: %v", err)
	}
	log.Printf("Directory %s created or already exists.\n", binDir)

	// 复制文件
	log.Printf("Copying executable from %s to %s...\n", exePath, targetPath)
	if err := copyFile(exePath, targetPath); err != nil {
		log.Printf("Failed to copy executable: %v\n", err)
		return fmt.Errorf("failed to copy executable: %v", err)
	}
	log.Println("Executable copied successfully.")

	// 设置可执行权限
	log.Printf("Setting executable permissions on %s...\n", targetPath)
	if err := os.Chmod(targetPath, 0755); err != nil {
		log.Printf("Failed to set executable permissions: %v\n", err)
		return fmt.Errorf("failed to set executable permissions: %v", err)
	}
	log.Println("Executable permissions set successfully.")

	// 确保日志目录存在
	logDir := "/opt/docker-tools/log"
	log.Printf("Ensuring log directory exists: %s\n", logDir)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("Failed to create log directory %s: %v\n", logDir, err)
		return fmt.Errorf("failed to create log directory: %v", err)
	}
	log.Printf("Log directory %s created or already exists.\n", logDir)

	// 创建 systemd 服务文件内容
	log.Println("Creating systemd service file content...")
	serviceContent := `
[Unit]
Description=Docker Tools Service
After=network.target

[Service]
Type=simple
WorkingDirectory=/opt/docker-tools
ExecStart=/opt/docker-tools/docker-tools
StandardOutput=append:/opt/docker-tools/log/docker-tools.log
StandardError=append:/opt/docker-tools/log/docker-tools.error.log
Restart=always
User=root
Group=root

# 确保日志文件存在并设置适当的权限
ExecStartPre=/bin/mkdir -p /opt/docker-tools/log
ExecStartPre=/bin/touch /opt/docker-tools/log/docker-tools.log /opt/docker-tools/log/docker-tools.error.log
ExecStartPre=/bin/chown root:root /opt/docker-tools/log/docker-tools.log /opt/docker-tools/log/docker-tools.error.log
ExecStartPre=/bin/chmod 644 /opt/docker-tools/log/docker-tools.log /opt/docker-tools/log/docker-tools.error.log

[Install]
WantedBy=multi-user.target
`

	// 将服务文件写入 /etc/systemd/system/docker-tools.service
	serviceFilePath := "/etc/systemd/system/docker-tools.service"
	log.Printf("Writing service file to %s...\n", serviceFilePath)
	err = os.WriteFile(serviceFilePath, []byte(serviceContent), 0644)
	if err != nil {
		log.Printf("Failed to write service file: %v\n", err)
		return fmt.Errorf("failed to write service file: %v", err)
	}
	log.Println("Service file written successfully.")

	// 重新加载 systemd 配置
	log.Println("Reloading systemd configuration...")
	reloadCmd := exec.Command("systemctl", "daemon-reload")
	err = reloadCmd.Run()
	if err != nil {
		log.Printf("Failed to reload systemd: %v\n", err)
		return fmt.Errorf("failed to reload systemd: %v", err)
	}
	log.Println("Systemd configuration reloaded successfully.")

	// 启用服务
	log.Println("Enabling service...")
	enableCmd := exec.Command("systemctl", "enable", "docker-tools.service")
	err = enableCmd.Run()
	if err != nil {
		log.Printf("Failed to enable service: %v\n", err)
		return fmt.Errorf("failed to enable service: %v", err)
	}
	log.Println("Service enabled successfully.")

	// 启动服务
	log.Println("Starting service...")
	startCmd := exec.Command("systemctl", "start", "docker-tools.service")
	err = startCmd.Run()
	if err != nil {
		log.Printf("Failed to start service: %v\n", err)
		return fmt.Errorf("failed to start service: %v", err)
	}
	log.Println("Service started successfully.")

	log.Println("Service installation completed.")
	return nil
}

// 卸载服务
func UninstallService() error {
	log.Println("Starting service uninstallation...")

	var err error
	binDir := "/opt/docker-tools"
	targetPath := filepath.Join(binDir, "docker-tools")

	// 停止服务
	log.Println("Stopping service...")
	stopCmd := exec.Command("systemctl", "stop", "docker-tools.service")
	err = stopCmd.Run()
	if err != nil {
		log.Printf("Failed to stop service: %v\n", err)
		return fmt.Errorf("failed to stop service: %v", err)
	}
	log.Println("Service stopped successfully.")

	// 禁用服务
	log.Println("Disabling service...")
	disableCmd := exec.Command("systemctl", "disable", "docker-tools.service")
	err = disableCmd.Run()
	if err != nil {
		log.Printf("Failed to disable service: %v\n", err)
		return fmt.Errorf("failed to disable service: %v", err)
	}
	log.Println("Service disabled successfully.")

	// 删除服务文件
	serviceFilePath := "/etc/systemd/system/docker-tools.service"
	log.Printf("Removing service file %s...\n", serviceFilePath)
	err = os.Remove(serviceFilePath)
	if err != nil {
		log.Printf("Failed to remove service file: %v\n", err)
		return fmt.Errorf("failed to remove service file: %v", err)
	}
	log.Println("Service file removed successfully.")

	// 重新加载 systemd 配置
	log.Println("Reloading systemd configuration...")
	reloadCmd := exec.Command("systemctl", "daemon-reload")
	err = reloadCmd.Run()
	if err != nil {
		log.Printf("Failed to reload systemd: %v\n", err)
		return fmt.Errorf("failed to reload systemd: %v", err)
	}
	log.Println("Systemd configuration reloaded successfully.")

	// 删除可执行文件
	log.Printf("Removing executable %s...\n", targetPath)
	err = os.Remove(targetPath)
	if err != nil {
		log.Printf("Failed to remove executable: %v\n", err)
		return fmt.Errorf("failed to remove executable: %v", err)
	}
	log.Println("Executable removed successfully.")

	log.Println("Service uninstallation completed.")
	return nil
}

// 复制文件
func copyFile(src, dst string) error {
	log.Printf("Copying file from %s to %s...\n", src, dst)

	// 确保目标文件的父目录存在
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		log.Printf("Failed to create parent directory for %s: %v\n", dst, err)
		return err
	}

	sourceFile, err := os.Open(src)
	if err != nil {
		log.Printf("Failed to open source file %s: %v\n", src, err)
		return err
	}
	defer sourceFile.Close()

	destinationFile, err := os.Create(dst)
	if err != nil {
		log.Printf("Failed to create destination file %s: %v\n", dst, err)
		return err
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		log.Printf("Failed to copy file content: %v\n", err)
		return err
	}

	log.Println("File copied successfully.")
	return nil
}

func UpdateService() error {
	log.Println("Starting service update...")
	// TODO: 下载文件

	// 停止服务
	log.Println("Stopping service...")
	stopCmd := exec.Command("systemctl", "stop", "docker-tools.service")
	if err := stopCmd.Run(); err != nil {
		log.Printf("Failed to stop service: %v\n", err)
		return fmt.Errorf("failed to stop service: %v", err)
	}
	log.Println("Service stopped successfully.")

	// 替换程序
	log.Println("Replacing program...")
	if err := replaceProgram(); err != nil {
		log.Printf("Failed to replace program: %v\n", err)
		return fmt.Errorf("failed to replace program: %v", err)
	}
	log.Println("Program replaced successfully.")

	// 启动服务
	log.Println("Starting service...")
	startCmd := exec.Command("systemctl", "start", "docker-tools.service")
	if err := startCmd.Run(); err != nil {
		log.Printf("Failed to start service: %v\n", err)
		return fmt.Errorf("failed to start service: %v", err)
	}
	log.Println("Service started successfully.")

	log.Println("Service update completed.")
	return nil
}

// 替换程序
func replaceProgram() error {
	// 获取当前程序的路径
	currentProgramPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current program path: %v", err)
	}

	// 备份当前程序
	backupPath := currentProgramPath + ".bak"
	log.Printf("Backing up current program to %s...\n", backupPath)
	if err := os.Rename(currentProgramPath, backupPath); err != nil {
		return fmt.Errorf("failed to backup current program: %v", err)
	}

	// 替换当前程序
	newProgramPath := "/opt/docker-tools/docker-tools-new"
	log.Printf("Replacing current program with new program...\n")
	if err := os.Rename(newProgramPath, currentProgramPath); err != nil {
		// 如果替换失败，尝试恢复备份
		if err := os.Rename(backupPath, currentProgramPath); err != nil {
			log.Printf("Failed to restore backup: %v\n", err)
		}
		return fmt.Errorf("failed to replace current program: %v", err)
	}

	return nil
}

// 触发更新
func TriggerUpdate() error {
	// 获取当前程序的路径
	currentProgramPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current program path: %v", err)
	}

	// 启动子进程执行更新逻辑
	log.Println("Starting update process...")
	updateCmd := exec.Command(currentProgramPath, "update")
	if err := updateCmd.Start(); err != nil {
		return fmt.Errorf("failed to start update process: %v", err)
	}
	log.Println("Update process started.")

	return nil
}
