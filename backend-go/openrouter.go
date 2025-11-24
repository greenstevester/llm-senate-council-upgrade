package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// QueryModel queries a single model via OpenRouter API with the given timeout.
// Returns the model's response or an error if the request fails.
func QueryModel(ctx context.Context, model string, messages []OpenRouterMessage, timeout time.Duration) (*OpenRouterResponse, error) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: timeout,
	}

	// Build request payload
	payload := OpenRouterRequest{
		Model:    model,
		Messages: messages,
	}

	// Marshal payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", OpenRouterAPIURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+OpenRouterAPIKey)
	req.Header.Set("Content-Type", "application/json")

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse response
	var apiResponse OpenRouterAPIResponse
	if err := json.Unmarshal(bodyBytes, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract message from response
	if len(apiResponse.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	message := apiResponse.Choices[0].Message
	return &OpenRouterResponse{
		Content:          message.Content,
		ReasoningDetails: message.ReasoningDetails,
	}, nil
}

// QueryModelsParallel queries multiple models in parallel using goroutines.
// Uses errgroup for parallel execution with graceful degradation - failed models
// return nil in the results map while successful models return their responses.
// Returns a map of model names to responses, or an error if all models fail.
func QueryModelsParallel(ctx context.Context, models []string, messages []OpenRouterMessage) (map[string]*OpenRouterResponse, error) {
	// Create errgroup for parallel execution
	g, ctx := errgroup.WithContext(ctx)

	// Results map and mutex for thread-safe writes
	results := make(map[string]*OpenRouterResponse)
	var mu sync.Mutex

	// Launch goroutine for each model
	for _, model := range models {
		model := model // Capture loop variable
		g.Go(func() error {
			// Query the model with 120 second timeout
			response, err := QueryModel(ctx, model, messages, 120*time.Second)

			// Graceful degradation: log error but don't fail entire request
			if err != nil {
				log.Printf("Error querying model %s: %v", model, err)
				mu.Lock()
				results[model] = nil
				mu.Unlock()
				return nil // Don't propagate error, continue with other models
			}

			// Store successful response
			mu.Lock()
			results[model] = response
			mu.Unlock()
			return nil
		})
	}

	// Wait for all goroutines to complete
	if err := g.Wait(); err != nil {
		return nil, err
	}

	return results, nil
}
