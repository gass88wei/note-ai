package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// LLMClient 封装与 LLM API 的交互
type LLMClient struct {
	db *Database
}

func NewLLMClient(db *Database) *LLMClient {
	return &LLMClient{db: db}
}

// llmMessage 表示一条聊天消息
type llmMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// llmChatRequest 表示 API 请求
type llmChatRequest struct {
	Model    string       `json:"model"`
	Messages []llmMessage `json:"messages"`
	Stream   bool         `json:"stream"`
}

// llmChatResponse 表示 API 响应
type llmChatResponse struct {
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// GetAnswer 基于搜索结果和问题，调用 LLM 获取回答
func (c *LLMClient) GetAnswer(searchResults []SearchResult, userQuestion string) (string, error) {
	baseURL, err := c.db.GetSetting("llm_base_url")
	if err != nil {
		return "", fmt.Errorf("读取 LLM API 地址失败：%v", err)
	}
	if baseURL == "" {
		return "", fmt.Errorf("未配置 LLM API 地址")
	}

	// Clean URL - remove trailing slash
	baseURL = strings.TrimRight(baseURL, "/")

	model, err := c.db.GetSetting("llm_model")
	if err != nil || model == "" {
		return "", fmt.Errorf("未配置 LLM 模型")
	}

	apiKey, _ := c.db.GetSetting("llm_api_key")
	systemPrompt, _ := c.db.GetSetting("ai_system_prompt")

	// Build context from search results (compact format to save tokens)
	var contextBuilder strings.Builder
	for i, result := range searchResults {
		source := result.Source
		if source == "" {
			source = "检索"
		}
		// Compact format: [来源1 关键词] 标题\n内容
		entry := fmt.Sprintf("[%d %s] %s\n%s\n", i+1, source, result.Metadata["title"], result.Text)
		if contextBuilder.Len()+len(entry) > 2500 {
			break
		}
		contextBuilder.WriteString(entry)
	}
	context := contextBuilder.String()

	// Concise prompt
	userPrompt := fmt.Sprintf(`参考资料：
%s
问题：%s
要求：只参考以上资料回答，引用来源编号，资料不足请说明。`, context, userQuestion)

	messages := []llmMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	// Build request
	reqBody := llmChatRequest{
		Model:    model,
		Messages: messages,
		Stream:   false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("Failed to marshal request: %v", err)
	}

	// Ensure URL ends with /chat/completions
	if baseURL[len(baseURL)-1] != '/' {
		baseURL += "/"
	}
	apiURL := baseURL + "chat/completions"

	fmt.Printf("[LLM] Calling API: %s\n", apiURL)
	fmt.Printf("[LLM] Model: %s\n", model)
	fmt.Printf("[LLM] Timeout: 120s\n")

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("Failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{Timeout: 120 * time.Second}
	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start)
	fmt.Printf("[LLM] API call took: %v\n", elapsed)
	if err != nil {
		return "", fmt.Errorf("Failed to call LLM API: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("Failed to read response: %v", err)
	}

	fmt.Printf("[LLM] Response status: %d, body length: %d\n", resp.StatusCode, len(body))

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("LLM API returned error (%d): %s", resp.StatusCode, string(body))
	}

	var chatResp llmChatResponse
	if err := json.Unmarshal(body, &chatResp); err == nil && len(chatResp.Choices) > 0 && chatResp.Choices[0].Message.Content != "" {
		return chatResp.Choices[0].Message.Content, nil
	}

	// Fallback: try parsing as SSE streaming format
	if content := parseSSEResponse(string(body)); content != "" {
		return content, nil
	}

	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("Failed to parse LLM response: %v", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("LLM did not return a valid answer (check LM Studio streaming setting)")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// parseSSEResponse 从 SSE 流式响应中提取完整内容
func parseSSEResponse(sseBody string) string {
	var content strings.Builder
	for _, line := range strings.Split(sseBody, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		// Try to parse delta content
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		for _, c := range chunk.Choices {
			content.WriteString(c.Delta.Content)
			if c.FinishReason != nil {
				break
			}
		}
	}
	return content.String()
}

// TestConnection 测试 LLM API 连接
func (c *LLMClient) TestConnection() (bool, string) {
	baseURL, err := c.db.GetSetting("llm_base_url")
	if err != nil {
		return false, "读取 LLM API 地址失败：" + err.Error()
	}
	if baseURL == "" {
		return false, "未配置 LLM API 地址"
	}

	// Clean URL - remove trailing slash
	baseURL = strings.TrimRight(baseURL, "/")

	model, err := c.db.GetSetting("llm_model")
	if err != nil || model == "" {
		return false, "未配置 LLM 模型"
	}

	apiKey, _ := c.db.GetSetting("llm_api_key")

	messages := []llmMessage{
		{Role: "user", Content: "Hi"},
	}

	reqBody := llmChatRequest{
		Model:    model,
		Messages: messages,
		Stream:   false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return false, "序列化请求失败"
	}

	if baseURL[len(baseURL)-1] != '/' {
		baseURL += "/"
	}
	apiURL := baseURL + "chat/completions"

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return false, "创建请求失败：" + err.Error()
	}

	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, "连接失败：" + err.Error()
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return true, "连接成功"
	}

	body, _ := io.ReadAll(resp.Body)
	return false, fmt.Sprintf("连接失败 (%d): %s", resp.StatusCode, string(body))
}
