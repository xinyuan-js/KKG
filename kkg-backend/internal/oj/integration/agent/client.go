package agent

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL     string
	apiKey      string
	model       string
	temperature float64
	client      *http.Client
}

func NewClient(baseURL, apiKey, model string, temperature float64, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 90 * time.Second
	}
	return &Client{
		baseURL:     strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		apiKey:      strings.TrimSpace(apiKey),
		model:       strings.TrimSpace(model),
		temperature: temperature,
		client:      &http.Client{Timeout: timeout},
	}
}

func (c *Client) Chat(prompt, feedback string) (string, error) {
	if c == nil || c.baseURL == "" || c.apiKey == "" || c.model == "" {
		return "", errors.New("agent 配置不完整")
	}
	userPrompt := prompt
	if strings.TrimSpace(feedback) != "" {
		userPrompt += "\n\n上一轮反馈：\n" + feedback
	}
	payload := map[string]interface{}{
		"model": c.model,
		"messages": []map[string]string{
			{"role": "system", "content": "你是 OJ 题解生成助手。"},
			{"role": "user", "content": userPrompt},
		},
		"temperature": c.temperature,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", errors.New("模型服务错误: " + string(data))
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", errors.New("模型返回为空")
	}
	return strings.TrimSpace(out.Choices[0].Message.Content), nil
}
