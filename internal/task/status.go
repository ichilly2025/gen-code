package task

import "time"

// Status represents the status of a task
type Status string

const (
	StatusPending      Status = "pending"
	StatusGenerating   Status = "generating"
	StatusMergingFiles Status = "merging_files"
	StatusCreatingRepo Status = "creating_repo"
	StatusPushing      Status = "pushing"
	StatusCompleted    Status = "completed"
	StatusFailed       Status = "failed"
)

// Task represents a code generation task
type Task struct {
	ID          string    `json:"task_id"`
	Prompt      string    `json:"prompt"`
	RepoName    string    `json:"repo_name"`
	Model       string    `json:"model"`
	GitHubOrg   string    `json:"github_org,omitempty"`
	Status      Status    `json:"status"`
	Message     string    `json:"message"`
	RepoURL     string    `json:"repo_url,omitempty"`
	Error       string    `json:"error,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UpdateStatus updates the task status and message
func (t *Task) UpdateStatus(status Status, message string) {
	t.Status = status
	t.Message = message
	t.UpdatedAt = time.Now()
}

// SetError sets the task error and status to failed
func (t *Task) SetError(err error) {
	t.Status = StatusFailed
	t.Error = err.Error()
	t.Message = "Task failed"
	t.UpdatedAt = time.Now()
}

// IsTerminal returns true if the task is in a terminal state
func (t *Task) IsTerminal() bool {
	return t.Status == StatusCompleted || t.Status == StatusFailed
}
