// +build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// Simple test program to verify the OpenRouter client works
// Run with: go run test_openrouter_client.go config.go models.go openrouter.go
func main() {
	fmt.Println("=== OpenRouter Client Test ===\n")

	// Load .env from parent directory
	if err := godotenv.Load("../.env"); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Get API key
	OpenRouterAPIKey = os.Getenv("OPENROUTER_API_KEY")
	if OpenRouterAPIKey == "" {
		log.Fatal("OPENROUTER_API_KEY environment variable is required")
	}

	// Create test messages
	messages := []OpenRouterMessage{
		{Role: "user", Content: "Say hello in exactly 5 words."},
	}

	ctx := context.Background()

	// Test 1: Single model query
	fmt.Println("Test 1: Querying single model (gemini-2.5-flash)...")
	start := time.Now()
	response, err := QueryModel(ctx, "google/gemini-2.5-flash", messages, 30*time.Second)
	elapsed := time.Since(start)

	if err != nil {
		log.Fatalf("❌ Single query failed: %v", err)
	}

	fmt.Printf("✅ Success! (%.2fs)\n", elapsed.Seconds())
	fmt.Printf("Response: %s\n\n", response.Content)

	// Test 2: Parallel model queries
	fmt.Println("Test 2: Querying multiple models in parallel...")
	testModels := []string{
		"google/gemini-2.5-flash",
		"anthropic/claude-3.5-haiku",
		"openai/gpt-4o-mini",
	}

	start = time.Now()
	responses, err := QueryModelsParallel(ctx, testModels, messages)
	elapsed = time.Since(start)

	if err != nil {
		log.Fatalf("❌ Parallel query failed: %v", err)
	}

	fmt.Printf("✅ Success! (%.2fs)\n", elapsed.Seconds())
	fmt.Println("\nResults:")
	successCount := 0
	for model, resp := range responses {
		if resp != nil {
			fmt.Printf("  ✅ %s: %s\n", model, resp.Content)
			successCount++
		} else {
			fmt.Printf("  ❌ %s: FAILED\n", model)
		}
	}

	fmt.Printf("\n=== Test Complete: %d/%d models succeeded ===\n", successCount, len(testModels))
}
