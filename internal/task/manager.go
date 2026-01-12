package task

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// StatusCallback is a function that is called when a task status changes
type StatusCallback func(task *Task)

// Manager manages tasks
type Manager struct {
	tasks             map[string]*Task
	mu                sync.RWMutex
	statusCallbacks   map[string][]StatusCallback
	callbackMu        sync.RWMutex
	maxConcurrentTasks int
	activeTasks       int
	taskQueue         chan *Task
	ctx               context.Context
	cancel            context.CancelFunc
}

// NewManager creates a new task manager
func NewManager(maxConcurrentTasks int) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	
	m := &Manager{
		tasks:             make(map[string]*Task),
		statusCallbacks:   make(map[string][]StatusCallback),
		maxConcurrentTasks: maxConcurrentTasks,
		taskQueue:         make(chan *Task, 100),
		ctx:               ctx,
		cancel:            cancel,
	}

	// Start worker pool
	for i := 0; i < maxConcurrentTasks; i++ {
		go m.worker()
	}

	return m
}

// CreateTask creates a new task
func (m *Manager) CreateTask(prompt, repoName, model, githubOrg string) *Task {
	task := &Task{
		ID:        uuid.New().String(),
		Prompt:    prompt,
		RepoName:  repoName,
		Model:     model,
		GitHubOrg: githubOrg,
		Status:    StatusPending,
		Message:   "Task created",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	m.mu.Lock()
	m.tasks[task.ID] = task
	m.mu.Unlock()

	return task
}

// GetTask retrieves a task by ID
func (m *Manager) GetTask(id string) (*Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	task, ok := m.tasks[id]
	if !ok {
		return nil, fmt.Errorf("task not found: %s", id)
	}

	return task, nil
}

// UpdateTask updates a task's status
func (m *Manager) UpdateTask(id string, status Status, message string) error {
	m.mu.Lock()
	task, ok := m.tasks[id]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("task not found: %s", id)
	}
	
	task.UpdateStatus(status, message)
	m.mu.Unlock()

	// Notify callbacks
	m.notifyCallbacks(task)

	return nil
}

// SetTaskError sets a task's error
func (m *Manager) SetTaskError(id string, err error) error {
	m.mu.Lock()
	task, ok := m.tasks[id]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("task not found: %s", id)
	}
	
	task.SetError(err)
	m.mu.Unlock()

	// Notify callbacks
	m.notifyCallbacks(task)

	return nil
}

// SetTaskRepoURL sets the repository URL for a task
func (m *Manager) SetTaskRepoURL(id string, repoURL string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.tasks[id]
	if !ok {
		return fmt.Errorf("task not found: %s", id)
	}
	
	task.RepoURL = repoURL
	task.UpdatedAt = time.Now()

	return nil
}

// SubscribeToTask subscribes to task status updates
func (m *Manager) SubscribeToTask(taskID string, callback StatusCallback) error {
	m.callbackMu.Lock()
	defer m.callbackMu.Unlock()

	// Check if task exists
	m.mu.RLock()
	_, ok := m.tasks[taskID]
	m.mu.RUnlock()
	
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}

	m.statusCallbacks[taskID] = append(m.statusCallbacks[taskID], callback)
	return nil
}

// notifyCallbacks notifies all callbacks for a task
func (m *Manager) notifyCallbacks(task *Task) {
	m.callbackMu.RLock()
	callbacks, ok := m.statusCallbacks[task.ID]
	m.callbackMu.RUnlock()

	if !ok {
		return
	}

	for _, callback := range callbacks {
		go callback(task)
	}

	// Clean up callbacks if task is terminal
	if task.IsTerminal() {
		m.callbackMu.Lock()
		delete(m.statusCallbacks, task.ID)
		m.callbackMu.Unlock()
	}
}

// worker processes tasks from the queue
func (m *Manager) worker() {
	for {
		select {
		case <-m.ctx.Done():
			return
		case task := <-m.taskQueue:
			// Task will be processed by the generator
			// This is just a placeholder for future expansion
			_ = task
		}
	}
}

// Shutdown shuts down the task manager
func (m *Manager) Shutdown() {
	m.cancel()
	close(m.taskQueue)
}
