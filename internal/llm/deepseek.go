package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

// DeepSeekClient implements the Client interface for DeepSeek
type DeepSeekClient struct {
	client *openai.Client
	model  string
}

// NewDeepSeekClient creates a new DeepSeek client
func NewDeepSeekClient(apiKey, baseURL string) *DeepSeekClient {
	config := openai.DefaultConfig(apiKey)
	config.BaseURL = baseURL
	
	return &DeepSeekClient{
		client: openai.NewClientWithConfig(config),
		model:  "deepseek-chat",
	}
}

// GetModelName returns the model name
func (c *DeepSeekClient) GetModelName() string {
	return "deepseek"
}

// GenerateProject generates a complete project structure and files
func (c *DeepSeekClient) GenerateProject(ctx context.Context, prompt string) (*GeneratedProject, error) {
	// First, generate the project structure
	systemPrompt := `你是一个专业的代码生成助手。根据用户的需求，生成完整的项目结构和代码。

请返回JSON格式的响应，格式如下：
{
  "name": "项目名称",
  "description": "项目描述",
  "files": [
    {
      "path": "文件路径",
      "content": "文件内容（保持简洁）",
      "type": "文件类型(go/python/js/md等)"
    }
  ]
}

重要注意事项：
1. 文件数量限制在5个以内
2. 每个文件的内容保持简洁，生成核心功能代码
3. README.md要简短清晰
4. 确保返回完整有效的JSON，不要截断
5. 文件内容中的字符串要正确转义
6. 支持的文件类型：go, py, js, ts, md, json, yaml`

	userPrompt := fmt.Sprintf("请根据以下需求生成项目：\n%s", prompt)

	resp, err := c.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: c.model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: systemPrompt,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: userPrompt,
				},
			},
			Temperature: 0.7,
			MaxTokens:   8000,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to call DeepSeek API: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from DeepSeek API")
	}

	content := resp.Choices[0].Message.Content
	
	// Try to extract JSON from markdown code blocks if present
	content = extractJSON(content)

	// Check if content is truncated
	if !strings.HasSuffix(strings.TrimSpace(content), "}") && !strings.HasSuffix(strings.TrimSpace(content), "]") {
		return nil, fmt.Errorf("response appears to be truncated, please simplify your prompt or reduce project complexity")
	}

	var project GeneratedProject
	if err := json.Unmarshal([]byte(content), &project); err != nil {
		// Try to provide helpful error message
		contentPreview := content
		if len(content) > 500 {
			contentPreview = content[:500] + "..." + content[len(content)-100:]
		}
		return nil, fmt.Errorf("failed to parse JSON response: %w. Preview: %s", err, contentPreview)
	}

	return &project, nil
}

// GenerateFile generates a single file
func (c *DeepSeekClient) GenerateFile(ctx context.Context, prompt string, filePath string, fileType string) (string, error) {
	systemPrompt := fmt.Sprintf(`你是一个专业的代码生成助手。根据用户需求生成%s类型的文件内容。
只返回文件的实际内容，不要包含任何解释或markdown标记。`, fileType)

	userPrompt := fmt.Sprintf("请为文件 %s 生成内容：\n%s", filePath, prompt)

	resp, err := c.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: c.model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: systemPrompt,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: userPrompt,
				},
			},
			Temperature: 0.7,
			MaxTokens:   2000,
		},
	)

	if err != nil {
		return "", fmt.Errorf("failed to call DeepSeek API: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from DeepSeek API")
	}

	return resp.Choices[0].Message.Content, nil
}

// extractJSON extracts JSON content from markdown code blocks
func extractJSON(content string) string {
	// Remove markdown code blocks if present
	content = strings.TrimSpace(content)
	
	if strings.HasPrefix(content, "```json") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimPrefix(content, "```")
		if idx := strings.LastIndex(content, "```"); idx != -1 {
			content = content[:idx]
		}
	} else if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```")
		if idx := strings.LastIndex(content, "```"); idx != -1 {
			content = content[:idx]
		}
	}
	
	return strings.TrimSpace(content)
}
