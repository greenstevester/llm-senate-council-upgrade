package main

import (
	"encoding/json"
	"testing"
	"time"
)

// TestMessageJSONMarshaling tests JSON marshaling and unmarshaling of Message
func TestMessageJSONMarshaling(t *testing.T) {
	tests := []struct {
		name    string
		message Message
	}{
		{
			name: "user message",
			message: Message{
				Role:    "user",
				Content: "Hello, world!",
			},
		},
		{
			name: "assistant message with all stages",
			message: Message{
				Role: "assistant",
				Stage1: []Stage1Response{
					{Model: "test/model", Response: "Test response"},
				},
				Stage2: []Stage2Ranking{
					{Model: "test/model", Ranking: "Test ranking", ParsedRanking: []string{"Response A"}},
				},
				Stage3: &Stage3Response{
					Model:    "test/chairman",
					Response: "Final response",
				},
			},
		},
		{
			name: "empty message",
			message: Message{
				Role: "user",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.message)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			// Unmarshal back
			var decoded Message
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			// Verify role
			if decoded.Role != tt.message.Role {
				t.Errorf("Role mismatch: got %s, want %s", decoded.Role, tt.message.Role)
			}

			// Verify content
			if decoded.Content != tt.message.Content {
				t.Errorf("Content mismatch: got %s, want %s", decoded.Content, tt.message.Content)
			}
		})
	}
}

// TestConversationJSONMarshaling tests JSON marshaling of Conversation
func TestConversationJSONMarshaling(t *testing.T) {
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	conversation := Conversation{
		ID:        "test-id",
		CreatedAt: fixedTime,
		Title:     "Test Conversation",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi"},
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(conversation)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal back
	var decoded Conversation
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify fields
	if decoded.ID != conversation.ID {
		t.Errorf("ID mismatch: got %s, want %s", decoded.ID, conversation.ID)
	}
	if decoded.Title != conversation.Title {
		t.Errorf("Title mismatch: got %s, want %s", decoded.Title, conversation.Title)
	}
	if len(decoded.Messages) != len(conversation.Messages) {
		t.Errorf("Messages length mismatch: got %d, want %d", len(decoded.Messages), len(conversation.Messages))
	}
}

// TestConversationMetadataJSONMarshaling tests JSON marshaling of ConversationMetadata
func TestConversationMetadataJSONMarshaling(t *testing.T) {
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	metadata := ConversationMetadata{
		ID:           "test-id",
		CreatedAt:    fixedTime,
		Title:        "Test",
		MessageCount: 5,
	}

	// Marshal and unmarshal
	data, err := json.Marshal(metadata)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded ConversationMetadata
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify
	if decoded.ID != metadata.ID {
		t.Errorf("ID mismatch: got %s, want %s", decoded.ID, metadata.ID)
	}
	if decoded.MessageCount != metadata.MessageCount {
		t.Errorf("MessageCount mismatch: got %d, want %d", decoded.MessageCount, metadata.MessageCount)
	}
}

// TestStage1ResponseJSONMarshaling tests JSON marshaling of Stage1Response
func TestStage1ResponseJSONMarshaling(t *testing.T) {
	response := Stage1Response{
		Model:    "test/model",
		Response: "This is a test response",
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded Stage1Response
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Model != response.Model {
		t.Errorf("Model mismatch: got %s, want %s", decoded.Model, response.Model)
	}
	if decoded.Response != response.Response {
		t.Errorf("Response mismatch: got %s, want %s", decoded.Response, response.Response)
	}
}

// TestStage2RankingJSONMarshaling tests JSON marshaling of Stage2Ranking
func TestStage2RankingJSONMarshaling(t *testing.T) {
	ranking := Stage2Ranking{
		Model:         "test/model",
		Ranking:       "FINAL RANKING:\n1. Response A\n2. Response B",
		ParsedRanking: []string{"Response A", "Response B"},
	}

	data, err := json.Marshal(ranking)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded Stage2Ranking
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Model != ranking.Model {
		t.Errorf("Model mismatch: got %s, want %s", decoded.Model, ranking.Model)
	}
	if len(decoded.ParsedRanking) != len(ranking.ParsedRanking) {
		t.Errorf("ParsedRanking length mismatch: got %d, want %d", len(decoded.ParsedRanking), len(ranking.ParsedRanking))
	}
}

// TestStage3ResponseJSONMarshaling tests JSON marshaling of Stage3Response
func TestStage3ResponseJSONMarshaling(t *testing.T) {
	response := Stage3Response{
		Model:    "test/chairman",
		Response: "Final synthesis",
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded Stage3Response
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Model != response.Model {
		t.Errorf("Model mismatch: got %s, want %s", decoded.Model, response.Model)
	}
	if decoded.Response != response.Response {
		t.Errorf("Response mismatch: got %s, want %s", decoded.Response, response.Response)
	}
}

// TestAggregateRankingJSONMarshaling tests JSON marshaling of AggregateRanking
func TestAggregateRankingJSONMarshaling(t *testing.T) {
	ranking := AggregateRanking{
		Model:         "test/model",
		AverageRank:   2.5,
		RankingsCount: 4,
	}

	data, err := json.Marshal(ranking)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded AggregateRanking
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Model != ranking.Model {
		t.Errorf("Model mismatch: got %s, want %s", decoded.Model, ranking.Model)
	}
	if decoded.AverageRank != ranking.AverageRank {
		t.Errorf("AverageRank mismatch: got %f, want %f", decoded.AverageRank, ranking.AverageRank)
	}
	if decoded.RankingsCount != ranking.RankingsCount {
		t.Errorf("RankingsCount mismatch: got %d, want %d", decoded.RankingsCount, ranking.RankingsCount)
	}
}

// TestMetadataJSONMarshaling tests JSON marshaling of Metadata
func TestMetadataJSONMarshaling(t *testing.T) {
	metadata := Metadata{
		LabelToModel: map[string]string{
			"Response A": "test/model1",
			"Response B": "test/model2",
		},
		AggregateRankings: []AggregateRanking{
			{Model: "test/model1", AverageRank: 1.5, RankingsCount: 2},
			{Model: "test/model2", AverageRank: 2.5, RankingsCount: 2},
		},
	}

	data, err := json.Marshal(metadata)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded Metadata
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(decoded.LabelToModel) != len(metadata.LabelToModel) {
		t.Errorf("LabelToModel length mismatch: got %d, want %d", len(decoded.LabelToModel), len(metadata.LabelToModel))
	}
	if len(decoded.AggregateRankings) != len(metadata.AggregateRankings) {
		t.Errorf("AggregateRankings length mismatch: got %d, want %d", len(decoded.AggregateRankings), len(metadata.AggregateRankings))
	}
}

// TestOpenRouterMessageJSONMarshaling tests JSON marshaling of OpenRouterMessage
func TestOpenRouterMessageJSONMarshaling(t *testing.T) {
	message := OpenRouterMessage{
		Role:    "user",
		Content: "Test content",
	}

	data, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded OpenRouterMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Role != message.Role {
		t.Errorf("Role mismatch: got %s, want %s", decoded.Role, message.Role)
	}
	if decoded.Content != message.Content {
		t.Errorf("Content mismatch: got %s, want %s", decoded.Content, message.Content)
	}
}

// TestOpenRouterRequestJSONMarshaling tests JSON marshaling of OpenRouterRequest
func TestOpenRouterRequestJSONMarshaling(t *testing.T) {
	request := OpenRouterRequest{
		Model: "test/model",
		Messages: []OpenRouterMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	data, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded OpenRouterRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Model != request.Model {
		t.Errorf("Model mismatch: got %s, want %s", decoded.Model, request.Model)
	}
	if len(decoded.Messages) != len(request.Messages) {
		t.Errorf("Messages length mismatch: got %d, want %d", len(decoded.Messages), len(request.Messages))
	}
}

// TestOpenRouterAPIResponseJSONUnmarshaling tests JSON unmarshaling of OpenRouterAPIResponse
func TestOpenRouterAPIResponseJSONUnmarshaling(t *testing.T) {
	jsonData := `{
		"choices": [
			{
				"message": {
					"content": "Test response",
					"reasoning_details": {"steps": 3}
				}
			}
		]
	}`

	var response OpenRouterAPIResponse
	if err := json.Unmarshal([]byte(jsonData), &response); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(response.Choices) != 1 {
		t.Errorf("Expected 1 choice, got %d", len(response.Choices))
	}
	if response.Choices[0].Message.Content != "Test response" {
		t.Errorf("Content mismatch: got %s, want 'Test response'", response.Choices[0].Message.Content)
	}
}

// TestSendMessageRequestJSONMarshaling tests JSON marshaling of SendMessageRequest
func TestSendMessageRequestJSONMarshaling(t *testing.T) {
	request := SendMessageRequest{
		Content: "What is Go?",
	}

	data, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded SendMessageRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Content != request.Content {
		t.Errorf("Content mismatch: got %s, want %s", decoded.Content, request.Content)
	}
}

// TestSendMessageResponseJSONMarshaling tests JSON marshaling of SendMessageResponse
func TestSendMessageResponseJSONMarshaling(t *testing.T) {
	response := SendMessageResponse{
		Stage1: []Stage1Response{
			{Model: "test/model", Response: "Response 1"},
		},
		Stage2: []Stage2Ranking{
			{Model: "test/model", Ranking: "Ranking", ParsedRanking: []string{"Response A"}},
		},
		Stage3: Stage3Response{
			Model:    "test/chairman",
			Response: "Final response",
		},
		Metadata: Metadata{
			LabelToModel: map[string]string{"Response A": "test/model"},
			AggregateRankings: []AggregateRanking{
				{Model: "test/model", AverageRank: 1.0, RankingsCount: 1},
			},
		},
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded SendMessageResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(decoded.Stage1) != len(response.Stage1) {
		t.Errorf("Stage1 length mismatch: got %d, want %d", len(decoded.Stage1), len(response.Stage1))
	}
	if len(decoded.Stage2) != len(response.Stage2) {
		t.Errorf("Stage2 length mismatch: got %d, want %d", len(decoded.Stage2), len(response.Stage2))
	}
}

// TestEmptySlicesInJSON tests that empty slices are marshaled as empty arrays, not null
func TestEmptySlicesInJSON(t *testing.T) {
	conversation := Conversation{
		ID:        "test",
		CreatedAt: time.Now(),
		Title:     "Test",
		Messages:  []Message{}, // Empty slice
	}

	data, err := json.Marshal(conversation)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Verify it contains [] not null
	jsonStr := string(data)
	if !contains(jsonStr, `"messages":[]`) {
		t.Errorf("Expected empty array for messages, got: %s", jsonStr)
	}
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
