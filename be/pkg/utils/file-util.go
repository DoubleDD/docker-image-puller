package utils

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

func MergeFiles(dirPath, outputFile string) {

	// 创建或追加到文件
	outFile, err := os.OpenFile(outputFile, os.O_CREATE|os.O_WRONLY, 0644)
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

func CheckFileAndDataMd5(filePath string, data []byte) bool {
	fileMd5 := FileMd5(filePath)
	dataMd5 := DataMd5(data)

	// fmt.Println("校验md5", "\n文件MD5：", fileMd5, "\n数据MD5：", dataMd5)
	return dataMd5 == fileMd5
}
func CheckDataMd5(data []byte, md5Hash string) bool {
	if md5Hash == "" {
		return false
	}
	dataMd5 := DataMd5(data)
	// fmt.Println("校验md5", "\n预期值：", md5Hash, "\n计算值：", dataMd5)
	return md5Hash == dataMd5
}
func CheckFileMd5(filePath, md5Hash string) bool {
	if md5Hash == "" {
		return false
	}
	fileMd5 := FileMd5(filePath)
	// fmt.Println("校验md5", filePath, "\n预期值：", md5Hash, "\n计算值：", fileMd5)
	return md5Hash == fileMd5
}

func DataMd5(data []byte) string {
	// MD5
	md5Hash, md5Time := DataHash(data, md5.New)
	fmt.Printf("MD5: %s (Time: %s)\n", md5Hash, md5Time)
	return md5Hash
}
func FileMd5(filePath string) string {
	// MD5
	md5Hash, md5Time := calculateHash(filePath, md5.New)
	fmt.Printf("MD5: %s (Time: %s)\n", md5Hash, md5Time)
	return md5Hash
}

func calculateHash(filePath string, hashFunc func() hash.Hash) (string, time.Duration) {
	// todo 检查文件是否存在

	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return "", 0
	}
	defer file.Close()

	hash := hashFunc()

	start := time.Now()
	if _, err := io.Copy(hash, file); err != nil {
		fmt.Println("Error hashing file:", err)
		return "", 0
	}
	elapsed := time.Since(start)

	hashInBytes := hash.Sum(nil)
	hashString := hex.EncodeToString(hashInBytes)

	return hashString, elapsed
}

func DataHash(data []byte, hashFunc func() hash.Hash) (string, time.Duration) {
	hash := hashFunc()

	start := time.Now()
	hash.Write(data)
	elapsed := time.Since(start)

	hashInBytes := hash.Sum(nil)
	hashString := hex.EncodeToString(hashInBytes)

	return hashString, elapsed
}

func CreateFileWithData(fileName string, data []byte) error {

	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// 写入数据
	if _, err := file.Write(data); err != nil {
		return err
	}

	return nil
}
