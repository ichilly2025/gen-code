package api

import (
	"fmt"
	"net/http"

	"github.com/cosmos-link/gen-code/internal/config"
	"github.com/cosmos-link/gen-code/internal/generator"
	"github.com/cosmos-link/gen-code/internal/task"
	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests
type Handler struct {
	generator  *generator.Generator
	taskMgr    *task.Manager
	sseManager *SSEManager
	cfg        *config.Config
}

// NewHandler creates a new handler
func NewHandler(gen *generator.Generator, taskMgr *task.Manager, sseManager *SSEManager, cfg *config.Config) *Handler {
	return &Handler{
		generator:  gen,
		taskMgr:    taskMgr,
		sseManager: sseManager,
		cfg:        cfg,
	}
}

// GenerateRequest represents a generate request
type GenerateRequest struct {
	Prompt    string `json:"prompt" binding:"required"`
	RepoName  string `json:"repo_name" binding:"required"`
	Model     string `json:"model"`
	GitHubOrg string `json:"github_org"`
}

// GenerateResponse represents a generate response
type GenerateResponse struct {
	TaskID  string `json:"task_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// HandleGenerate handles the generate request
func (h *Handler) HandleGenerate(c *gin.Context) {
	var req GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Use default model if not specified
	if req.Model == "" {
		req.Model = h.cfg.LLM.DefaultModel
	}

	// Validate model
	if req.Model != "deepseek" && req.Model != "openai" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid model, must be 'deepseek' or 'openai'"})
		return
	}

	// Create task
	t := h.taskMgr.CreateTask(req.Prompt, req.RepoName, req.Model, req.GitHubOrg)

	// Subscribe SSE manager to task updates
	h.taskMgr.SubscribeToTask(t.ID, func(task *task.Task) {
		h.sseManager.Broadcast(task)
	})

	// Start generation asynchronously
	go h.generator.ProcessTask(t.ID)

	// Return task ID immediately
	c.JSON(http.StatusOK, GenerateResponse{
		TaskID:  t.ID,
		Status:  string(t.Status),
		Message: t.Message,
	})
}

// HandleGetTask handles the get task request
func (h *Handler) HandleGetTask(c *gin.Context) {
	taskID := c.Param("task_id")

	t, err := h.taskMgr.GetTask(taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	c.JSON(http.StatusOK, t)
}

// HandleStatus handles the SSE status endpoint
func (h *Handler) HandleStatus(c *gin.Context) {
	HandleSSE(c, h.sseManager, h.taskMgr)
}

// HandleHealth handles health check
func (h *Handler) HandleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"service": "gen-code",
	})
}

// SetupRouter sets up the Gin router
func SetupRouter(handler *Handler) *gin.Engine {
	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// API routes
	api := r.Group("/api/v1")
	{
		api.POST("/generate", handler.HandleGenerate)
		api.GET("/task/:task_id", handler.HandleGetTask)
		api.GET("/status/:task_id", handler.HandleStatus)
	}

	// Health check
	r.GET("/health", handler.HandleHealth)

	return r
}

// ErrorResponse represents an error response
func ErrorResponse(c *gin.Context, code int, message string) {
	c.JSON(code, gin.H{"error": message})
}

// SuccessResponse represents a success response
func SuccessResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, data)
}

// ValidationError returns a validation error
func ValidationError(message string) error {
	return fmt.Errorf("validation error: %s", message)
}
