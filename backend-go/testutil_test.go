package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestHelper provides utilities for tests
type TestHelper struct {
	t       *testing.T
	tempDir string
}

// NewTestHelper creates a new test helper
func NewTestHelper(t *testing.T) *TestHelper {
	return &TestHelper{t: t}
}

// CreateTempDir creates a temporary directory for testing
func (h *TestHelper) CreateTempDir() string {
	tempDir, err := os.MkdirTemp("", "llm-council-test-*")
	if err != nil {
		h.t.Fatalf("Failed to create temp dir: %v", err)
	}
	h.tempDir = tempDir
	return tempDir
}

// Cleanup removes the temporary directory
func (h *TestHelper) Cleanup() {
	if h.tempDir != "" {
		os.RemoveAll(h.tempDir)
	}
}

// WriteJSONFile writes JSON data to a file in the temp directory
func (h *TestHelper) WriteJSONFile(filename string, data interface{}) string {
	if h.tempDir == "" {
		h.CreateTempDir()
	}

	path := filepath.Join(h.tempDir, filename)
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		h.t.Fatalf("Failed to marshal JSON: %v", err)
	}

	if err := os.WriteFile(path, jsonData, 0644); err != nil {
		h.t.Fatalf("Failed to write file: %v", err)
	}

	return path
}

// ReadJSONFile reads and unmarshals JSON from a file
func (h *TestHelper) ReadJSONFile(path string, v interface{}) {
	data, err := os.ReadFile(path)
	if err != nil {
		h.t.Fatalf("Failed to read file: %v", err)
	}

	if err := json.Unmarshal(data, v); err != nil {
		h.t.Fatalf("Failed to unmarshal JSON: %v", err)
	}
}

// AssertEqual checks if two values are equal
func (h *TestHelper) AssertEqual(got, want interface{}, message string) {
	if got != want {
		h.t.Errorf("%s: got %v, want %v", message, got, want)
	}
}

// AssertNotNil checks if a value is not nil
func (h *TestHelper) AssertNotNil(v interface{}, message string) {
	if v == nil {
		h.t.Errorf("%s: expected non-nil value", message)
	}
}

// AssertNil checks if a value is nil
func (h *TestHelper) AssertNil(v interface{}, message string) {
	if v != nil && !isNil(v) {
		h.t.Errorf("%s: expected nil, got %v", message, v)
	}
}

// isNil checks if an interface value is nil (handles typed nil pointers)
func isNil(v interface{}) bool {
	if v == nil {
		return true
	}
	// Use type assertion to check for nil pointer
	switch v := v.(type) {
	case *Conversation:
		return v == nil
	default:
		return false
	}
}

// AssertNoError checks if an error is nil
func (h *TestHelper) AssertNoError(err error, message string) {
	if err != nil {
		h.t.Errorf("%s: unexpected error: %v", message, err)
	}
}

// AssertError checks if an error is not nil
func (h *TestHelper) AssertError(err error, message string) {
	if err == nil {
		h.t.Errorf("%s: expected error, got nil", message)
	}
}

// MockOpenRouterServer creates a mock HTTP server for OpenRouter API
func MockOpenRouterServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

// CreateMockOpenRouterHandler creates a handler that returns successful responses
func CreateMockOpenRouterHandler(t *testing.T, response string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		if r.Header.Get("Authorization") == "" {
			t.Errorf("Missing Authorization header")
		}

		// Return mock response
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
						Content: response,
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(apiResponse)
	}
}

// CreateMockOpenRouterErrorHandler creates a handler that returns errors
func CreateMockOpenRouterErrorHandler(statusCode int, errorMsg string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		w.Write([]byte(errorMsg))
	}
}

// SampleConversation creates a sample conversation for testing
func SampleConversation(id string) *Conversation {
	return &Conversation{
		ID:        id,
		CreatedAt: testTime(),
		Title:     "Test Conversation",
		Messages: []Message{
			{
				Role:    "user",
				Content: "What is Go?",
			},
			{
				Role: "assistant",
				Stage1: []Stage1Response{
					{Model: "test/model1", Response: "Go is a programming language."},
					{Model: "test/model2", Response: "Go is developed by Google."},
				},
				Stage2: []Stage2Ranking{
					{
						Model:         "test/model1",
						Ranking:       "FINAL RANKING:\n1. Response B\n2. Response A",
						ParsedRanking: []string{"Response B", "Response A"},
					},
				},
				Stage3: &Stage3Response{
					Model:    "test/chairman",
					Response: "Go is a programming language developed by Google.",
				},
			},
		},
	}
}

// testTime returns a fixed time for testing
func testTime() time.Time {
	return time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
}
