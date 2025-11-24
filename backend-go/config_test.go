package main

import (
	"os"
	"testing"
)

// TestLoadConfig tests configuration loading
func TestLoadConfig(t *testing.T) {
	// Save original env
	oldAPIKey := os.Getenv("OPENROUTER_API_KEY")
	defer func() {
		if oldAPIKey != "" {
			os.Setenv("OPENROUTER_API_KEY", oldAPIKey)
		} else {
			os.Unsetenv("OPENROUTER_API_KEY")
		}
	}()

	t.Run("loads API key from environment", func(t *testing.T) {
		// Set test API key
		os.Setenv("OPENROUTER_API_KEY", "test-key-12345")

		// LoadConfig will try to load .env but that's OK if it fails
		// The main thing is it should read from environment
		LoadConfig()

		if OpenRouterAPIKey != "test-key-12345" {
			t.Errorf("API key = %q, want 'test-key-12345'", OpenRouterAPIKey)
		}
	})
}

// TestConfigConstants tests configuration constants
func TestConfigConstants(t *testing.T) {
	// Verify council models are set
	if len(CouncilModels) == 0 {
		t.Error("CouncilModels should not be empty")
	}

	// Verify chairman model is set
	if ChairmanModel == "" {
		t.Error("ChairmanModel should not be empty")
	}

	// Verify API URL is set
	if OpenRouterAPIURL == "" {
		t.Error("OpenRouterAPIURL should not be empty")
	}

	// Verify it's the correct URL
	expectedURL := "https://openrouter.ai/api/v1/chat/completions"
	if OpenRouterAPIURL != expectedURL {
		t.Errorf("OpenRouterAPIURL = %q, want %q", OpenRouterAPIURL, expectedURL)
	}

	// Verify data directory is set
	if DataDir == "" {
		t.Error("DataDir should not be empty")
	}

	expectedDataDir := "data/conversations"
	if DataDir != expectedDataDir {
		t.Errorf("DataDir = %q, want %q", DataDir, expectedDataDir)
	}
}

// TestCouncilModels tests that council models are properly configured
func TestCouncilModels(t *testing.T) {
	expectedModels := []string{
		"openai/gpt-5.1",
		"google/gemini-3-pro-preview",
		"anthropic/claude-sonnet-4.5",
		"x-ai/grok-4",
	}

	if len(CouncilModels) != len(expectedModels) {
		t.Errorf("CouncilModels length = %d, want %d", len(CouncilModels), len(expectedModels))
	}

	for i, expected := range expectedModels {
		if i >= len(CouncilModels) {
			t.Errorf("Missing model at index %d", i)
			continue
		}
		if CouncilModels[i] != expected {
			t.Errorf("CouncilModels[%d] = %q, want %q", i, CouncilModels[i], expected)
		}
	}
}

// TestChairmanModel tests chairman model configuration
func TestChairmanModel(t *testing.T) {
	expected := "google/gemini-3-pro-preview"
	if ChairmanModel != expected {
		t.Errorf("ChairmanModel = %q, want %q", ChairmanModel, expected)
	}
}
