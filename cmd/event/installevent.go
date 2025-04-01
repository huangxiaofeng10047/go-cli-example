package installevent

import (
	"context"
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"helm.sh/helm/v4/pkg/action"
	"helm.sh/helm/v4/pkg/cli"
	"strings"
	"time"

	//"helm.sh/helm/v4/pkg/cli"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var count = 100000
var bar = pb.StartNew(count)

//var settings = cli.New()

// TestClient 定义了测试用例触发接口
type TestClient interface {
	Trigger(ctx context.Context, ip string, token string) error
}

// installEvent 处理安装事件
type installEvent struct {
	client TestClient
}

// NewInstallEvent 创建新的安装事件处理器
func NewInstallEvent() *installEvent {
	var client TestClient = NewHTTPTestClient()
	return &installEvent{client: client}
}

// GetServiceExternalIp 获取服务的外部IP
func GetServiceExternalIp(settings *cli.EnvSettings, ctx context.Context, clientset kubernetes.Interface, namespace, serviceName string) (string, error) {
	svc, err := clientset.CoreV1().Services(namespace).Get(ctx, serviceName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get service %s: %v", serviceName, err)
	}

	if len(svc.Status.LoadBalancer.Ingress) > 0 {
		return svc.Status.LoadBalancer.Ingress[0].IP, nil
	}

	return "", fmt.Errorf("no external IP found for service %s", serviceName)
}

// GetPulsarProxyToken 获取Pulsar代理的Token
func GetPulsarProxyToken(settings *cli.EnvSettings, ctx context.Context, clientset kubernetes.Interface, namespace, secretName string) (string, error) {
	secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get secret %s: %v", secretName, err)
	}

	token, ok := secret.Data["token"]
	if !ok {
		return "", fmt.Errorf("token not found in secret %s", secretName)
	}

	return string(token), nil
}

// FinishInstall 完成安装过程
func (e *installEvent) FinishInstall(settings *cli.EnvSettings, cfg *action.Configuration, name string) error {

	// 创建进度条并开始
	bar.Increment()
	bar.Finish()

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
func (e *installEvent) WaitTestCaseFinish(settings *cli.EnvSettings, ctx context.Context, out io.Writer) error {
	return nil
}

func (e *installEvent) QueryRunningPod(settings *cli.EnvSettings, ctx context.Context, cfg *action.Configuration, out io.Writer) error {
	clientSet, err := cfg.KubernetesClientSet()
	if err != nil {
		return fmt.Errorf("获取k8s客户端失败: %v\n")
	}

	namespace := settings.Namespace()
	timeout := time.After(5 * time.Minute)
	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()

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
