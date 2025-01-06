package linux

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// 安装服务
func InstallService() error {
	// 获取当前可执行文件的路径
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %v", err)
	}

	binDir := "/opt/docker-tools"
	targetPath := filepath.Join(binDir, "docker-tools")

	// 确保 /opt/docker-tools 目录存在
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("failed to create /opt/docker-tools directory: %v", err)
	}

	// 复制文件
	if err := copyFile(exePath, targetPath); err != nil {
		return fmt.Errorf("failed to copy executable: %v", err)
	}

	// 设置可执行权限
	if err := os.Chmod(targetPath, 0755); err != nil {
		return fmt.Errorf("failed to set executable permissions: %v", err)
	}

	// 确保日志目录存在
	logDir := "/opt/docker-tools/log"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}

	// 创建 systemd 服务文件内容
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
	err = os.WriteFile(serviceFilePath, []byte(serviceContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write service file: %v", err)
	}

	// 重新加载 systemd 配置
	reloadCmd := exec.Command("systemctl", "daemon-reload")
	err = reloadCmd.Run()
	if err != nil {
		return fmt.Errorf("failed to reload systemd: %v", err)
	}

	// 启用服务
	enableCmd := exec.Command("systemctl", "enable", "docker-tools.service")
	err = enableCmd.Run()
	if err != nil {
		return fmt.Errorf("failed to enable service: %v", err)
	}

	// 启动服务
	startCmd := exec.Command("systemctl", "start", "docker-tools.service")
	err = startCmd.Run()
	if err != nil {
		return fmt.Errorf("failed to start service: %v", err)
	}

	return nil
}

// 卸载服务
func UninstallService() error {
	var err error
	binDir := "/opt/docker-tools"
	targetPath := filepath.Join(binDir, "docker-tools")

	// 停止服务
	stopCmd := exec.Command("systemctl", "stop", "docker-tools.service")
	err = stopCmd.Run()
	if err != nil {
		return fmt.Errorf("failed to stop service: %v", err)
	}

	// 禁用服务
	disableCmd := exec.Command("systemctl", "disable", "docker-tools.service")
	err = disableCmd.Run()
	if err != nil {
		return fmt.Errorf("failed to disable service: %v", err)
	}

	// 删除服务文件
	serviceFilePath := "/etc/systemd/system/docker-tools.service"
	err = os.Remove(serviceFilePath)
	if err != nil {
		return fmt.Errorf("failed to remove service file: %v", err)
	}

	// 重新加载 systemd 配置
	reloadCmd := exec.Command("systemctl", "daemon-reload")
	err = reloadCmd.Run()
	if err != nil {
		return fmt.Errorf("failed to reload systemd: %v", err)
	}

	// 删除可执行文件
	err = os.Remove(targetPath)
	if err != nil {
		return fmt.Errorf("failed to remove executable: %v", err)
	}

	return nil
}

// 复制文件
func copyFile(src, dst string) error {
	// 确保目标文件的父目录存在
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destinationFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		return err
	}

	return nil
}
