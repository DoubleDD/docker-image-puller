package registry

import (
	"archive/tar"
	"docker-image-handler/pkg/utils"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func MergeImageLayers(image string, manifest *Manifest, layerFileMap map[string]string, dest string) (string, error) {
	_, _, repo, tag, err := utils.ParseImageAddress(image)
	if err != nil {
		return "解析镜像地址报错", err
	}
	// 创建临时目录用于存放最终的镜像文件
	outputFile := filepath.Join(dest, fmt.Sprintf("%s_%s.tar", repo+"_"+tag, time.Now().Format("20060102150405")))
	tf, err := utils.CreateFile(outputFile)
	if err != nil {
		return "创建输出文件失败", err
	}
	defer tf.Close()

	// 创建tar writer
	tw := tar.NewWriter(tf)
	defer tw.Close()

	// 构建manifest.json内容
	manifestList := []map[string]interface{}{
		{
			"Config":       "blobs/sha256/" + manifest.Config.Digest[7:] + ".json", // 移除 "sha256:" 前缀
			"RepoTags":     []string{image},
			"Layers":       make([]string, len(*manifest.Layers)),
			"LayerSources": make(map[string]interface{}),
		},
	}

	// 准备层文件名列表
	for i, layer := range *manifest.Layers {
		manifestList[0]["Layers"].([]string)[i] = "blobs/sha256/" + layer.Digest[7:]
		manifestList[0]["LayerSources"].(map[string]interface{})[layer.Digest] = layer
	}

	// 写入 manifest.json
	manifestJson, err := json.MarshalIndent(manifestList, "", "    ")
	if err != nil {
		return "写入 manifest.json 失败", err
	}

	err = addFileToTar(tw, "manifest.json", manifestJson)
	if err != nil {
		return "将 manifest.json 添加到 tar 中失败", err
	}

	configFile, ok := layerFileMap[manifest.Config.Digest]
	if !ok {
		return "读取配置文件失败", err
	}

	// 读取配置文件
	configData, err := os.ReadFile(configFile)
	if err != nil {
		return "读取配置文件失败", err
	}

	// 添加配置文件到tar
	err = addFileToTar(tw, "blobs/sha256/"+manifest.Config.Digest[7:]+".json", configData)
	if err != nil {
		return "将 config.json 添加到 tar 中失败", err
	}

	// 处理每一层
	for _, layer := range *manifest.Layers {
		// 获取层文件
		layerFile, ok := layerFileMap[layer.Digest]
		if !ok {
			// 如果在map中找不到，尝试从标准路径获取
			layerDir := filepath.Join(utils.UserHomeTmpDir(), strings.Replace(layer.Digest, ":", "_", -1))
			layerFile = filepath.Join(layerDir, "all")

			// 检查文件是否存在，如果不存在则合并目录中的文件
			if _, err := os.Stat(layerFile); os.IsNotExist(err) {
				utils.MergeFiles(layerDir, layerFile)
			}
		}

		// 添加层数据
		layerData, err := os.ReadFile(layerFile)
		if err != nil {
			return "读取层数据失败: " + layerFile, err
		}

		// 创建层目录路径
		layerDir := "blobs/sha256/" + layer.Digest[7:]

		err = addFileToTar(tw, layerDir, layerData)
		if err != nil {
			return "将 layer 添加到 tar 中失败, " + layerFile, err
		}
	}

	// 完成tar文件写入
	if err := tw.Close(); err != nil {
		return "Failed to finalize tar file", err
	}

	return outputFile, nil
}

// 辅助函数：添加文件到tar
func addFileToTar(tw *tar.Writer, name string, data []byte) error {
	header := &tar.Header{
		Name:    name,
		Mode:    0644,
		Size:    int64(len(data)),
		ModTime: time.Now(),
	}

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	if _, err := tw.Write(data); err != nil {
		return err
	}

	return nil
}
