package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	Server  ServerConfig
	GitHub  GitHubConfig
	LLM     LLMConfig
	Task    TaskConfig
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Port string
	Host string
}

// GitHubConfig holds GitHub-related configuration
type GitHubConfig struct {
	Token string
	Owner string
}

// LLMConfig holds LLM-related configuration
type LLMConfig struct {
	DeepSeekAPIKey  string
	DeepSeekBaseURL string
	OpenAIAPIKey    string
	OpenAIBaseURL   string
	DefaultModel    string
}

// TaskConfig holds task-related configuration
type TaskConfig struct {
	MaxConcurrentTasks int
	TaskTimeout        int
	TempDir            string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Try to load .env file, but don't fail if it doesn't exist
	_ = godotenv.Load()

	cfg := &Config{
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
		},
		GitHub: GitHubConfig{
			Token: getEnv("GITHUB_TOKEN", ""),
			Owner: getEnv("GITHUB_OWNER", ""),
		},
		LLM: LLMConfig{
			DeepSeekAPIKey:  getEnv("DEEPSEEK_API_KEY", ""),
			DeepSeekBaseURL: getEnv("DEEPSEEK_BASE_URL", "https://api.deepseek.com"),
			OpenAIAPIKey:    getEnv("OPENAI_API_KEY", ""),
			OpenAIBaseURL:   getEnv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
			DefaultModel:    getEnv("DEFAULT_MODEL", "deepseek"),
		},
		Task: TaskConfig{
			MaxConcurrentTasks: getEnvAsInt("MAX_CONCURRENT_TASKS", 5),
			TaskTimeout:        getEnvAsInt("TASK_TIMEOUT", 600),
			TempDir:            getEnv("TEMP_DIR", "./tmp"),
		},
	}

	// Validate required fields
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks if required configuration fields are set
func (c *Config) Validate() error {
	if c.GitHub.Token == "" {
		return fmt.Errorf("GITHUB_TOKEN is required")
	}

	// At least one LLM API key must be set
	if c.LLM.DeepSeekAPIKey == "" && c.LLM.OpenAIAPIKey == "" {
		return fmt.Errorf("at least one LLM API key (DEEPSEEK_API_KEY or OPENAI_API_KEY) is required")
	}

	return nil
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt gets an environment variable as int or returns a default value
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
