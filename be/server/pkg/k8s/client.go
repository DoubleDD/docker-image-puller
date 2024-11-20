package k8s

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"sync"

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

func GetLogs(ns, podName, containerName string, msgConsumer func(string)) error {
	// 获取单例的 clientset
	clientset, err := GetClientset()
	if err != nil {
		return err
	}

	req := clientset.CoreV1().Pods("default").GetLogs(podName, &v1.PodLogOptions{
		TailLines: func() *int64 { t := int64(100); return &t }(),
		Container: containerName,
		Follow:    true,
	})

	stream, err := req.Stream(context.Background())
	if err != nil {
		return err
	}
	defer stream.Close()

	buf := make([]byte, 2048)
	for {
		n, err := stream.Read(buf)
		if err != nil {
			break
		}
		msgConsumer(string(buf[:n]))
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
