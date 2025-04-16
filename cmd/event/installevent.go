package installevent

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"helm.sh/helm/v4/pkg/action"
	"helm.sh/helm/v4/pkg/cli"
	"strings"
	"sync"
	"time"

	//"helm.sh/helm/v4/pkg/cli"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type TestConfig struct {
	Schema string
	Host   string
	Port   int
}

// 定义响应结构体
type TestCaseResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data"`
}

// 嵌套的错误信息结构体
type ErrorInfo struct {
	ErrorMsg string `json:"errorMsg"`
	ReqId    int64  `json:"reqId"`
	Remote   string `json:"remote"`
	Local    string `json:"local"`
}

// 数据项结构体
type TestDataItem struct {
	Name      string    `json:"name"`
	Message   string    `json:"message"`
	ErrorInfo ErrorInfo `json:"-"` // 用于存储解析后的错误信息
}

// 状态响应结构体
type TaskStatusResponse struct {
	Code    int            `json:"code"`
	Message string         `json:"message"`
	Data    []TestDataItem `json:"data"`
}

func NewInstallEvent(config *TestConfig) *InstallEvent {
	var client TestClient = NewHTTPTestClient(config.Schema, config.Host, config.Port)
	showProgressBar()
	return &InstallEvent{client: client}
}

var bar *pb.ProgressBar

func showProgressBar() {
	if bar != nil {
		// 避免重复创建
		return
	}
	tmpl := `{{ red "With funcs:" }} {{ bar . "<" "-" (cycle . "↖" "↗" "↘" "↙" ) "." ">"}} {{speed . | rndcolor }} {{percent .}} {{string . "my_green_string" | green}} {{string . "my_blue_string" | blue}}`
	// 基于 pb 的模板开启一个进度条
	bar = pb.ProgressBarTemplate(tmpl).Start64(2)
	// 为 string 元素设置值
	bar.Set("my_green_string", "green").
		Set("my_blue_string", "blue")
	//defer bar.Finish()
}

var barMutex sync.Mutex

func updateProgress() {
	barMutex.Lock()
	defer barMutex.Unlock()
	if bar != nil {
		bar.Increment()
		if bar.Current() >= bar.Total() {
			bar.Finish()
			bar = nil
		}
	}
}

// TestClient 定义了测试用例触发接口
type TestClient interface {
	Trigger(ctx context.Context, ip string, token string) ([]byte, error)
	Trigger2(ctx context.Context, taskId string) (string, error)
}

// installEvent 处理安装事件
type InstallEvent struct {
	client TestClient
}

//// NewInstallEvent 创建新的安装事件处理器
//func NewInstallEvent(schema string, host string, port int) *installEvent {
//	var client TestClient = NewHTTPTestClient(schema, host, port)
//	return &installEvent{client: client}
//}

// GetServiceExternalIp 获取服务的外部IP
func GetServiceExternalIp(settings *cli.EnvSettings, ctx context.Context, clientset kubernetes.Interface, namespace, serviceName string) (string, error) {
	svc, err := clientset.CoreV1().Services(namespace).Get(ctx, serviceName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get service %s: %v", serviceName, err)
	}

	// 检查 LoadBalancer 类型
	if svc.Spec.Type == "LoadBalancer" {
		if len(svc.Status.LoadBalancer.Ingress) > 0 {
			// 优先检查 IP
			if svc.Status.LoadBalancer.Ingress[0].IP != "" {
				return svc.Status.LoadBalancer.Ingress[0].IP, nil
			}
			// 如果没有 IP，则检查 Hostname
			if svc.Status.LoadBalancer.Ingress[0].Hostname != "" {
				return svc.Status.LoadBalancer.Ingress[0].Hostname, nil
			}
		}
	}

	// 检查 NodePort 类型
	if svc.Spec.Type == "NodePort" {
		// 获取集群中的第一个节点
		nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		if err != nil {
			return "", fmt.Errorf("failed to list nodes: %v", err)
		}

		if len(nodes.Items) > 0 {
			for _, addr := range nodes.Items[0].Status.Addresses {
				if addr.Type == "ExternalIP" {
					return addr.Address, nil
				}
				// 如果没有外部IP，使用内部IP
				if addr.Type == "InternalIP" {
					return addr.Address, nil
				}
			}
		}
	}
	// 检查 ClusterIP 类型
	if svc.Spec.ClusterIP != "" && svc.Spec.ClusterIP != "None" {
		return svc.Spec.ClusterIP, nil
	}

	return "", fmt.Errorf("no IP address found for service %s (type: %s)", serviceName, svc.Spec.Type)
}

// GetPulsarProxyToken 获取Pulsar代理的Token
func GetPulsarProxyToken(settings *cli.EnvSettings, ctx context.Context, clientset kubernetes.Interface, namespace, secretName string) (string, error) {
	secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get secret %s: %v", secretName, err)
	}

	token, ok := secret.Data["TOKEN"]
	if !ok {
		return "", fmt.Errorf("token not found in secret %s", secretName)
	}
	fmt.Println(string(token))
	return string(token), nil
}

// FinishInstall 完成安装过程
func (e *InstallEvent) FinishInstall(settings *cli.EnvSettings, cfg *action.Configuration, name string) (string, error) {

	updateProgress()

	clientSet, err := cfg.KubernetesClientSet()
	if err != nil {
		return "", err
	}

	ctx := context.Background()
	ip, err := GetServiceExternalIp(settings, ctx, clientSet, settings.Namespace(), fmt.Sprintf("%s-proxy", name))
	if err != nil {
		return "", err
	}

	token, err := GetPulsarProxyToken(settings, ctx, clientSet, settings.Namespace(), fmt.Sprintf("%s-token-proxy-admin", name))
	if err != nil {
		return "", err
	}

	// 触发测试用例
	responseBody, _ := e.client.Trigger(context.Background(), ip, token)
	var testCaseResponse TestCaseResponse
	err = json.Unmarshal(responseBody, &testCaseResponse)
	if err != nil {
		return "", fmt.Errorf("解析响应体失败: %v", err)
	}

	return testCaseResponse.Data, err
}
func (e *InstallEvent) WaitTestCaseFinish(settings *cli.EnvSettings, ctx context.Context, out io.Writer, taskId string) error {
	// 定义最大重试次数和重试间隔
	maxRetries := 1000000
	retryInterval := 600 * time.Second
	for i := 0; i < maxRetries; i++ {
		// 调用 Trigger2 方法检查任务状态
		resp, err := e.client.Trigger2(context.Background(), taskId)
		if err != nil {
			return err
		}
		var testCaseResponse TestCaseResponse

		statusResponseBody := []byte(resp)
		err = json.Unmarshal(statusResponseBody, &testCaseResponse)

		if testCaseResponse.Data == "任务正在执行中" {
			time.Sleep(retryInterval)
			continue
		}
		var statusResponse TaskStatusResponse
		err = json.Unmarshal(statusResponseBody, &statusResponse)
		if err != nil {
			return fmt.Errorf("解析任务状态响应体失败: %v", err)
		}

		// 解析每个数据项中的 message 字段
		for i := range statusResponse.Data {
			var errorInfo ErrorInfo
			err := json.Unmarshal([]byte(statusResponse.Data[i].Message), &errorInfo)
			if err != nil {
				fmt.Printf("解析 %s 的错误信息失败: %v\n", statusResponse.Data[i].Name, err)
			} else {
				statusResponse.Data[i].ErrorInfo = errorInfo
			}
		}

		// 这里可以根据解析后的数据进行相应的处理
		var isDone bool = false
		for _, item := range statusResponse.Data {
			fmt.Printf("测试名称: %s\n", item.Name)
			fmt.Printf("错误信息: %s\n", item.ErrorInfo.ErrorMsg)
			fmt.Printf("请求 ID: %d\n", item.ErrorInfo.ReqId)
			fmt.Printf("远程地址: %s\n", item.ErrorInfo.Remote)
			fmt.Printf("本地地址: %s\n", item.ErrorInfo.Local)
			fmt.Println("------------------------")
			isDone = true
		}
		if isDone {
			fmt.Println("任务已完成（达到最大重试次数）")
			break
		}
		// 假设没有专门的状态字段，可根据实际情况调整循环退出条件
		// 这里简单模拟任务完成情况
		// 如果需要根据实际状态判断，可添加相应逻辑
		if i == maxRetries-1 {
			fmt.Println("任务已完成（达到最大重试次数）")
			break
		}

		fmt.Printf("任务未完成，等待 %v 后重试...\n", retryInterval)
		time.Sleep(retryInterval)
	}

	return nil
}

func (e *InstallEvent) QueryRunningPod(settings *cli.EnvSettings, ctx context.Context, cfg *action.Configuration, out io.Writer) error {
	clientSet, err := cfg.KubernetesClientSet()
	if err != nil {
		return fmt.Errorf("获取k8s客户端失败: %v\n")
	}

	namespace := settings.Namespace()
	timeout := time.After(15 * time.Minute)
	tick := time.NewTicker(15 * time.Second)
	defer tick.Stop()
	updateProgress()
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("操作被取消")
		case <-timeout:
			return fmt.Errorf("等待Pod运行超时")
		case <-tick.C:
			pods, err := clientSet.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
			if err != nil {
				return fmt.Errorf("获取Pod列表失败: %v", err)
			}

			if len(pods.Items) == 0 {
				fmt.Fprintf(out, "命名空间 %s 中没有发现Pod\n", namespace)
				continue
			}

			allRunning := true
			notRunningPods := []string{}

			for _, pod := range pods.Items {
				if pod.Status.Phase != "Running" && pod.Status.Phase != "Succeeded" {
					allRunning = false
					notRunningPods = append(notRunningPods, fmt.Sprintf("%s(%s)", pod.Name, pod.Status.Phase))
				}
			}

			if allRunning {
				fmt.Fprintf(out, "命名空间 %s 中的所有Pod都已运行\n", namespace)
				return nil
			}

			fmt.Fprintf(out, "等待以下Pod运行: %s\n", strings.Join(notRunningPods, ", "))
		}
	}
}
