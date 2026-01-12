package generator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cosmos-link/gen-code/internal/github"
	"github.com/cosmos-link/gen-code/internal/llm"
	"github.com/cosmos-link/gen-code/internal/task"
)

// Generator handles code generation and repository creation
type Generator struct {
	llmClient    llm.Client
	githubClient *github.Client
	taskManager  *task.Manager
	tempDir      string
}

// NewGenerator creates a new generator
func NewGenerator(llmClient llm.Client, githubClient *github.Client, taskManager *task.Manager, tempDir string) *Generator {
	return &Generator{
		llmClient:    llmClient,
		githubClient: githubClient,
		taskManager:  taskManager,
		tempDir:      tempDir,
	}
}

// GenerateAndPush generates code and pushes it to GitHub
func (g *Generator) GenerateAndPush(ctx context.Context, taskID string) error {
	// Get task
	t, err := g.taskManager.GetTask(taskID)
	if err != nil {
		return err
	}

	// Update status to generating
	if err := g.taskManager.UpdateTask(taskID, task.StatusGenerating, "Generating code with LLM..."); err != nil {
		return err
	}

	// Generate project using LLM
	project, err := g.llmClient.GenerateProject(ctx, t.Prompt)
	if err != nil {
		g.taskManager.SetTaskError(taskID, fmt.Errorf("failed to generate code: %w", err))
		return err
	}

	// Create temp directory for this project
	projectDir := filepath.Join(g.tempDir, taskID)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		g.taskManager.SetTaskError(taskID, fmt.Errorf("failed to create temp directory: %w", err))
		return err
	}
	// Note: We'll clean up on success, but keep files on error for debugging

	// Update status to merging files
	if err := g.taskManager.UpdateTask(taskID, task.StatusMergingFiles, "Writing files to disk..."); err != nil {
		return err
	}

	// Write files to disk
	fileMap := make(map[string]string)
	for _, file := range project.Files {
		fileMap[file.Path] = file.Content
	}

	if err := github.WriteFilesToDirectory(projectDir, fileMap); err != nil {
		g.taskManager.SetTaskError(taskID, fmt.Errorf("failed to write files: %w", err))
		return err
	}

	// Update status to creating repo
	if err := g.taskManager.UpdateTask(taskID, task.StatusCreatingRepo, "Creating GitHub repository..."); err != nil {
		return err
	}

	// Create GitHub repository
	repo, err := g.githubClient.CreateRepository(ctx, t.RepoName, project.Description, false)
	if err != nil {
		g.taskManager.SetTaskError(taskID, fmt.Errorf("failed to create repository: %w", err))
		return err
	}

	repoURL := *repo.CloneURL
	g.taskManager.SetTaskRepoURL(taskID, *repo.HTMLURL)

	// Update status to pushing
	if err := g.taskManager.UpdateTask(taskID, task.StatusPushing, "Pushing code to GitHub..."); err != nil {
		return err
	}

	// Push files to GitHub
	commitMessage := fmt.Sprintf("Initial commit: %s", project.Description)
	if err := g.githubClient.PushFiles(ctx, repoURL, projectDir, commitMessage); err != nil {
		g.taskManager.SetTaskError(taskID, fmt.Errorf("failed to push files: %w", err))
		return err
	}

	// Update status to completed
	if err := g.taskManager.UpdateTask(taskID, task.StatusCompleted, "Successfully generated and pushed code!"); err != nil {
		return err
	}

	// Clean up temp directory on success
	os.RemoveAll(projectDir)

	return nil
}

// ProcessTask is a convenience method to process a task asynchronously
func (g *Generator) ProcessTask(taskID string) {
	ctx := context.Background()
	if err := g.GenerateAndPush(ctx, taskID); err != nil {
		// Error is already set in the task
		// Temp files are kept for debugging
		return
	}
}
