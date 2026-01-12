package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

// OpenAIClient implements the Client interface for OpenAI
type OpenAIClient struct {
	client *openai.Client
	model  string
}

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient(apiKey, baseURL string) *OpenAIClient {
	config := openai.DefaultConfig(apiKey)
	if baseURL != "" {
		config.BaseURL = baseURL
	}
	
	return &OpenAIClient{
		client: openai.NewClientWithConfig(config),
		model:  "gpt-4-turbo-preview",
	}
}

// GetModelName returns the model name
func (c *OpenAIClient) GetModelName() string {
	return "openai"
}

// GenerateProject generates a complete project structure and files
func (c *OpenAIClient) GenerateProject(ctx context.Context, prompt string) (*GeneratedProject, error) {
	systemPrompt := `You are a professional code generation assistant. Based on user requirements, generate complete project structure and code.

Please return a JSON response in the following format:
{
  "name": "project name",
  "description": "project description",
  "files": [
    {
      "path": "file path",
      "content": "file content (keep concise)",
      "type": "file type (go/python/js/md etc.)"
    }
  ]
}

IMPORTANT:
1. Limit to 5 files maximum
2. Keep file content concise with core functionality only
3. README.md should be brief and clear
4. Ensure complete valid JSON response without truncation
5. Properly escape strings in file content
6. Supported file types: go, py, js, ts, md, json, yaml`

	userPrompt := fmt.Sprintf("Please generate a project based on the following requirements:\n%s", prompt)

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
		return nil, fmt.Errorf("failed to call OpenAI API: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI API")
	}

	content := resp.Choices[0].Message.Content
	content = extractJSON(content)

	// Check if content is truncated
	if !strings.HasSuffix(strings.TrimSpace(content), "}") && !strings.HasSuffix(strings.TrimSpace(content), "]") {
		return nil, fmt.Errorf("response appears to be truncated, please simplify your prompt or reduce project complexity")
	}

	var project GeneratedProject
	if err := json.Unmarshal([]byte(content), &project); err != nil {
		contentPreview := content
		if len(content) > 500 {
			contentPreview = content[:500] + "..." + content[len(content)-100:]
		}
		return nil, fmt.Errorf("failed to parse JSON response: %w. Preview: %s", err, contentPreview)
	}

	return &project, nil
}

// GenerateFile generates a single file
func (c *OpenAIClient) GenerateFile(ctx context.Context, prompt string, filePath string, fileType string) (string, error) {
	systemPrompt := fmt.Sprintf(`You are a professional code generation assistant. Generate content for a %s file based on user requirements.
Only return the actual file content, without any explanations or markdown formatting.`, fileType)

	userPrompt := fmt.Sprintf("Please generate content for file %s:\n%s", filePath, prompt)

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
		return "", fmt.Errorf("failed to call OpenAI API: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI API")
	}

	return resp.Choices[0].Message.Content, nil
}
