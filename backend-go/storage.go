package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// EnsureDataDir ensures the data directory exists.
// Creates the directory with 0755 permissions if it doesn't exist.
func EnsureDataDir() error {
	return os.MkdirAll(DataDir, 0755)
}

// GetConversationPath returns the file path for a conversation.
// Joins the data directory with the conversation ID and .json extension.
func GetConversationPath(conversationID string) string {
	return filepath.Join(DataDir, conversationID+".json")
}

// CreateConversation creates a new conversation with the given ID.
// Initializes an empty conversation with default title and saves it to disk.
// Returns the created conversation or an error if creation fails.
func CreateConversation(conversationID string) (*Conversation, error) {
	// Ensure data directory exists
	if err := EnsureDataDir(); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Create new conversation
	conversation := &Conversation{
		ID:        conversationID,
		CreatedAt: time.Now().UTC(),
		Title:     "New Conversation",
		Messages:  []Message{},
	}

	// Save to file
	if err := SaveConversation(conversation); err != nil {
		return nil, err
	}

	return conversation, nil
}

// GetConversation loads a conversation from storage by ID.
// Returns nil without error if the conversation doesn't exist.
// Returns an error only if file reading or JSON parsing fails.
func GetConversation(conversationID string) (*Conversation, error) {
	path := GetConversationPath(conversationID)

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil // Not found, return nil without error
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read conversation file: %w", err)
	}

	// Parse JSON
	var conversation Conversation
	if err := json.Unmarshal(data, &conversation); err != nil {
		return nil, fmt.Errorf("failed to parse conversation JSON: %w", err)
	}

	return &conversation, nil
}

// SaveConversation saves a conversation to storage.
// Writes the conversation as formatted JSON to disk.
// Returns an error if directory creation, marshaling, or writing fails.
func SaveConversation(conversation *Conversation) error {
	// Ensure data directory exists
	if err := EnsureDataDir(); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(conversation, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal conversation: %w", err)
	}

	// Write to file
	path := GetConversationPath(conversation.ID)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write conversation file: %w", err)
	}

	return nil
}

// ListConversations lists all conversations with metadata only.
// Returns a slice of conversation metadata sorted by creation time (newest first).
// Silently skips invalid or unreadable files. Returns empty slice if no conversations exist.
func ListConversations() ([]ConversationMetadata, error) {
	// Ensure data directory exists
	if err := EnsureDataDir(); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Read directory
	entries, err := os.ReadDir(DataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read data directory: %w", err)
	}

	// Collect metadata (initialize with empty slice to avoid null in JSON)
	conversations := make([]ConversationMetadata, 0)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		// Read file
		path := filepath.Join(DataDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue // Skip files we can't read
		}

		// Parse JSON (just enough to get metadata)
		var conv Conversation
		if err := json.Unmarshal(data, &conv); err != nil {
			continue // Skip invalid JSON
		}

		// Extract metadata
		conversations = append(conversations, ConversationMetadata{
			ID:           conv.ID,
			CreatedAt:    conv.CreatedAt,
			Title:        conv.Title,
			MessageCount: len(conv.Messages),
		})
	}

	// Sort by creation time, newest first
	sort.Slice(conversations, func(i, j int) bool {
		return conversations[i].CreatedAt.After(conversations[j].CreatedAt)
	})

	return conversations, nil
}

// AddUserMessage adds a user message to a conversation.
// Appends the message to the conversation's message history and saves to disk.
// Returns an error if the conversation doesn't exist or saving fails.
func AddUserMessage(conversationID string, content string) error {
	// Load conversation
	conversation, err := GetConversation(conversationID)
	if err != nil {
		return err
	}
	if conversation == nil {
		return fmt.Errorf("conversation %s not found", conversationID)
	}

	// Append user message
	conversation.Messages = append(conversation.Messages, Message{
		Role:    "user",
		Content: content,
	})

	// Save conversation
	return SaveConversation(conversation)
}

// AddAssistantMessage adds an assistant message with all 3 stages.
// Stores the complete council results (stage1, stage2, stage3) as a single message.
// Returns an error if the conversation doesn't exist or saving fails.
func AddAssistantMessage(conversationID string, stage1 []Stage1Response, stage2 []Stage2Ranking, stage3 Stage3Response) error {
	// Load conversation
	conversation, err := GetConversation(conversationID)
	if err != nil {
		return err
	}
	if conversation == nil {
		return fmt.Errorf("conversation %s not found", conversationID)
	}

	// Append assistant message
	conversation.Messages = append(conversation.Messages, Message{
		Role:   "assistant",
		Stage1: stage1,
		Stage2: stage2,
		Stage3: &stage3,
	})

	// Save conversation
	return SaveConversation(conversation)
}

// UpdateConversationTitle updates the title of a conversation.
// Loads the conversation, updates its title field, and saves back to disk.
// Returns an error if the conversation doesn't exist or saving fails.
func UpdateConversationTitle(conversationID string, title string) error {
	// Load conversation
	conversation, err := GetConversation(conversationID)
	if err != nil {
		return err
	}
	if conversation == nil {
		return fmt.Errorf("conversation %s not found", conversationID)
	}

	// Update title
	conversation.Title = title

	// Save conversation
	return SaveConversation(conversation)
}
