package installevent

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// HTTPTestClient 实现TestClient接口的HTTP客户端
type HTTPTestClient struct {
	client *http.Client
}

// NewHTTPTestClient 创建新的HTTP测试客户端
func NewHTTPTestClient() *HTTPTestClient {
	return &HTTPTestClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Trigger 实现TestClient接口的Trigger方法
func (c *HTTPTestClient) Trigger(ctx context.Context, ip string, token string) error {
	// 构建请求URL
	url := fmt.Sprintf("http://%s:8080/test", ip)

	// 创建新的HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return fmt.Errorf("创建HTTP请求失败: %v", err)
	}

	// 添加认证token
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// 发送请求
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("发送HTTP请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP请求返回非200状态码: %d", resp.StatusCode)
	}

	return nil
}
