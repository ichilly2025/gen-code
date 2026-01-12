package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cosmos-link/gen-code/internal/api"
	"github.com/cosmos-link/gen-code/internal/config"
	"github.com/cosmos-link/gen-code/internal/generator"
	"github.com/cosmos-link/gen-code/internal/github"
	"github.com/cosmos-link/gen-code/internal/llm"
	"github.com/cosmos-link/gen-code/internal/task"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Starting gen-code service on %s:%s", cfg.Server.Host, cfg.Server.Port)

	// Ensure temp directory exists
	if err := os.MkdirAll(cfg.Task.TempDir, 0755); err != nil {
		log.Fatalf("Failed to create temp directory: %v", err)
	}

	// Create LLM client based on default model
	var llmClient llm.Client
	switch cfg.LLM.DefaultModel {
	case "deepseek":
		if cfg.LLM.DeepSeekAPIKey == "" {
			log.Fatal("DEEPSEEK_API_KEY is required when using deepseek model")
		}
		llmClient = llm.NewDeepSeekClient(cfg.LLM.DeepSeekAPIKey, cfg.LLM.DeepSeekBaseURL)
		log.Println("Using DeepSeek as LLM provider")
	case "openai":
		if cfg.LLM.OpenAIAPIKey == "" {
			log.Fatal("OPENAI_API_KEY is required when using openai model")
		}
		llmClient = llm.NewOpenAIClient(cfg.LLM.OpenAIAPIKey, cfg.LLM.OpenAIBaseURL)
		log.Println("Using OpenAI as LLM provider")
	default:
		log.Fatalf("Unknown model: %s", cfg.LLM.DefaultModel)
	}

	// Create GitHub client
	githubClient := github.NewClient(cfg.GitHub.Token, cfg.GitHub.Owner)
	log.Println("GitHub client initialized")

	// Create task manager
	taskManager := task.NewManager(cfg.Task.MaxConcurrentTasks)
	log.Printf("Task manager initialized with %d concurrent tasks", cfg.Task.MaxConcurrentTasks)

	// Create generator
	gen := generator.NewGenerator(llmClient, githubClient, taskManager, cfg.Task.TempDir)
	log.Println("Code generator initialized")

	// Create SSE manager
	sseManager := api.NewSSEManager()
	log.Println("SSE manager initialized")

	// Create handler
	handler := api.NewHandler(gen, taskManager, sseManager, cfg)

	// Setup router
	router := api.SetupRouter(handler)

	// Create HTTP server
	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	// Shutdown task manager
	taskManager.Shutdown()

	log.Println("Server stopped")
}
