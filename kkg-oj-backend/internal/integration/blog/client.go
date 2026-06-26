package blog

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL       string
	internalToken string
	client        *http.Client
}

type AgentPostRequest struct {
	QuestionID int64
	Title      string
	Summary    string
	Markdown   string
	Account    string
	Password   string
	Email      string
}

func NewClient(baseURL string, internalToken string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Client{
		baseURL:       strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		internalToken: strings.TrimSpace(internalToken),
		client:        &http.Client{Timeout: timeout},
	}
}

func (c *Client) Enabled() bool {
	return c != nil && c.baseURL != ""
}

func (c *Client) FetchPostPreviewMap(postIDs []int64) map[int64]map[string]interface{} {
	result := make(map[int64]map[string]interface{}, len(postIDs))
	if !c.Enabled() || len(postIDs) == 0 {
		return result
	}
	for _, postID := range postIDs {
		if postID <= 0 {
			continue
		}
		resp, err := c.client.Get(fmt.Sprintf("%s/api/v1/posts/%d", c.baseURL, postID))
		if err != nil || resp == nil {
			continue
		}
		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil || resp.StatusCode >= 400 {
			continue
		}
		var envelope struct {
			Code int                    `json:"code"`
			Data map[string]interface{} `json:"data"`
		}
		if json.Unmarshal(body, &envelope) != nil || envelope.Code != 0 || envelope.Data == nil {
			continue
		}
		result[postID] = map[string]interface{}{
			"id":                envelope.Data["id"],
			"title":             envelope.Data["title"],
			"summary":           envelope.Data["summary"],
			"author_id":         envelope.Data["author_id"],
			"author_name":       envelope.Data["author_name"],
			"authorName":        envelope.Data["author_name"],
			"author_avatar_url": envelope.Data["author_avatar_url"],
			"updated_at":        envelope.Data["updated_at"],
		}
	}
	return result
}

func (c *Client) PublishAgentPost(req AgentPostRequest) (int64, string, error) {
	if !c.Enabled() {
		return 0, "", errors.New("blog base_url 未配置")
	}
	if c.internalToken != "" {
		return c.publishAgentPostInternal(req)
	}
	account := strings.TrimSpace(req.Account)
	password := strings.TrimSpace(req.Password)
	email := strings.TrimSpace(req.Email)
	if account == "" || password == "" || email == "" {
		return 0, "", errors.New("blog agent 账号配置不完整")
	}
	token, err := c.loginOrRegister(account, password, email)
	if err != nil {
		return 0, "", err
	}
	content := req.Markdown + fmt.Sprintf("\n\n---\n\n> 该题解由 KKG Agent 自动生成并经评测通过后发布。题号：%d", req.QuestionID) +
		fmt.Sprintf("\n\n👉 [跳转到题目](/oj/questions/%d)", req.QuestionID)
	postResp, err := c.request(c.baseURL+"/api/v1/posts", token, map[string]interface{}{
		"title":       req.Title,
		"summary":     req.Summary,
		"tags":        []string{"题解", "Agent", "OJ", fmt.Sprintf("Q%d", req.QuestionID)},
		"raw_content": content,
	})
	if err != nil {
		return 0, "", err
	}
	postID, _ := toInt64(postResp["id"])
	if postID <= 0 {
		return 0, "", errors.New("博客创建失败: 无 post id")
	}
	if _, err = c.request(fmt.Sprintf("%s/api/v1/posts/%d/publish", c.baseURL, postID), token, map[string]interface{}{}); err != nil {
		return 0, "", err
	}
	return postID, fmt.Sprintf("/posts/%d", postID), nil
}

func (c *Client) publishAgentPostInternal(req AgentPostRequest) (int64, string, error) {
	account := strings.TrimSpace(req.Account)
	email := strings.TrimSpace(req.Email)
	if account == "" || email == "" {
		return 0, "", errors.New("blog agent 账号配置不完整")
	}
	content := req.Markdown + fmt.Sprintf("\n\n---\n\n> 该题解由 KKG Agent 自动生成并经评测通过后发布。题号：%d", req.QuestionID) +
		fmt.Sprintf("\n\n👉 [跳转到题目](/oj/questions/%d)", req.QuestionID)
	postResp, err := c.requestInternal(c.baseURL+"/api/v1/internal/agent/posts", map[string]interface{}{
		"question_id":   req.QuestionID,
		"title":         req.Title,
		"summary":       req.Summary,
		"markdown":      content,
		"tags":          []string{"题解", "Agent", "OJ", fmt.Sprintf("Q%d", req.QuestionID)},
		"account":       req.Account,
		"email":         req.Email,
		"display_name":  req.Account,
		"source_system": "oj",
	})
	if err != nil {
		return 0, "", err
	}
	postID, _ := toInt64(postResp["id"])
	if postID <= 0 {
		return 0, "", errors.New("博客创建失败: 无 post id")
	}
	postURL, _ := postResp["url"].(string)
	if strings.TrimSpace(postURL) == "" {
		postURL = fmt.Sprintf("/posts/%d", postID)
	}
	return postID, postURL, nil
}

func (c *Client) loginOrRegister(account, password, email string) (string, error) {
	loginResp, err := c.request(c.baseURL+"/api/v1/auth/login", "", map[string]interface{}{
		"account":  account,
		"password": password,
	})
	if err == nil {
		token, _ := loginResp["access_token"].(string)
		if strings.TrimSpace(token) != "" {
			return token, nil
		}
	}
	_, _ = c.request(c.baseURL+"/api/v1/auth/register", "", map[string]interface{}{
		"username": account,
		"email":    email,
		"password": password,
	})
	loginResp, err = c.request(c.baseURL+"/api/v1/auth/login", "", map[string]interface{}{
		"account":  account,
		"password": password,
	})
	if err != nil {
		return "", err
	}
	token, _ := loginResp["access_token"].(string)
	if strings.TrimSpace(token) == "" {
		return "", errors.New("blog 登录成功但 token 为空")
	}
	return token, nil
}

func (c *Client) request(url string, token string, body map[string]interface{}) (map[string]interface{}, error) {
	payload, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out struct {
		Code    int                    `json:"code"`
		Message string                 `json:"message"`
		Data    map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, errors.New("blog 响应解析失败: " + string(raw))
	}
	if resp.StatusCode >= 300 || out.Code != 0 {
		return nil, errors.New(out.Message)
	}
	return out.Data, nil
}

func (c *Client) requestInternal(url string, body map[string]interface{}) (map[string]interface{}, error) {
	payload, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Auth", c.internalToken)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out struct {
		Code    int                    `json:"code"`
		Message string                 `json:"message"`
		Data    map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, errors.New("blog 响应解析失败: " + string(raw))
	}
	if resp.StatusCode >= 300 || out.Code != 0 {
		return nil, errors.New(out.Message)
	}
	return out.Data, nil
}

func toInt64(v interface{}) (int64, error) {
	switch t := v.(type) {
	case int64:
		return t, nil
	case int:
		return int64(t), nil
	case int32:
		return int64(t), nil
	case float64:
		return int64(t), nil
	case string:
		var id int64
		_, err := fmt.Sscanf(t, "%d", &id)
		return id, err
	default:
		return 0, errors.New("invalid type")
	}
}
