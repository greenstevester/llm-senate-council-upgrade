package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestEnsureDataDir tests directory creation
func TestEnsureDataDir(t *testing.T) {
	helper := NewTestHelper(t)
	tempDir := helper.CreateTempDir()
	defer helper.Cleanup()

	// Override DataDir for testing
	oldDataDir := DataDir
	DataDir = filepath.Join(tempDir, "test-conversations")
	defer func() { DataDir = oldDataDir }()

	// Test creating directory
	err := EnsureDataDir()
	helper.AssertNoError(err, "EnsureDataDir should succeed")

	// Verify directory exists
	if _, err := os.Stat(DataDir); os.IsNotExist(err) {
		t.Errorf("Directory was not created: %s", DataDir)
	}

	// Test that calling again doesn't error
	err = EnsureDataDir()
	helper.AssertNoError(err, "EnsureDataDir should be idempotent")
}

// TestGetConversationPath tests path generation
func TestGetConversationPath(t *testing.T) {
	oldDataDir := DataDir
	DataDir = "/test/data"
	defer func() { DataDir = oldDataDir }()

	tests := []struct {
		id       string
		expected string
	}{
		{"abc-123", "/test/data/abc-123.json"},
		{"test", "/test/data/test.json"},
		{"", "/test/data/.json"},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			path := GetConversationPath(tt.id)
			if path != tt.expected {
				t.Errorf("GetConversationPath(%q) = %q, want %q", tt.id, path, tt.expected)
			}
		})
	}
}

// TestCreateConversation tests creating a new conversation
func TestCreateConversation(t *testing.T) {
	helper := NewTestHelper(t)
	tempDir := helper.CreateTempDir()
	defer helper.Cleanup()

	oldDataDir := DataDir
	DataDir = filepath.Join(tempDir, "conversations")
	defer func() { DataDir = oldDataDir }()

	// Create conversation
	conv, err := CreateConversation("test-id-123")
	helper.AssertNoError(err, "CreateConversation should succeed")
	helper.AssertNotNil(conv, "Conversation should not be nil")

	// Verify fields
	if conv.ID != "test-id-123" {
		t.Errorf("ID = %q, want %q", conv.ID, "test-id-123")
	}
	if conv.Title != "New Conversation" {
		t.Errorf("Title = %q, want %q", conv.Title, "New Conversation")
	}
	if len(conv.Messages) != 0 {
		t.Errorf("Expected empty messages, got %d", len(conv.Messages))
	}

	// Verify file was created
	path := GetConversationPath("test-id-123")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("Conversation file was not created: %s", path)
	}
}

// TestGetConversation tests retrieving a conversation
func TestGetConversation(t *testing.T) {
	helper := NewTestHelper(t)
	tempDir := helper.CreateTempDir()
	defer helper.Cleanup()

	oldDataDir := DataDir
	DataDir = tempDir
	defer func() { DataDir = oldDataDir }()

	// Create sample conversation file
	sampleConv := SampleConversation("test-get-123")
	jsonData, _ := json.MarshalIndent(sampleConv, "", "  ")
	os.WriteFile(filepath.Join(tempDir, "test-get-123.json"), jsonData, 0644)

	// Test retrieving existing conversation
	conv, err := GetConversation("test-get-123")
	helper.AssertNoError(err, "GetConversation should succeed")
	helper.AssertNotNil(conv, "Conversation should not be nil")

	if conv.ID != "test-get-123" {
		t.Errorf("ID = %q, want %q", conv.ID, "test-get-123")
	}
	if conv.Title != sampleConv.Title {
		t.Errorf("Title = %q, want %q", conv.Title, sampleConv.Title)
	}

	// Test retrieving non-existent conversation
	conv, err = GetConversation("non-existent")
	helper.AssertNoError(err, "GetConversation should not error for non-existent")
	helper.AssertNil(conv, "Non-existent conversation should return nil")
}

// TestGetConversationInvalidJSON tests handling of invalid JSON
func TestGetConversationInvalidJSON(t *testing.T) {
	helper := NewTestHelper(t)
	tempDir := helper.CreateTempDir()
	defer helper.Cleanup()

	oldDataDir := DataDir
	DataDir = tempDir
	defer func() { DataDir = oldDataDir }()

	// Create invalid JSON file
	os.WriteFile(filepath.Join(tempDir, "invalid.json"), []byte("{ invalid json"), 0644)

	// Test retrieving conversation with invalid JSON
	_, err := GetConversation("invalid")
	helper.AssertError(err, "Should error on invalid JSON")
}

// TestSaveConversation tests saving a conversation
func TestSaveConversation(t *testing.T) {
	helper := NewTestHelper(t)
	tempDir := helper.CreateTempDir()
	defer helper.Cleanup()

	oldDataDir := DataDir
	DataDir = tempDir
	defer func() { DataDir = oldDataDir }()

	// Create conversation
	conv := &Conversation{
		ID:        "save-test",
		CreatedAt: time.Now(),
		Title:     "Save Test",
		Messages:  []Message{},
	}

	// Save conversation
	err := SaveConversation(conv)
	helper.AssertNoError(err, "SaveConversation should succeed")

	// Verify file exists and can be read back
	path := GetConversationPath("save-test")
	data, err := os.ReadFile(path)
	helper.AssertNoError(err, "Should be able to read saved file")

	// Unmarshal and verify
	var loaded Conversation
	err = json.Unmarshal(data, &loaded)
	helper.AssertNoError(err, "Should be able to unmarshal saved data")

	if loaded.ID != conv.ID {
		t.Errorf("Loaded ID = %q, want %q", loaded.ID, conv.ID)
	}
	if loaded.Title != conv.Title {
		t.Errorf("Loaded Title = %q, want %q", loaded.Title, conv.Title)
	}
}

// TestListConversations tests listing all conversations
func TestListConversations(t *testing.T) {
	helper := NewTestHelper(t)
	tempDir := helper.CreateTempDir()
	defer helper.Cleanup()

	oldDataDir := DataDir
	DataDir = tempDir
	defer func() { DataDir = oldDataDir }()

	// Test empty directory
	conversations, err := ListConversations()
	helper.AssertNoError(err, "ListConversations should succeed on empty dir")
	if len(conversations) != 0 {
		t.Errorf("Expected 0 conversations, got %d", len(conversations))
	}

	// Create multiple conversations
	times := []time.Time{
		time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 3, 12, 0, 0, 0, time.UTC),
	}

	for i, t := range times {
		conv := &Conversation{
			ID:        string(rune('a' + i)),
			CreatedAt: t,
			Title:     "Conversation " + string(rune('A'+i)),
			Messages:  []Message{{Role: "user", Content: "Test"}},
		}
		SaveConversation(conv)
	}

	// List conversations
	conversations, err = ListConversations()
	helper.AssertNoError(err, "ListConversations should succeed")

	if len(conversations) != 3 {
		t.Fatalf("Expected 3 conversations, got %d", len(conversations))
	}

	// Verify sorted by creation time (newest first)
	if !conversations[0].CreatedAt.After(conversations[1].CreatedAt) {
		t.Error("Conversations should be sorted newest first")
	}
	if !conversations[1].CreatedAt.After(conversations[2].CreatedAt) {
		t.Error("Conversations should be sorted newest first")
	}

	// Verify message count
	for _, conv := range conversations {
		if conv.MessageCount != 1 {
			t.Errorf("Expected MessageCount=1, got %d", conv.MessageCount)
		}
	}
}

// TestListConversationsWithInvalidFiles tests listing with invalid files
func TestListConversationsWithInvalidFiles(t *testing.T) {
	helper := NewTestHelper(t)
	tempDir := helper.CreateTempDir()
	defer helper.Cleanup()

	oldDataDir := DataDir
	DataDir = tempDir
	defer func() { DataDir = oldDataDir }()

	// Create valid conversation
	SaveConversation(&Conversation{
		ID:        "valid",
		CreatedAt: time.Now(),
		Title:     "Valid",
		Messages:  []Message{},
	})

	// Create invalid JSON file
	os.WriteFile(filepath.Join(tempDir, "invalid.json"), []byte("{ invalid }"), 0644)

	// Create non-JSON file (should be skipped)
	os.WriteFile(filepath.Join(tempDir, "readme.txt"), []byte("text"), 0644)

	// Create directory (should be skipped)
	os.Mkdir(filepath.Join(tempDir, "subdir"), 0755)

	// List conversations - should only return valid one
	conversations, err := ListConversations()
	helper.AssertNoError(err, "ListConversations should succeed despite invalid files")

	if len(conversations) != 1 {
		t.Errorf("Expected 1 valid conversation, got %d", len(conversations))
	}
	if conversations[0].ID != "valid" {
		t.Errorf("Expected valid conversation, got %s", conversations[0].ID)
	}
}

// TestAddUserMessage tests adding a user message
func TestAddUserMessage(t *testing.T) {
	helper := NewTestHelper(t)
	tempDir := helper.CreateTempDir()
	defer helper.Cleanup()

	oldDataDir := DataDir
	DataDir = tempDir
	defer func() { DataDir = oldDataDir }()

	// Create conversation
	CreateConversation("test-user-msg")

	// Add user message
	err := AddUserMessage("test-user-msg", "Hello, world!")
	helper.AssertNoError(err, "AddUserMessage should succeed")

	// Load conversation and verify
	conv, err := GetConversation("test-user-msg")
	helper.AssertNoError(err, "Should load conversation")

	if len(conv.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(conv.Messages))
	}

	msg := conv.Messages[0]
	if msg.Role != "user" {
		t.Errorf("Role = %q, want 'user'", msg.Role)
	}
	if msg.Content != "Hello, world!" {
		t.Errorf("Content = %q, want 'Hello, world!'", msg.Content)
	}
}

// TestAddUserMessageNonExistent tests adding message to non-existent conversation
func TestAddUserMessageNonExistent(t *testing.T) {
	helper := NewTestHelper(t)
	tempDir := helper.CreateTempDir()
	defer helper.Cleanup()

	oldDataDir := DataDir
	DataDir = tempDir
	defer func() { DataDir = oldDataDir }()

	// Try to add message to non-existent conversation
	err := AddUserMessage("non-existent", "Hello")
	helper.AssertError(err, "Should error on non-existent conversation")
}

// TestAddAssistantMessage tests adding an assistant message
func TestAddAssistantMessage(t *testing.T) {
	helper := NewTestHelper(t)
	tempDir := helper.CreateTempDir()
	defer helper.Cleanup()

	oldDataDir := DataDir
	DataDir = tempDir
	defer func() { DataDir = oldDataDir }()

	// Create conversation
	CreateConversation("test-assistant-msg")

	// Create stage data
	stage1 := []Stage1Response{
		{Model: "test/model", Response: "Test response"},
	}
	stage2 := []Stage2Ranking{
		{Model: "test/model", Ranking: "Test ranking", ParsedRanking: []string{"Response A"}},
	}
	stage3 := Stage3Response{
		Model:    "test/chairman",
		Response: "Final response",
	}

	// Add assistant message
	err := AddAssistantMessage("test-assistant-msg", stage1, stage2, stage3)
	helper.AssertNoError(err, "AddAssistantMessage should succeed")

	// Load conversation and verify
	conv, err := GetConversation("test-assistant-msg")
	helper.AssertNoError(err, "Should load conversation")

	if len(conv.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(conv.Messages))
	}

	msg := conv.Messages[0]
	if msg.Role != "assistant" {
		t.Errorf("Role = %q, want 'assistant'", msg.Role)
	}
	if len(msg.Stage1) != 1 {
		t.Errorf("Expected 1 Stage1 response, got %d", len(msg.Stage1))
	}
	if len(msg.Stage2) != 1 {
		t.Errorf("Expected 1 Stage2 ranking, got %d", len(msg.Stage2))
	}
	if msg.Stage3 == nil {
		t.Error("Stage3 should not be nil")
	}
}

// TestAddAssistantMessageNonExistent tests adding assistant message to non-existent conversation
func TestAddAssistantMessageNonExistent(t *testing.T) {
	helper := NewTestHelper(t)
	tempDir := helper.CreateTempDir()
	defer helper.Cleanup()

	oldDataDir := DataDir
	DataDir = tempDir
	defer func() { DataDir = oldDataDir }()

	// Try to add message to non-existent conversation
	err := AddAssistantMessage("non-existent", []Stage1Response{}, []Stage2Ranking{}, Stage3Response{})
	helper.AssertError(err, "Should error on non-existent conversation")
}

// TestUpdateConversationTitle tests updating conversation title
func TestUpdateConversationTitle(t *testing.T) {
	helper := NewTestHelper(t)
	tempDir := helper.CreateTempDir()
	defer helper.Cleanup()

	oldDataDir := DataDir
	DataDir = tempDir
	defer func() { DataDir = oldDataDir }()

	// Create conversation
	CreateConversation("test-title-update")

	// Update title
	err := UpdateConversationTitle("test-title-update", "Updated Title")
	helper.AssertNoError(err, "UpdateConversationTitle should succeed")

	// Load conversation and verify
	conv, err := GetConversation("test-title-update")
	helper.AssertNoError(err, "Should load conversation")

	if conv.Title != "Updated Title" {
		t.Errorf("Title = %q, want 'Updated Title'", conv.Title)
	}
}

// TestUpdateConversationTitleNonExistent tests updating title of non-existent conversation
func TestUpdateConversationTitleNonExistent(t *testing.T) {
	helper := NewTestHelper(t)
	tempDir := helper.CreateTempDir()
	defer helper.Cleanup()

	oldDataDir := DataDir
	DataDir = tempDir
	defer func() { DataDir = oldDataDir }()

	// Try to update non-existent conversation
	err := UpdateConversationTitle("non-existent", "New Title")
	helper.AssertError(err, "Should error on non-existent conversation")
}

// TestConversationWorkflow tests a complete workflow
func TestConversationWorkflow(t *testing.T) {
	helper := NewTestHelper(t)
	tempDir := helper.CreateTempDir()
	defer helper.Cleanup()

	oldDataDir := DataDir
	DataDir = tempDir
	defer func() { DataDir = oldDataDir }()

	// Create conversation
	conv, err := CreateConversation("workflow-test")
	helper.AssertNoError(err, "CreateConversation should succeed")

	// Add user message
	err = AddUserMessage(conv.ID, "What is Go?")
	helper.AssertNoError(err, "AddUserMessage should succeed")

	// Add assistant message
	stage1 := []Stage1Response{{Model: "test", Response: "Go is great"}}
	stage2 := []Stage2Ranking{{Model: "test", Ranking: "FINAL RANKING:\n1. Response A", ParsedRanking: []string{"Response A"}}}
	stage3 := Stage3Response{Model: "chairman", Response: "Go is a programming language"}

	err = AddAssistantMessage(conv.ID, stage1, stage2, stage3)
	helper.AssertNoError(err, "AddAssistantMessage should succeed")

	// Update title
	err = UpdateConversationTitle(conv.ID, "Go Programming")
	helper.AssertNoError(err, "UpdateConversationTitle should succeed")

	// Load final conversation
	finalConv, err := GetConversation(conv.ID)
	helper.AssertNoError(err, "Should load conversation")

	// Verify final state
	if finalConv.Title != "Go Programming" {
		t.Errorf("Final title = %q, want 'Go Programming'", finalConv.Title)
	}
	if len(finalConv.Messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(finalConv.Messages))
	}

	// List conversations
	conversations, err := ListConversations()
	helper.AssertNoError(err, "ListConversations should succeed")

	if len(conversations) != 1 {
		t.Errorf("Expected 1 conversation in list, got %d", len(conversations))
	}
	if conversations[0].MessageCount != 2 {
		t.Errorf("Expected MessageCount=2, got %d", conversations[0].MessageCount)
	}
}

// TestSaveConversationError tests error handling in SaveConversation
func TestSaveConversationError(t *testing.T) {
	oldDataDir := DataDir
	DataDir = "/root/cannot/write/here"
	defer func() { DataDir = oldDataDir }()

	conv := &Conversation{
		ID:        "test",
		CreatedAt: time.Now(),
		Title:     "Test",
		Messages:  []Message{},
	}

	err := SaveConversation(conv)
	if err == nil {
		t.Error("Expected error when saving to invalid directory")
	}
}

// TestCreateConversationSaveError tests error during conversation save
func TestCreateConversationSaveError(t *testing.T) {
	oldDataDir := DataDir
	DataDir = "/root/cannot/write/here"
	defer func() { DataDir = oldDataDir }()

	_, err := CreateConversation("test")
	if err == nil {
		t.Error("Expected error when creating conversation in invalid directory")
	}
}
