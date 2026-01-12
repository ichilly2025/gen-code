package api

import (
	"fmt"
	"io"
	"log"
	"time"

	"github.com/cosmos-link/gen-code/internal/task"
	"github.com/gin-gonic/gin"
)

// SSEClient represents an SSE client connection
type SSEClient struct {
	TaskID  string
	Channel chan *task.Task
}

// SSEManager manages SSE connections
type SSEManager struct {
	clients    map[string][]*SSEClient
	register   chan *SSEClient
	unregister chan *SSEClient
	broadcast  chan *task.Task
}

// NewSSEManager creates a new SSE manager
func NewSSEManager() *SSEManager {
	manager := &SSEManager{
		clients:    make(map[string][]*SSEClient),
		register:   make(chan *SSEClient),
		unregister: make(chan *SSEClient),
		broadcast:  make(chan *task.Task),
	}

	go manager.run()
	return manager
}

// run starts the SSE manager event loop
func (m *SSEManager) run() {
	for {
		select {
		case client := <-m.register:
			m.clients[client.TaskID] = append(m.clients[client.TaskID], client)

		case client := <-m.unregister:
			if clients, ok := m.clients[client.TaskID]; ok {
				for i, c := range clients {
					if c == client {
						m.clients[client.TaskID] = append(clients[:i], clients[i+1:]...)
						close(c.Channel)
						break
					}
				}
				
				if len(m.clients[client.TaskID]) == 0 {
					delete(m.clients, client.TaskID)
				}
			}

		case task := <-m.broadcast:
			if clients, ok := m.clients[task.ID]; ok {
				for _, client := range clients {
					select {
					case client.Channel <- task:
					default:
						// Client channel is full, skip
					}
				}
			}
		}
	}
}

// Register registers a new SSE client
func (m *SSEManager) Register(taskID string) *SSEClient {
	client := &SSEClient{
		TaskID:  taskID,
		Channel: make(chan *task.Task, 10),
	}
	m.register <- client
	return client
}

// Unregister unregisters an SSE client
func (m *SSEManager) Unregister(client *SSEClient) {
	m.unregister <- client
}

// Broadcast broadcasts a task update to all connected clients
func (m *SSEManager) Broadcast(task *task.Task) {
	m.broadcast <- task
}

// HandleSSE handles SSE connections for task status updates
func HandleSSE(c *gin.Context, sseManager *SSEManager, taskManager *task.Manager) {
	taskID := c.Param("task_id")

	// Check if task exists
	t, err := taskManager.GetTask(taskID)
	if err != nil {
		c.JSON(404, gin.H{"error": "Task not found"})
		return
	}

	// Set headers for SSE
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// Ensure we have a flusher
	flusher, ok := c.Writer.(gin.ResponseWriter)
	if !ok {
		c.JSON(500, gin.H{"error": "Streaming not supported"})
		return
	}

	// Register client
	client := sseManager.Register(taskID)
	defer sseManager.Unregister(client)
	
	log.Printf("SSE client connected for task %s", taskID)
	defer log.Printf("SSE client disconnected for task %s", taskID)

	// Send initial status
	sendSSEMessage(c.Writer, "status", t)
	flusher.Flush()

	// If task is already terminal, close connection
	if t.IsTerminal() {
		return
	}

	// Stream updates
	clientGone := c.Request.Context().Done()
	ticker := time.NewTicker(15 * time.Second) // Send heartbeat every 15 seconds
	defer ticker.Stop()

	for {
		select {
		case <-clientGone:
			log.Printf("SSE client disconnected (context done) for task %s", taskID)
			return

		case task := <-client.Channel:
			log.Printf("SSE sending status update for task %s: %s", taskID, task.Status)
			sendSSEMessage(c.Writer, "status", task)
			flusher.Flush()
			
			// Close connection if task is terminal
			if task.IsTerminal() {
				log.Printf("SSE task %s is terminal, closing connection", taskID)
				return
			}

		case <-ticker.C:
			// Send heartbeat
			fmt.Fprintf(c.Writer, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}

// sendSSEMessage sends an SSE message
func sendSSEMessage(w io.Writer, event string, task *task.Task) {
	fmt.Fprintf(w, "event: %s\n", event)
	fmt.Fprintf(w, "data: {\"status\":\"%s\",\"message\":\"%s\"", task.Status, task.Message)
	
	if task.RepoURL != "" {
		fmt.Fprintf(w, ",\"repo_url\":\"%s\"", task.RepoURL)
	}
	
	if task.Error != "" {
		fmt.Fprintf(w, ",\"error\":\"%s\"", task.Error)
	}
	
	fmt.Fprintf(w, "}\n\n")
}
