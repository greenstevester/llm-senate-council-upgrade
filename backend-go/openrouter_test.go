package main

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// TestQueryModel tests QueryModel with mock server
func TestQueryModel(t *testing.T) {
	// Save original config
	oldAPIURL := OpenRouterAPIURL
	oldAPIKey := OpenRouterAPIKey
	defer func() {
		OpenRouterAPIURL = oldAPIURL
		OpenRouterAPIKey = oldAPIKey
	}()

	t.Run("successful query", func(t *testing.T) {
		mockServer := MockOpenRouterServer(t, CreateMockOpenRouterHandler(t, "Test response content"))
		defer mockServer.Close()

		OpenRouterAPIURL = mockServer.URL
		OpenRouterAPIKey = "test-key"

		messages := []OpenRouterMessage{
			{Role: "user", Content: "Test question"},
		}

		ctx := context.Background()
		response, err := QueryModel(ctx, "test/model", messages, 10*time.Second)

		if err != nil {
			t.Fatalf("QueryModel failed: %v", err)
		}
		if response == nil {
			t.Fatal("Response should not be nil")
		}
		if response.Content != "Test response content" {
			t.Errorf("Content = %q, want 'Test response content'", response.Content)
		}
	})

	t.Run("API error response", func(t *testing.T) {
		mockServer := MockOpenRouterServer(t, CreateMockOpenRouterErrorHandler(500, "Internal server error"))
		defer mockServer.Close()

		OpenRouterAPIURL = mockServer.URL
		OpenRouterAPIKey = "test-key"

		messages := []OpenRouterMessage{
			{Role: "user", Content: "Test"},
		}

		ctx := context.Background()
		_, err := QueryModel(ctx, "test/model", messages, 10*time.Second)

		if err == nil {
			t.Error("Expected error for 500 response, got nil")
		}
	})

	t.Run("timeout", func(t *testing.T) {
		// Create server that delays response
		slowHandler := func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(2 * time.Second)
			w.WriteHeader(http.StatusOK)
		}
		mockServer := MockOpenRouterServer(t, slowHandler)
		defer mockServer.Close()

		OpenRouterAPIURL = mockServer.URL
		OpenRouterAPIKey = "test-key"

		messages := []OpenRouterMessage{
			{Role: "user", Content: "Test"},
		}

		ctx := context.Background()
		_, err := QueryModel(ctx, "test/model", messages, 100*time.Millisecond)

		if err == nil {
			t.Error("Expected timeout error, got nil")
		}
	})

	t.Run("invalid JSON response", func(t *testing.T) {
		invalidHandler := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("{ invalid json }"))
		}
		mockServer := MockOpenRouterServer(t, invalidHandler)
		defer mockServer.Close()

		OpenRouterAPIURL = mockServer.URL
		OpenRouterAPIKey = "test-key"

		messages := []OpenRouterMessage{
			{Role: "user", Content: "Test"},
		}

		ctx := context.Background()
		_, err := QueryModel(ctx, "test/model", messages, 10*time.Second)

		if err == nil {
			t.Error("Expected error for invalid JSON, got nil")
		}
	})

	t.Run("empty choices in response", func(t *testing.T) {
		emptyHandler := func(w http.ResponseWriter, r *http.Request) {
			apiResponse := OpenRouterAPIResponse{
				Choices: []struct {
					Message struct {
						Content          string      `json:"content"`
						ReasoningDetails interface{} `json:"reasoning_details,omitempty"`
					} `json:"message"`
				}{},
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(apiResponse)
		}
		mockServer := MockOpenRouterServer(t, emptyHandler)
		defer mockServer.Close()

		OpenRouterAPIURL = mockServer.URL
		OpenRouterAPIKey = "test-key"

		messages := []OpenRouterMessage{
			{Role: "user", Content: "Test"},
		}

		ctx := context.Background()
		_, err := QueryModel(ctx, "test/model", messages, 10*time.Second)

		if err == nil {
			t.Error("Expected error for empty choices, got nil")
		}
	})
}

// TestQueryModelsParallel tests parallel model querying
func TestQueryModelsParallel(t *testing.T) {
	// Save original config
	oldAPIURL := OpenRouterAPIURL
	oldAPIKey := OpenRouterAPIKey
	defer func() {
		OpenRouterAPIURL = oldAPIURL
		OpenRouterAPIKey = oldAPIKey
	}()

	t.Run("all models succeed", func(t *testing.T) {
		mockServer := MockOpenRouterServer(t, CreateMockOpenRouterHandler(t, "Success response"))
		defer mockServer.Close()

		OpenRouterAPIURL = mockServer.URL
		OpenRouterAPIKey = "test-key"

		models := []string{"model/a", "model/b", "model/c"}
		messages := []OpenRouterMessage{
			{Role: "user", Content: "Test"},
		}

		ctx := context.Background()
		results, err := QueryModelsParallel(ctx, models, messages)

		if err != nil {
			t.Fatalf("QueryModelsParallel failed: %v", err)
		}
		if len(results) != 3 {
			t.Errorf("Expected 3 results, got %d", len(results))
		}

		// All should be successful
		for model, response := range results {
			if response == nil {
				t.Errorf("Model %s returned nil", model)
			} else if response.Content != "Success response" {
				t.Errorf("Model %s: content = %q, want 'Success response'", model, response.Content)
			}
		}
	})

	t.Run("graceful degradation - some models fail", func(t *testing.T) {
		// Handler that fails for specific model
		failingHandler := func(w http.ResponseWriter, r *http.Request) {
			var req OpenRouterRequest
			json.NewDecoder(r.Body).Decode(&req)

			if req.Model == "model/fail" {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			apiResponse := OpenRouterAPIResponse{
				Choices: []struct {
					Message struct {
						Content          string      `json:"content"`
						ReasoningDetails interface{} `json:"reasoning_details,omitempty"`
					} `json:"message"`
				}{
					{
						Message: struct {
							Content          string      `json:"content"`
							ReasoningDetails interface{} `json:"reasoning_details,omitempty"`
						}{
							Content: "Success",
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(apiResponse)
		}

		mockServer := MockOpenRouterServer(t, failingHandler)
		defer mockServer.Close()

		OpenRouterAPIURL = mockServer.URL
		OpenRouterAPIKey = "test-key"

		models := []string{"model/success", "model/fail"}
		messages := []OpenRouterMessage{
			{Role: "user", Content: "Test"},
		}

		ctx := context.Background()
		results, err := QueryModelsParallel(ctx, models, messages)

		// Should not error - graceful degradation
		if err != nil {
			t.Fatalf("QueryModelsParallel should not error: %v", err)
		}

		// Check successful model
		if results["model/success"] == nil {
			t.Error("Successful model should have response")
		}

		// Check failed model
		if results["model/fail"] != nil {
			t.Error("Failed model should have nil response")
		}
	})

	t.Run("empty model list", func(t *testing.T) {
		mockServer := MockOpenRouterServer(t, CreateMockOpenRouterHandler(t, "Test"))
		defer mockServer.Close()

		OpenRouterAPIURL = mockServer.URL
		OpenRouterAPIKey = "test-key"

		models := []string{}
		messages := []OpenRouterMessage{
			{Role: "user", Content: "Test"},
		}

		ctx := context.Background()
		results, err := QueryModelsParallel(ctx, models, messages)

		if err != nil {
			t.Fatalf("Should handle empty model list: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("Expected 0 results for empty model list, got %d", len(results))
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		// Create handler that delays
		slowHandler := func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(1 * time.Second)
			w.WriteHeader(http.StatusOK)
		}
		mockServer := MockOpenRouterServer(t, slowHandler)
		defer mockServer.Close()

		OpenRouterAPIURL = mockServer.URL
		OpenRouterAPIKey = "test-key"

		models := []string{"model/slow"}
		messages := []OpenRouterMessage{
			{Role: "user", Content: "Test"},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		results, err := QueryModelsParallel(ctx, models, messages)

		// Should handle timeout gracefully
		if err != nil {
			t.Fatalf("Should handle context cancellation gracefully: %v", err)
		}
		// Result should be nil due to timeout
		if results["model/slow"] != nil {
			t.Error("Expected nil result for timed out model")
		}
	})
}

// TestOpenRouterMessageJSON tests JSON marshaling of OpenRouterMessage
func TestOpenRouterMessageJSON(t *testing.T) {
	msg := OpenRouterMessage{
		Role:    "user",
		Content: "Hello, world!",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded OpenRouterMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Role != msg.Role {
		t.Errorf("Role mismatch: got %s, want %s", decoded.Role, msg.Role)
	}
	if decoded.Content != msg.Content {
		t.Errorf("Content mismatch: got %s, want %s", decoded.Content, msg.Content)
	}
}

// TestOpenRouterRequestJSON tests JSON marshaling of OpenRouterRequest
func TestOpenRouterRequestJSON(t *testing.T) {
	req := OpenRouterRequest{
		Model: "test/model",
		Messages: []OpenRouterMessage{
			{Role: "user", Content: "test"},
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded OpenRouterRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Model != req.Model {
		t.Errorf("Model mismatch: got %s, want %s", decoded.Model, req.Model)
	}
	if len(decoded.Messages) != len(req.Messages) {
		t.Errorf("Messages length mismatch: got %d, want %d", len(decoded.Messages), len(req.Messages))
	}
}
