package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)
}

// TestHealthCheck tests the health check endpoint
func TestHealthCheck(t *testing.T) {
	router := gin.New()
	router.GET("/", healthCheck)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("Status = %v, want 'ok'", response["status"])
	}
	if response["service"] != "LLM Council API" {
		t.Errorf("Service = %v, want 'LLM Council API'", response["service"])
	}
}

// TestListConversationsHandler tests listing conversations
func TestListConversationsHandler(t *testing.T) {
	helper := NewTestHelper(t)
	tempDir := helper.CreateTempDir()
	defer helper.Cleanup()

	oldDataDir := DataDir
	DataDir = tempDir
	defer func() { DataDir = oldDataDir }()

	// Create test conversations
	CreateConversation("test1")
	CreateConversation("test2")

	router := gin.New()
	router.GET("/api/conversations", listConversationsHandler)

	req := httptest.NewRequest("GET", "/api/conversations", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	var conversations []ConversationMetadata
	if err := json.Unmarshal(w.Body.Bytes(), &conversations); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(conversations) != 2 {
		t.Errorf("Got %d conversations, want 2", len(conversations))
	}
}

// TestCreateConversationHandler tests conversation creation
func TestCreateConversationHandler(t *testing.T) {
	helper := NewTestHelper(t)
	tempDir := helper.CreateTempDir()
	defer helper.Cleanup()

	oldDataDir := DataDir
	DataDir = tempDir
	defer func() { DataDir = oldDataDir }()

	router := gin.New()
	router.POST("/api/conversations", createConversationHandler)

	req := httptest.NewRequest("POST", "/api/conversations", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	var conversation Conversation
	if err := json.Unmarshal(w.Body.Bytes(), &conversation); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if conversation.ID == "" {
		t.Error("Conversation ID should not be empty")
	}
	if conversation.Title != "New Conversation" {
		t.Errorf("Title = %q, want 'New Conversation'", conversation.Title)
	}
}

// TestGetConversationHandler tests getting a specific conversation
func TestGetConversationHandler(t *testing.T) {
	helper := NewTestHelper(t)
	tempDir := helper.CreateTempDir()
	defer helper.Cleanup()

	oldDataDir := DataDir
	DataDir = tempDir
	defer func() { DataDir = oldDataDir }()

	// Create test conversation
	CreateConversation("test-get")

	router := gin.New()
	router.GET("/api/conversations/:id", getConversationHandler)

	t.Run("existing conversation", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/conversations/test-get", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
		}

		var conversation Conversation
		if err := json.Unmarshal(w.Body.Bytes(), &conversation); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if conversation.ID != "test-get" {
			t.Errorf("ID = %q, want 'test-get'", conversation.ID)
		}
	})

	t.Run("non-existent conversation", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/conversations/non-existent", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})
}

// TestSendMessageHandler tests sending a message
func TestSendMessageHandler(t *testing.T) {
	helper := NewTestHelper(t)
	tempDir := helper.CreateTempDir()
	defer helper.Cleanup()

	oldDataDir := DataDir
	oldAPIURL := OpenRouterAPIURL
	oldAPIKey := OpenRouterAPIKey
	oldModels := CouncilModels
	oldChairman := ChairmanModel
	defer func() {
		DataDir = oldDataDir
		OpenRouterAPIURL = oldAPIURL
		OpenRouterAPIKey = oldAPIKey
		CouncilModels = oldModels
		ChairmanModel = oldChairman
	}()

	DataDir = tempDir
	CouncilModels = []string{"model/a", "model/b"}
	ChairmanModel = "model/chairman"

	// Create mock server
	requestCount := 0
	mockHandler := func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		var response string
		if requestCount <= 2 {
			response = "Response " + string(rune('A'+requestCount-1))
		} else if requestCount <= 4 {
			response = "FINAL RANKING:\n1. Response B\n2. Response A"
		} else {
			response = "Final synthesis"
		}

		apiResponse := OpenRouterAPIResponse{
			Choices: []struct {
				Message struct {
					Content          string      `json:"content"`
					ReasoningDetails interface{} `json:"reasoning_details,omitempty"`
				} `json:"message"`
			}{
				{Message: struct {
					Content          string      `json:"content"`
					ReasoningDetails interface{} `json:"reasoning_details,omitempty"`
				}{Content: response}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(apiResponse)
	}

	mockServer := MockOpenRouterServer(t, mockHandler)
	defer mockServer.Close()

	OpenRouterAPIURL = mockServer.URL
	OpenRouterAPIKey = "test-key"

	// Create conversation
	CreateConversation("test-send")

	router := gin.New()
	router.POST("/api/conversations/:id/message", sendMessageHandler)

	t.Run("successful message send", func(t *testing.T) {
		requestBody := map[string]string{
			"content": "What is Go?",
		}
		bodyBytes, _ := json.Marshal(requestBody)

		req := httptest.NewRequest("POST", "/api/conversations/test-send/message", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
		}

		var response SendMessageResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if len(response.Stage1) == 0 {
			t.Error("Stage1 should not be empty")
		}
		if len(response.Stage2) == 0 {
			t.Error("Stage2 should not be empty")
		}
		if response.Stage3.Response == "" {
			t.Error("Stage3 response should not be empty")
		}
	})

	t.Run("invalid request body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/conversations/test-send/message", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("non-existent conversation", func(t *testing.T) {
		requestBody := map[string]string{
			"content": "Test",
		}
		bodyBytes, _ := json.Marshal(requestBody)

		req := httptest.NewRequest("POST", "/api/conversations/non-existent/message", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})
}

// TestSendSSEEvent tests SSE event sending
func TestSendSSEEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	data := gin.H{"type": "test", "message": "hello"}
	sendSSEEvent(c, data)

	// Check that data was written
	body := w.Body.String()
	if body == "" {
		t.Error("Expected SSE data to be written")
	}

	// Should contain "data:" prefix
	if len(body) < 5 || body[:5] != "data:" {
		t.Errorf("Expected SSE format with 'data:' prefix, got: %s", body)
	}
}

// TestSendSSEError tests SSE error sending
func TestSendSSEError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	sendSSEError(c, "test error message")

	body := w.Body.String()
	if body == "" {
		t.Error("Expected SSE error data to be written")
	}

	// Should contain error type
	var eventData map[string]interface{}
	// Extract JSON from SSE format (after "data: " prefix)
	jsonStr := body[6:] // Skip "data: "
	if err := json.Unmarshal([]byte(jsonStr), &eventData); err == nil {
		if eventData["type"] != "error" {
			t.Errorf("Expected type 'error', got %v", eventData["type"])
		}
		if eventData["message"] != "test error message" {
			t.Errorf("Expected message 'test error message', got %v", eventData["message"])
		}
	}
}

// TestSendMessageStreamHandler tests the SSE streaming endpoint
func TestSendMessageStreamHandler(t *testing.T) {
	helper := NewTestHelper(t)
	tempDir := helper.CreateTempDir()
	defer helper.Cleanup()

	oldDataDir := DataDir
	oldAPIURL := OpenRouterAPIURL
	oldAPIKey := OpenRouterAPIKey
	oldModels := CouncilModels
	oldChairman := ChairmanModel
	defer func() {
		DataDir = oldDataDir
		OpenRouterAPIURL = oldAPIURL
		OpenRouterAPIKey = oldAPIKey
		CouncilModels = oldModels
		ChairmanModel = oldChairman
	}()

	DataDir = tempDir
	CouncilModels = []string{"model/a"}
	ChairmanModel = "model/chairman"

	// Create simple mock server
	mockServer := MockOpenRouterServer(t, CreateMockOpenRouterHandler(t, "Test response"))
	defer mockServer.Close()

	OpenRouterAPIURL = mockServer.URL
	OpenRouterAPIKey = "test-key"

	// Create conversation
	CreateConversation("test-stream")

	router := gin.New()
	router.POST("/api/conversations/:id/message/stream", sendMessageStreamHandler)

	t.Run("stream with valid request", func(t *testing.T) {
		requestBody := map[string]string{
			"content": "Test question",
		}
		bodyBytes, _ := json.Marshal(requestBody)

		req := httptest.NewRequest("POST", "/api/conversations/test-stream/message/stream", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Should succeed
		if w.Code != http.StatusOK {
			t.Errorf("Status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
		}

		// Check that it's SSE format
		if w.Header().Get("Content-Type") != "text/event-stream" {
			t.Errorf("Content-Type = %s, want 'text/event-stream'", w.Header().Get("Content-Type"))
		}

		// Check that response contains event data
		body := w.Body.String()
		if body == "" {
			t.Error("Expected SSE stream data")
		}
	})

	t.Run("stream with invalid request", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/conversations/test-stream/message/stream", bytes.NewReader([]byte("invalid")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("stream with non-existent conversation", func(t *testing.T) {
		requestBody := map[string]string{
			"content": "Test",
		}
		bodyBytes, _ := json.Marshal(requestBody)

		req := httptest.NewRequest("POST", "/api/conversations/non-existent/message/stream", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})
}

// TestRunFullCouncilErrorHandling tests error cases in the full council flow
func TestRunFullCouncilErrorHandling(t *testing.T) {
	// Save original config
	oldAPIURL := OpenRouterAPIURL
	oldAPIKey := OpenRouterAPIKey
	oldModels := CouncilModels
	defer func() {
		OpenRouterAPIURL = oldAPIURL
		OpenRouterAPIKey = oldAPIKey
		CouncilModels = oldModels
	}()

	// Create mock server that always fails
	failingServer := MockOpenRouterServer(t, CreateMockOpenRouterErrorHandler(500, "Server error"))
	defer failingServer.Close()

	OpenRouterAPIURL = failingServer.URL
	OpenRouterAPIKey = "test-key"
	CouncilModels = []string{"model/a"}

	ctx := context.Background()
	stage1, stage2, stage3, metadata, err := RunFullCouncil(ctx, "Test question")

	// When all models fail, we should get an error now
	if err == nil {
		t.Error("Expected error when all models fail, got nil")
	}

	// Results should be nil/empty on error
	if stage1 != nil {
		t.Errorf("Expected nil stage1 on error, got: %v", stage1)
	}
	if stage2 != nil {
		t.Errorf("Expected nil stage2 on error, got: %v", stage2)
	}

	_ = stage3
	_ = metadata
}

// TestListConversationsHandlerError tests error handling in list conversations
func TestListConversationsHandlerError(t *testing.T) {
	oldDataDir := DataDir
	// Set to invalid directory that will cause error
	DataDir = "/invalid/path/that/does/not/exist/and/cannot/be/created"
	defer func() { DataDir = oldDataDir }()

	router := gin.New()
	router.GET("/api/conversations", listConversationsHandler)

	req := httptest.NewRequest("GET", "/api/conversations", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// TestCreateConversationHandlerError tests error handling in create conversation
func TestCreateConversationHandlerError(t *testing.T) {
	oldDataDir := DataDir
	// Set to invalid directory
	DataDir = "/invalid/path/that/does/not/exist"
	defer func() { DataDir = oldDataDir }()

	router := gin.New()
	router.POST("/api/conversations", createConversationHandler)

	req := httptest.NewRequest("POST", "/api/conversations", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// TestGetConversationHandlerError tests error handling in get conversation
func TestGetConversationHandlerError(t *testing.T) {
	helper := NewTestHelper(t)
	tempDir := helper.CreateTempDir()
	defer helper.Cleanup()

	oldDataDir := DataDir
	DataDir = tempDir
	defer func() { DataDir = oldDataDir }()

	// Create a conversation file with invalid JSON to cause parsing error
	os.WriteFile(GetConversationPath("invalid"), []byte("{invalid json}"), 0644)

	router := gin.New()
	router.GET("/api/conversations/:id", getConversationHandler)

	req := httptest.NewRequest("GET", "/api/conversations/invalid", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// TestSendMessageHandlerGetConversationError tests error when getting conversation fails
func TestSendMessageHandlerGetConversationError(t *testing.T) {
	helper := NewTestHelper(t)
	tempDir := helper.CreateTempDir()
	defer helper.Cleanup()

	oldDataDir := DataDir
	DataDir = tempDir
	defer func() { DataDir = oldDataDir }()

	// Create conversation with invalid JSON
	os.WriteFile(GetConversationPath("invalid"), []byte("{invalid}"), 0644)

	router := gin.New()
	router.POST("/api/conversations/:id/message", sendMessageHandler)

	requestBody := map[string]string{"content": "Test"}
	bodyBytes, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/api/conversations/invalid/message", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// TestSendMessageStreamHandlerGetConversationError tests stream error handling
func TestSendMessageStreamHandlerGetConversationError(t *testing.T) {
	helper := NewTestHelper(t)
	tempDir := helper.CreateTempDir()
	defer helper.Cleanup()

	oldDataDir := DataDir
	DataDir = tempDir
	defer func() { DataDir = oldDataDir }()

	// Create conversation with invalid JSON
	os.WriteFile(GetConversationPath("invalid"), []byte("{invalid}"), 0644)

	router := gin.New()
	router.POST("/api/conversations/:id/message/stream", sendMessageStreamHandler)

	requestBody := map[string]string{"content": "Test"}
	bodyBytes, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/api/conversations/invalid/message/stream", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// TestSendMessageHandlerAddUserMessageError tests error when adding user message fails
func TestSendMessageHandlerAddUserMessageError(t *testing.T) {
	helper := NewTestHelper(t)
	tempDir := helper.CreateTempDir()
	defer helper.Cleanup()

	oldDataDir := DataDir
	DataDir = tempDir
	defer func() { DataDir = oldDataDir }()

	// Create conversation then make directory read-only
	CreateConversation("readonly-test")
	// Make the conversation file read-only
	os.Chmod(GetConversationPath("readonly-test"), 0444)

	router := gin.New()
	router.POST("/api/conversations/:id/message", sendMessageHandler)

	requestBody := map[string]string{"content": "Test"}
	bodyBytes, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/api/conversations/readonly-test/message", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}
