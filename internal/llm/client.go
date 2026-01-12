package llm

import (
	"context"
	"encoding/json"
)

// FileInfo represents a file in the generated project
type FileInfo struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Type    string `json:"type"` // go, python, js, md, etc.
}

// GeneratedProject represents the complete generated project
type GeneratedProject struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Files       []FileInfo `json:"files"`
}

// Client is the interface for LLM clients
type Client interface {
	// GenerateProject generates a complete project from a prompt
	GenerateProject(ctx context.Context, prompt string) (*GeneratedProject, error)
	
	// GenerateFile generates a single file content
	GenerateFile(ctx context.Context, prompt string, filePath string, fileType string) (string, error)
	
	// GetModelName returns the name of the model being used
	GetModelName() string
}

// ParseProjectStructure parses the project structure from LLM response
func ParseProjectStructure(response string) (*GeneratedProject, error) {
	var project GeneratedProject
	if err := json.Unmarshal([]byte(response), &project); err != nil {
		return nil, err
	}
	return &project, nil
}
