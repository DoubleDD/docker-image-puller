package k8s

import (
	"bufio"
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	once     sync.Once
	instance *kubernetes.Clientset
)

// GetClientset 返回一个单例的 Kubernetes Clientset
func GetClientset() (*kubernetes.Clientset, error) {
	var err error
	once.Do(func() {
		var config *rest.Config

		// 尝试获取 InClusterConfig
		config, err = rest.InClusterConfig()
		if err != nil {
			// 如果失败，尝试使用 kubeconfig 文件
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return
			}
			kubeconfig := filepath.Join(homeDir, ".kube", "config")
			config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
			if err != nil {
				return
			}
		}

		// 创建 Kubernetes 客户端
		instance, err = kubernetes.NewForConfig(config)
		if err != nil {
			return
		}
	})

	return instance, err
}

func GetImagesNew(namespace string) ([]NamespaceImages, error) {
	var result []NamespaceImages
	var nsList []string

	clientset, err := GetClientset()
	if err != nil {
		return nil, err
	}

	if namespace != "" {
		nsList = append(nsList, namespace)
	} else {
		// 获取所有命名空间
		namespaces, err := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, ns := range namespaces.Items {
			nsList = append(nsList, ns.Name)
		}
	}

	for _, ns := range nsList {
		ns = strings.TrimSpace(ns)
		if ns == "" {
			continue
		}

		// 获取命名空间下的所有 Deployment
		deployments, err := clientset.AppsV1().Deployments(ns).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return nil, err
		}

		var deploymentList []Deployment
		for _, deployment := range deployments.Items {
			deploymentName := deployment.Name

			// 获取 initContainers 的镜像
			var imagesStr string
			for _, container := range deployment.Spec.Template.Spec.InitContainers {
				imagesStr += container.Image + "\n"
			}

			// 如果没有 initContainers，获取 containers 的镜像
			if imagesStr == "" {
				for _, container := range deployment.Spec.Template.Spec.Containers {
					imagesStr += container.Image + "\n"
				}
			}

			if imagesStr != "" {
				deploymentList = append(deploymentList, Deployment{
					Name:      deploymentName,
					ImageName: strings.TrimSpace(imagesStr),
				})
			}
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

func RolloutDeployment(name string, namespace string) string {
	if namespace == "" {
		namespace = "default"
	}

	clientset, err := GetClientset()
	if err != nil {
		return err.Error()
	}

	// 重启 Deployment
	_, err = clientset.AppsV1().Deployments(namespace).Patch(context.TODO(), name, "application/strategic-merge-patch+json", []byte(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt": "`+time.Now().Format(time.RFC3339)+`"}}}}}`), metav1.PatchOptions{})
	if err != nil {
		return err.Error()
	}

	return "Deployment restarted successfully"
}

func GetLogs(ns, podName, containerName string, msgConsumer func(string)) error {
	// 获取单例的 clientset
	clientset, err := GetClientset()
	if err != nil {
		return err
	}

	req := clientset.CoreV1().Pods(ns).GetLogs(podName, &v1.PodLogOptions{
		TailLines: func() *int64 { t := int64(100); return &t }(),
		Container: containerName,
		Follow:    true,
		// Timestamps: true,
	})

	stream, err := req.Stream(context.Background())
	if err != nil {
		return err
	}
	defer stream.Close()

	// 4. 处理日志流
	reader := bufio.NewReader(stream)

	// 用于存储未完成的行
	var incompleteLine string

	for {
		// 读取日志
		buf := make([]byte, 4096) // 缓冲区大小
		n, err := reader.Read(buf)
		if err != nil && err != io.EOF {
			log.Fatalf("Error reading log stream: %v", err)
		}
		if n == 0 {
			break
		}

		// 合并未完成的行和新读取的日志
		logChunk := incompleteLine + string(buf[:n])

		// 分割日志行
		lines := strings.Split(logChunk, "\n")

		// 检查最后一行是否完整
		if !strings.HasSuffix(logChunk, "\n") {
			// 缓存最后一行
			incompleteLine = lines[len(lines)-1]
			lines = lines[:len(lines)-1] // 排除最后一行
		} else {
			// 如果最后一行是完整的，清空缓存
			incompleteLine = ""
		}
		msgConsumer(strings.Join(lines, "\n"))

		// 打印完整的日志行
		// for _, line := range lines {
		// 	fmt.Println(line)
		// }

		// 如果是 EOF 且有未处理的部分，打印缓存
		if err == io.EOF {
			if incompleteLine != "" {
				msgConsumer(strings.Join(lines, "\n"))
			}
			break
		}
	}

	return nil
}

// 获取 Deployment 对应的 Pod 列表
func GetPodsForDeployment(namespace, deploymentName string) ([]string, error) {
	clientset, err := GetClientset()
	if err != nil {
		log.Fatalf("Error getting clientset: %v", err)
		return nil, err
	}

	// 获取 Deployment
	deployment, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// 获取 Deployment 的标签选择器
	selector, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
	if err != nil {
		return nil, err
	}

	// 获取所有匹配标签选择器的 Pod
	podList, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return nil, err
	}

	// 提取 Pod 名称
	podNames := make([]string, len(podList.Items))
	for i, pod := range podList.Items {
		podNames[i] = pod.Name
	}

	return podNames, nil
}

func GetContainersForPod(namespace, podName string) ([]string, error) {
	clientset, err := GetClientset()
	if err != nil {
		log.Fatalf("Error getting clientset: %v", err)
		return nil, err
	}
	// 获取 Pod 对象
	pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		log.Fatalf("Error getting pod: %s", err.Error())
	}

	// 获取 Pod 中的容器列表
	containers := pod.Spec.Containers

	// 创建一个字符串数组来存储容器名称
	containerNames := make([]string, len(containers))
	for i, container := range containers {
		containerNames[i] = container.Name
	}

	return containerNames, nil
}
