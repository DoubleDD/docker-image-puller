package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

func UserHomeDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error:", err)
		return ""
	}
	return homeDir

}

func UserHomeTmpDir() string {
	tmpDir := filepath.Join(UserHomeDir(), "tmp")
	return tmpDir
}
