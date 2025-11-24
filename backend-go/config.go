package main

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
)

// Configuration constants
var (
	// OpenRouterAPIKey is the API key for OpenRouter
	OpenRouterAPIKey string

	// CouncilModels is the list of models to query in parallel
	CouncilModels = []string{
		"openai/gpt-5.1",
		"google/gemini-3-pro-preview",
		"anthropic/claude-sonnet-4.5",
		"x-ai/grok-4",
	}

	// ChairmanModel is the model used for final synthesis
	ChairmanModel = "google/gemini-3-pro-preview"

	// OpenRouterAPIURL is the endpoint for OpenRouter API
	OpenRouterAPIURL = "https://openrouter.ai/api/v1/chat/completions"

	// DataDir is the directory for conversation storage
	DataDir = "data/conversations"

	// Timeout constants
	ModelQueryTimeout = 120 * time.Second
	TitleGenTimeout   = 30 * time.Second

	// CORS allowed origins (configurable via environment)
	// In development (empty/default), allows any localhost port
	// In production, set CORS_ALLOWED_ORIGINS environment variable
	CORSAllowedOrigins = []string{}

	// MaxRequestBodySize is the maximum allowed request body size (1MB)
	MaxRequestBodySize int64 = 1 << 20

	// BillsCacheTTL is the time-to-live for bills cache (default 5 minutes)
	BillsCacheTTL = 5 * time.Minute
)

// LoadConfig loads configuration from environment variables
func LoadConfig() {
	// Load .env file - try multiple locations
	envLocations := []string{
		".env",        // Current directory
		"../.env",     // Parent directory
	}

	// Try to find and load .env file
	envLoaded := false
	for _, envPath := range envLocations {
		absPath, err := filepath.Abs(envPath)
		if err != nil {
			continue
		}

		if _, err := os.Stat(absPath); err == nil {
			if err := godotenv.Load(absPath); err == nil {
				log.Printf("Loaded .env from: %s", absPath)
				envLoaded = true
				break
			}
		}
	}

	if !envLoaded {
		log.Printf("Warning: .env file not found in any expected location")
	}

	// Get OpenRouter API key
	OpenRouterAPIKey = os.Getenv("OPENROUTER_API_KEY")
	if OpenRouterAPIKey == "" {
		log.Fatal("OPENROUTER_API_KEY environment variable is required")
	}

	// Load CORS origins from environment if provided
	if corsOrigins := os.Getenv("CORS_ALLOWED_ORIGINS"); corsOrigins != "" {
		CORSAllowedOrigins = []string{}
		for _, origin := range filepath.SplitList(corsOrigins) {
			if origin != "" {
				CORSAllowedOrigins = append(CORSAllowedOrigins, origin)
			}
		}
	}

	log.Println("Configuration loaded successfully")
}
