package trackapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"m8-track-go/config"
)

// Client 17track HTTP 客户端
type Client struct {
	apiKey    string
	baseURL   string
	batchSize int
	httpClient *http.Client
}

// NewClient 创建 17track API 客户端
func NewClient(cfg config.Track17Config) *Client {
	return &Client{
		apiKey:    cfg.APIKey,
		baseURL:   cfg.BaseURL,
		batchSize: cfg.BatchSize,
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.HTTPTimeoutMs) * time.Millisecond,
		},
	}
}

// BatchSize 返回批次大小
func (c *Client) BatchSize() int {
	return c.batchSize
}

// post 发送 POST 请求到 17track API
func (c *Client) post(ctx context.Context, path string, body interface{}) (json.RawMessage, error) {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("序列化请求体失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("17token", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 17track %s 失败: %w", path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	log.Printf("17track %s 响应: %s", path, string(respBody))

	var result struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data,omitempty"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if result.Code != 0 {
		log.Printf("17track %s 接口返回错误: %s", path, string(respBody))
		return nil, fmt.Errorf("17track API 错误码: %d", result.Code)
	}

	return result.Data, nil
}
