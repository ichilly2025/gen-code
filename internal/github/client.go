package github

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

// Client handles GitHub operations
type Client struct {
	client *github.Client
	token  string
	owner  string
}

// NewClient creates a new GitHub client
func NewClient(token, owner string) *Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	return &Client{
		client: github.NewClient(tc),
		token:  token,
		owner:  owner,
	}
}

// CreateRepository creates a new GitHub repository
func (c *Client) CreateRepository(ctx context.Context, name, description string, private bool) (*github.Repository, error) {
	repo := &github.Repository{
		Name:        github.String(name),
		Description: github.String(description),
		Private:     github.Bool(private),
		AutoInit:    github.Bool(false),
	}

	// Use owner if specified, otherwise create under authenticated user
	owner := c.owner
	if owner == "" {
		owner = "" // Empty string means authenticated user
	}

	createdRepo, resp, err := c.client.Repositories.Create(ctx, owner, repo)
	if err != nil {
		// Provide more helpful error messages
		if resp != nil && resp.StatusCode == 404 {
			if owner != "" {
				return nil, fmt.Errorf("failed to create repository: organization or user '%s' not found, or token lacks permission. Check: 1) Organization exists 2) You are a member 3) Token has 'repo' and 'admin:org' permissions. To create under your personal account, remove GITHUB_OWNER from .env", owner)
			}
			return nil, fmt.Errorf("failed to create repository: authentication failed or token lacks 'repo' permission: %w", err)
		}
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}

	return createdRepo, nil
}

// PushFiles pushes files to a GitHub repository
func (c *Client) PushFiles(ctx context.Context, repoURL, localPath, commitMessage string) error {
	// Clone or init the repository
	repo, err := git.PlainInit(localPath, false)
	if err != nil {
		return fmt.Errorf("failed to init repository: %w", err)
	}

	// Add remote
	_, err = repo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{repoURL},
	})
	if err != nil {
		return fmt.Errorf("failed to add remote: %w", err)
	}

	// Get worktree
	w, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Add all files
	err = w.AddGlob(".")
	if err != nil {
		return fmt.Errorf("failed to add files: %w", err)
	}

	// Commit
	_, err = w.Commit(commitMessage, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Gen Code Bot",
			Email: "bot@gencode.dev",
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	// Push
	err = repo.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth: &http.BasicAuth{
			Username: "git",
			Password: c.token,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to push: %w", err)
	}

	return nil
}

// WriteFilesToDirectory writes files to a local directory
func WriteFilesToDirectory(baseDir string, files map[string]string) error {
	for filePath, content := range files {
		fullPath := filepath.Join(baseDir, filePath)
		
		// Create directory if it doesn't exist
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		// Write file
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", fullPath, err)
		}
	}

	return nil
}

// GetRepoURL returns the HTTPS clone URL for a repository
func GetRepoURL(owner, repoName string) string {
	return fmt.Sprintf("https://github.com/%s/%s.git", owner, repoName)
}
