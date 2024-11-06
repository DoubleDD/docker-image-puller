package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

func MergeFiles(dirPath, outputFile string) {

	// 创建或追加到文件
	outFile, err := os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error creating output file:", err)
		return
	}
	defer outFile.Close()

	// 存储所有文件的路径
	var filePaths []string

	// 遍历目录，收集文件路径
	err = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		// 过滤目录，只处理文件
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if path == outputFile {
			return nil
		}

		// 将文件路径添加到文件路径切片中
		filePaths = append(filePaths, path)
		return nil
	})

	if err != nil {
		fmt.Println("Error reading directory:", err)
		return
	}

	// 按文件名字母表顺序排序
	sort.Strings(filePaths)

	// 遍历排序后的文件路径
	for _, path := range filePaths {
		// 打开每个文件
		inFile, err := os.Open(path)
		if err != nil {
			fmt.Println("Error opening file:", err)
			continue
		}
		defer inFile.Close()

		// 将文件内容复制到目标大文件中
		_, err = io.Copy(outFile, inFile)
		if err != nil {
			fmt.Println("Error copying file content:", err)
			continue
		}
	}
}

func CreateFile(filename string) (*os.File, error) {
	// 获取目标文件的父目录路径
	dir := filepath.Dir(filename)

	// 确保父目录存在，若不存在则递归创建
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		fmt.Println("Error creating directories:", err)
		return nil, err
	}
	return os.Create(filename)
}
