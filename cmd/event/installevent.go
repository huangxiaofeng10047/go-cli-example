package installevent

import (
	"context"
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
	Trigger(ctx context.Context, ip string, token string) error
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
func (e *InstallEvent) FinishInstall(settings *cli.EnvSettings, cfg *action.Configuration, name string) error {

	updateProgress()

	clientSet, err := cfg.KubernetesClientSet()
	if err != nil {
		return err
	}

	ctx := context.Background()
	ip, err := GetServiceExternalIp(settings, ctx, clientSet, settings.Namespace(), fmt.Sprintf("%s-proxy", name))
	if err != nil {
		return err
	}

	token, err := GetPulsarProxyToken(settings, ctx, clientSet, settings.Namespace(), fmt.Sprintf("%s-token-proxy-admin", name))
	if err != nil {
		return err
	}

	// 触发测试用例
	err = e.client.Trigger(context.Background(), ip, token)
	return err
}
func (e *InstallEvent) WaitTestCaseFinish(settings *cli.EnvSettings, ctx context.Context, out io.Writer) error {
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
