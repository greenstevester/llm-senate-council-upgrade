package main

import "time"

// Message represents a single message in a conversation
type Message struct {
	Role    string                 `json:"role"`
	Content string                 `json:"content,omitempty"`
	Stage1  []Stage1Response       `json:"stage1,omitempty"`
	Stage2  []Stage2Ranking        `json:"stage2,omitempty"`
	Stage3  *Stage3Response        `json:"stage3,omitempty"`
}

// Conversation represents a full conversation with all messages
type Conversation struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Title     string    `json:"title"`
	Messages  []Message `json:"messages"`
}

// ConversationMetadata represents conversation list metadata
type ConversationMetadata struct {
	ID           string    `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	Title        string    `json:"title"`
	MessageCount int       `json:"message_count"`
}

// Stage1Response represents a single model's response in Stage 1
type Stage1Response struct {
	Model    string `json:"model"`
	Response string `json:"response"`
}

// Stage2Ranking represents a model's ranking of other responses
type Stage2Ranking struct {
	Model          string   `json:"model"`
	Ranking        string   `json:"ranking"`
	ParsedRanking  []string `json:"parsed_ranking"`
}

// Stage3Response represents the chairman's final synthesis
type Stage3Response struct {
	Model    string `json:"model"`
	Response string `json:"response"`
}

// AggregateRanking represents the aggregate ranking across all models
type AggregateRanking struct {
	Model          string  `json:"model"`
	AverageRank    float64 `json:"average_rank"`
	RankingsCount  int     `json:"rankings_count"`
}

// Metadata contains additional information about the council process
type Metadata struct {
	LabelToModel       map[string]string  `json:"label_to_model"`
	AggregateRankings  []AggregateRanking `json:"aggregate_rankings"`
}

// OpenRouterMessage represents a message for OpenRouter API
type OpenRouterMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenRouterRequest represents a request to OpenRouter API
type OpenRouterRequest struct {
	Model    string                `json:"model"`
	Messages []OpenRouterMessage   `json:"messages"`
}

// OpenRouterResponse represents a response from OpenRouter API
type OpenRouterResponse struct {
	Content          string      `json:"content"`
	ReasoningDetails interface{} `json:"reasoning_details,omitempty"`
}

// OpenRouterAPIResponse represents the full API response structure
type OpenRouterAPIResponse struct {
	Choices []struct {
		Message struct {
			Content          string      `json:"content"`
			ReasoningDetails interface{} `json:"reasoning_details,omitempty"`
		} `json:"message"`
	} `json:"choices"`
}

// CreateConversationRequest represents a request to create a new conversation
type CreateConversationRequest struct {
	// Empty for now
}

// SendMessageRequest represents a request to send a message
type SendMessageRequest struct {
	Content string `json:"content"`
}

// SendMessageResponse represents the response after sending a message
type SendMessageResponse struct {
	Stage1   []Stage1Response `json:"stage1"`
	Stage2   []Stage2Ranking  `json:"stage2"`
	Stage3   Stage3Response   `json:"stage3"`
	Metadata Metadata         `json:"metadata"`
}
