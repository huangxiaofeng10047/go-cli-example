package installevent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HTTPTestClient 实现TestClient接口的HTTP客户端
type HTTPTestClient struct {
	client *http.Client
	schema string
	host   string
	port   int
}

// NewHTTPTestClient 创建新的HTTP测试客户端
func NewHTTPTestClient(schema string, host string, port int) *HTTPTestClient {
	return &HTTPTestClient{
		client: &http.Client{
			Timeout: 30 * time.Minute,
		},
		schema: schema,
		host:   host,
		port:   port,
	}
}

// Trigger 实现TestClient接口的Trigger方法
func (c *HTTPTestClient) Trigger(ctx context.Context, ip string, token string) error {
	// 构建请求URL，使用配置的参数
	url := fmt.Sprintf("%s://%s:%d/testpulsar", c.schema, c.host, c.port)
	//输出请求信息：
	fmt.Printf("请求信息:\n")
	fmt.Printf("URL: %s\n", url)
	fmt.Printf("Client配置: schema=%s, host=%s, port=%d\n", c.schema, c.host, c.port)
	// 构建请求体
	requestBody := map[string]string{
		"pulsarServiceUrl":     "pulsar://" + ip + ":6650",
		"pulsarHttpServiceUrl": "http://" + ip,
		"authToken":            token,
	}
	// 打印请求体信息
	fmt.Printf("请求体:\n")
	fmt.Printf("PulsarServiceUrl: %s\n", requestBody["pulsarServiceUrl"])
	fmt.Printf("PulsarHttpServiceUrl: %s\n", requestBody["pulsarHttpServiceUrl"])
	fmt.Printf("Token长度: %d\n", len(token))
	// 将请求体转换为JSON
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("JSON编码失败: %v", err)
	}

	// 创建请求，设置请求体
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("创建HTTP请求失败: %v", err)
	}

	// 设置Content-Type
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("发送HTTP请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应体失败: %v", err)
	}

	// 打印响应信息
	fmt.Printf("\n响应信息:\n")
	fmt.Printf("状态码: %d\n", resp.StatusCode)
	fmt.Printf("响应头: %v\n", resp.Header)
	fmt.Printf("响应体: %s\n", string(body))

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP请求返回非200状态码: %d", resp.StatusCode)
	}

	return nil
}
