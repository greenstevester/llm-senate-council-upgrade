package main

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
)

// TestParseRankingFromText tests the ranking parser with various formats
func TestParseRankingFromText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name: "standard format with FINAL RANKING",
			input: `Response A is good but lacks detail.
Response B provides comprehensive coverage.
Response C is accurate but brief.

FINAL RANKING:
1. Response B
2. Response A
3. Response C`,
			expected: []string{"Response B", "Response A", "Response C"},
		},
		{
			name: "format without numbered list",
			input: `FINAL RANKING:
Response C
Response A
Response B`,
			expected: []string{"Response C", "Response A", "Response B"},
		},
		{
			name: "format with extra whitespace",
			input: `FINAL RANKING:
1.  Response A
2.  Response B
3.  Response C`,
			expected: []string{"Response A", "Response B", "Response C"},
		},
		{
			name: "format with text after ranking section",
			input: `FINAL RANKING:
1. Response B
2. Response A
3. Response C

These are my rankings based on quality.`,
			expected: []string{"Response B", "Response A", "Response C"},
		},
		{
			name:     "no FINAL RANKING header - fallback",
			input:    `I think Response A is best, then Response C, then Response B.`,
			expected: []string{"Response A", "Response C", "Response B"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name: "FINAL RANKING with no responses",
			input: `FINAL RANKING:
No responses to rank.`,
			expected: []string{},
		},
		{
			name: "multiple occurrences - only from FINAL RANKING section",
			input: `Response A is mentioned here first.
Response B is also mentioned.

FINAL RANKING:
1. Response C
2. Response A`,
			expected: []string{"Response C", "Response A"},
		},
		{
			name: "responses with letters beyond C",
			input: `FINAL RANKING:
1. Response D
2. Response A
3. Response B
4. Response C`,
			expected: []string{"Response D", "Response A", "Response B", "Response C"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseRankingFromText(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("Length mismatch: got %d, want %d", len(result), len(tt.expected))
				t.Errorf("Got: %v", result)
				t.Errorf("Want: %v", tt.expected)
				return
			}

			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("At index %d: got %q, want %q", i, result[i], tt.expected[i])
				}
			}
		})
	}
}

// TestCalculateAggregateRankings tests aggregate ranking calculation
func TestCalculateAggregateRankings(t *testing.T) {
	tests := []struct {
		name          string
		stage2Results []Stage2Ranking
		labelToModel  map[string]string
		expectedLen   int
		checkFirst    string // Expected first model in ranking
	}{
		{
			name: "single model ranking all responses",
			stage2Results: []Stage2Ranking{
				{
					Model:         "test/ranker1",
					ParsedRanking: []string{"Response A", "Response B", "Response C"},
				},
			},
			labelToModel: map[string]string{
				"Response A": "model/a",
				"Response B": "model/b",
				"Response C": "model/c",
			},
			expectedLen: 3,
			checkFirst:  "model/a", // Should be first (rank 1)
		},
		{
			name: "multiple models with consensus",
			stage2Results: []Stage2Ranking{
				{
					Model:         "test/ranker1",
					ParsedRanking: []string{"Response A", "Response B"},
				},
				{
					Model:         "test/ranker2",
					ParsedRanking: []string{"Response A", "Response B"},
				},
			},
			labelToModel: map[string]string{
				"Response A": "model/a",
				"Response B": "model/b",
			},
			expectedLen: 2,
			checkFirst:  "model/a",
		},
		{
			name: "multiple models with disagreement",
			stage2Results: []Stage2Ranking{
				{
					Model:         "test/ranker1",
					ParsedRanking: []string{"Response A", "Response B"},
				},
				{
					Model:         "test/ranker2",
					ParsedRanking: []string{"Response B", "Response A"},
				},
			},
			labelToModel: map[string]string{
				"Response A": "model/a",
				"Response B": "model/b",
			},
			expectedLen: 2,
			// Average: model/a = (1+2)/2 = 1.5, model/b = (2+1)/2 = 1.5
			// With tie, order may vary, so we don't check first
		},
		{
			name: "empty rankings",
			stage2Results: []Stage2Ranking{
				{
					Model:         "test/ranker1",
					ParsedRanking: []string{},
				},
			},
			labelToModel: map[string]string{
				"Response A": "model/a",
			},
			expectedLen: 0,
		},
		{
			name: "partial rankings - not all models ranked",
			stage2Results: []Stage2Ranking{
				{
					Model:         "test/ranker1",
					ParsedRanking: []string{"Response A"},
				},
				{
					Model:         "test/ranker2",
					ParsedRanking: []string{"Response A", "Response B"},
				},
			},
			labelToModel: map[string]string{
				"Response A": "model/a",
				"Response B": "model/b",
			},
			expectedLen: 2,
			checkFirst:  "model/a", // Gets 1 from both rankers
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateAggregateRankings(tt.stage2Results, tt.labelToModel)

			if len(result) != tt.expectedLen {
				t.Errorf("Length mismatch: got %d, want %d", len(result), tt.expectedLen)
			}

			// Check that rankings are sorted (lower average rank = better)
			for i := 0; i < len(result)-1; i++ {
				if result[i].AverageRank > result[i+1].AverageRank {
					t.Errorf("Rankings not sorted: position %d has rank %.2f, position %d has rank %.2f",
						i, result[i].AverageRank, i+1, result[i+1].AverageRank)
				}
			}

			// Check first model if specified
			if tt.checkFirst != "" && len(result) > 0 {
				if result[0].Model != tt.checkFirst {
					t.Errorf("First model: got %q, want %q", result[0].Model, tt.checkFirst)
				}
			}

			// Verify all rankings have positive count
			for _, ranking := range result {
				if ranking.RankingsCount <= 0 {
					t.Errorf("Model %s has invalid RankingsCount: %d", ranking.Model, ranking.RankingsCount)
				}
			}
		})
	}
}

// TestCalculateAggregateRankingsAverages tests specific average calculations
func TestCalculateAggregateRankingsAverages(t *testing.T) {
	stage2Results := []Stage2Ranking{
		{
			Model:         "ranker1",
			ParsedRanking: []string{"Response A", "Response B", "Response C"},
		},
		{
			Model:         "ranker2",
			ParsedRanking: []string{"Response B", "Response C", "Response A"},
		},
		{
			Model:         "ranker3",
			ParsedRanking: []string{"Response C", "Response A", "Response B"},
		},
	}

	labelToModel := map[string]string{
		"Response A": "model/a",
		"Response B": "model/b",
		"Response C": "model/c",
	}

	result := CalculateAggregateRankings(stage2Results, labelToModel)

	// Calculate expected averages:
	// model/a: (1+3+2)/3 = 6/3 = 2.0
	// model/b: (2+1+3)/3 = 6/3 = 2.0
	// model/c: (3+2+1)/3 = 6/3 = 2.0

	if len(result) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(result))
	}

	for _, r := range result {
		if r.AverageRank != 2.0 {
			t.Errorf("Model %s: expected average rank 2.0, got %.2f", r.Model, r.AverageRank)
		}
		if r.RankingsCount != 3 {
			t.Errorf("Model %s: expected 3 rankings, got %d", r.Model, r.RankingsCount)
		}
	}
}

// TestStage1CollectResponses tests Stage 1 with mocked API
func TestStage1CollectResponses(t *testing.T) {
	// Save original config
	oldAPIURL := OpenRouterAPIURL
	oldAPIKey := OpenRouterAPIKey
	oldModels := CouncilModels
	defer func() {
		OpenRouterAPIURL = oldAPIURL
		OpenRouterAPIKey = oldAPIKey
		CouncilModels = oldModels
	}()

	// Create mock server
	mockServer := MockOpenRouterServer(t, CreateMockOpenRouterHandler(t, "This is a test response from the model."))
	defer mockServer.Close()

	// Configure for testing
	OpenRouterAPIURL = mockServer.URL
	OpenRouterAPIKey = "test-key"
	CouncilModels = []string{"test/model1", "test/model2"}

	// Run Stage 1
	ctx := context.Background()
	results, err := Stage1CollectResponses(ctx, "What is Go?")

	if err != nil {
		t.Fatalf("Stage1CollectResponses failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// Verify all results have content
	for _, result := range results {
		if result.Response == "" {
			t.Errorf("Model %s returned empty response", result.Model)
		}
	}
}

// TestStage2CollectRankings tests Stage 2 ranking collection
func TestStage2CollectRankings(t *testing.T) {
	// Save original config
	oldAPIURL := OpenRouterAPIURL
	oldAPIKey := OpenRouterAPIKey
	oldModels := CouncilModels
	defer func() {
		OpenRouterAPIURL = oldAPIURL
		OpenRouterAPIKey = oldAPIKey
		CouncilModels = oldModels
	}()

	// Create mock server that returns a ranking
	mockRankingResponse := `Response A provides good detail.
Response B is comprehensive.

FINAL RANKING:
1. Response B
2. Response A`

	mockServer := MockOpenRouterServer(t, CreateMockOpenRouterHandler(t, mockRankingResponse))
	defer mockServer.Close()

	// Configure for testing
	OpenRouterAPIURL = mockServer.URL
	OpenRouterAPIKey = "test-key"
	CouncilModels = []string{"test/ranker"}

	// Create stage1 results
	stage1 := []Stage1Response{
		{Model: "model/a", Response: "Response from model A"},
		{Model: "model/b", Response: "Response from model B"},
	}

	// Run Stage 2
	ctx := context.Background()
	results, labelToModel, err := Stage2CollectRankings(ctx, "What is Go?", stage1)

	if err != nil {
		t.Fatalf("Stage2CollectRankings failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	// Check label mapping
	if len(labelToModel) != 2 {
		t.Errorf("Expected 2 label mappings, got %d", len(labelToModel))
	}

	// Verify labels are Response A and Response B
	if _, ok := labelToModel["Response A"]; !ok {
		t.Error("Missing Response A in label mapping")
	}
	if _, ok := labelToModel["Response B"]; !ok {
		t.Error("Missing Response B in label mapping")
	}

	// Check parsed ranking
	if len(results) > 0 {
		parsed := results[0].ParsedRanking
		expected := []string{"Response B", "Response A"}
		if !reflect.DeepEqual(parsed, expected) {
			t.Errorf("ParsedRanking = %v, want %v", parsed, expected)
		}
	}
}

// TestStage3SynthesizeFinal tests Stage 3 synthesis
func TestStage3SynthesizeFinal(t *testing.T) {
	// Save original config
	oldAPIURL := OpenRouterAPIURL
	oldAPIKey := OpenRouterAPIKey
	oldChairman := ChairmanModel
	defer func() {
		OpenRouterAPIURL = oldAPIURL
		OpenRouterAPIKey = oldAPIKey
		ChairmanModel = oldChairman
	}()

	// Create mock server
	mockServer := MockOpenRouterServer(t, CreateMockOpenRouterHandler(t, "Go is a statically typed, compiled programming language designed at Google."))
	defer mockServer.Close()

	// Configure for testing
	OpenRouterAPIURL = mockServer.URL
	OpenRouterAPIKey = "test-key"
	ChairmanModel = "test/chairman"

	// Create stage1 and stage2 data
	stage1 := []Stage1Response{
		{Model: "model/a", Response: "Go is a programming language."},
		{Model: "model/b", Response: "Go was created by Google."},
	}

	stage2 := []Stage2Ranking{
		{
			Model:         "model/a",
			Ranking:       "FINAL RANKING:\n1. Response B\n2. Response A",
			ParsedRanking: []string{"Response B", "Response A"},
		},
	}

	// Run Stage 3
	ctx := context.Background()
	result, err := Stage3SynthesizeFinal(ctx, "What is Go?", stage1, stage2)

	if err != nil {
		t.Fatalf("Stage3SynthesizeFinal failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if result.Model != ChairmanModel {
		t.Errorf("Model = %q, want %q", result.Model, ChairmanModel)
	}

	if result.Response == "" {
		t.Error("Response should not be empty")
	}
}

// TestGenerateConversationTitle tests title generation
func TestGenerateConversationTitle(t *testing.T) {
	// Save original config
	oldAPIURL := OpenRouterAPIURL
	oldAPIKey := OpenRouterAPIKey
	defer func() {
		OpenRouterAPIURL = oldAPIURL
		OpenRouterAPIKey = oldAPIKey
	}()

	// Create mock server
	mockServer := MockOpenRouterServer(t, CreateMockOpenRouterHandler(t, "Go Programming Language"))
	defer mockServer.Close()

	// Configure for testing
	OpenRouterAPIURL = mockServer.URL
	OpenRouterAPIKey = "test-key"

	// Generate title
	ctx := context.Background()
	title, err := GenerateConversationTitle(ctx, "What is the Go programming language and how does it work?")

	if err != nil {
		t.Fatalf("GenerateConversationTitle failed: %v", err)
	}

	if title == "" {
		t.Error("Title should not be empty")
	}

	if len(title) > 50 {
		t.Errorf("Title too long: %d characters (max 50)", len(title))
	}
}

// TestRunFullCouncil tests the complete 3-stage workflow
func TestRunFullCouncil(t *testing.T) {
	// This is an integration test covering all stages

	// Save original config
	oldAPIURL := OpenRouterAPIURL
	oldAPIKey := OpenRouterAPIKey
	oldModels := CouncilModels
	oldChairman := ChairmanModel
	defer func() {
		OpenRouterAPIURL = oldAPIURL
		OpenRouterAPIKey = oldAPIKey
		CouncilModels = oldModels
		ChairmanModel = oldChairman
	}()

	// Track which stage we're in based on the request
	requestCount := 0
	mockHandler := func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		var response string
		if requestCount <= 2 {
			// Stage 1 responses
			response = "This is response " + string(rune('A'+requestCount-1))
		} else if requestCount <= 4 {
			// Stage 2 rankings
			response = "FINAL RANKING:\n1. Response B\n2. Response A"
		} else {
			// Stage 3 synthesis
			response = "Go is a programming language created by Google."
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
						Content: response,
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(apiResponse)
	}

	mockServer := MockOpenRouterServer(t, mockHandler)
	defer mockServer.Close()

	// Configure for testing
	OpenRouterAPIURL = mockServer.URL
	OpenRouterAPIKey = "test-key"
	CouncilModels = []string{"model/a", "model/b"}
	ChairmanModel = "model/chairman"

	// Run full council
	ctx := context.Background()
	stage1, stage2, stage3, metadata, err := RunFullCouncil(ctx, "What is Go?")

	if err != nil {
		t.Fatalf("RunFullCouncil failed: %v", err)
	}

	// Verify Stage 1
	if len(stage1) != 2 {
		t.Errorf("Stage1: expected 2 responses, got %d", len(stage1))
	}

	// Verify Stage 2
	if len(stage2) != 2 {
		t.Errorf("Stage2: expected 2 rankings, got %d", len(stage2))
	}

	// Verify Stage 3
	if stage3.Response == "" {
		t.Error("Stage3: response should not be empty")
	}

	// Verify metadata
	if len(metadata.LabelToModel) == 0 {
		t.Error("Metadata: labelToModel should not be empty")
	}
	if len(metadata.AggregateRankings) == 0 {
		t.Error("Metadata: aggregateRankings should not be empty")
	}
}

// TestStage3WithChairmanError tests error handling in stage 3
func TestStage3WithChairmanError(t *testing.T) {
	oldAPIURL := OpenRouterAPIURL
	oldAPIKey := OpenRouterAPIKey
	oldChairman := ChairmanModel
	defer func() {
		OpenRouterAPIURL = oldAPIURL
		OpenRouterAPIKey = oldAPIKey
		ChairmanModel = oldChairman
	}()

	// Create failing mock server
	failingServer := MockOpenRouterServer(t, CreateMockOpenRouterErrorHandler(500, "Error"))
	defer failingServer.Close()

	OpenRouterAPIURL = failingServer.URL
	OpenRouterAPIKey = "test-key"
	ChairmanModel = "test/chairman"

	stage1 := []Stage1Response{{Model: "model/a", Response: "Test"}}
	stage2 := []Stage2Ranking{{Model: "model/a", Ranking: "FINAL RANKING:\n1. Response A", ParsedRanking: []string{"Response A"}}}

	ctx := context.Background()
	result, err := Stage3SynthesizeFinal(ctx, "Test", stage1, stage2)

	// Should return error now instead of error message
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if result != nil {
		t.Errorf("Expected nil result on error, got: %v", result)
	}
}

// TestGenerateConversationTitleError tests error handling in title generation
func TestGenerateConversationTitleError(t *testing.T) {
	oldAPIURL := OpenRouterAPIURL
	oldAPIKey := OpenRouterAPIKey
	defer func() {
		OpenRouterAPIURL = oldAPIURL
		OpenRouterAPIKey = oldAPIKey
	}()

	// Create failing mock server
	failingServer := MockOpenRouterServer(t, CreateMockOpenRouterErrorHandler(500, "Error"))
	defer failingServer.Close()

	OpenRouterAPIURL = failingServer.URL
	OpenRouterAPIKey = "test-key"

	ctx := context.Background()
	title, err := GenerateConversationTitle(ctx, "Test")

	// Should return error now instead of default title
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if title != "" {
		t.Errorf("Expected empty title on error, got: %s", title)
	}
}

// TestGenerateConversationTitleTruncation tests title truncation
func TestGenerateConversationTitleTruncation(t *testing.T) {
	oldAPIURL := OpenRouterAPIURL
	oldAPIKey := OpenRouterAPIKey
	defer func() {
		OpenRouterAPIURL = oldAPIURL
		OpenRouterAPIKey = oldAPIKey
	}()

	// Create mock that returns very long title
	longTitle := "This is a very long title that exceeds the maximum length and should be truncated"
	mockServer := MockOpenRouterServer(t, CreateMockOpenRouterHandler(t, longTitle))
	defer mockServer.Close()

	OpenRouterAPIURL = mockServer.URL
	OpenRouterAPIKey = "test-key"

	ctx := context.Background()
	title, err := GenerateConversationTitle(ctx, "Test")

	if err != nil {
		t.Fatalf("GenerateConversationTitle failed: %v", err)
	}

	if len(title) > 50 {
		t.Errorf("Title not truncated: length = %d", len(title))
	}

	// Should end with "..."
	if len(title) == 50 && title[len(title)-3:] != "..." {
		t.Error("Truncated title should end with '...'")
	}
}

// TestGenerateConversationTitleQuoteRemoval tests quote removal from title
func TestGenerateConversationTitleQuoteRemoval(t *testing.T) {
	oldAPIURL := OpenRouterAPIURL
	oldAPIKey := OpenRouterAPIKey
	defer func() {
		OpenRouterAPIURL = oldAPIURL
		OpenRouterAPIKey = oldAPIKey
	}()

	// Create mock that returns title with quotes
	mockServer := MockOpenRouterServer(t, CreateMockOpenRouterHandler(t, "\"Go Programming\""))
	defer mockServer.Close()

	OpenRouterAPIURL = mockServer.URL
	OpenRouterAPIKey = "test-key"

	ctx := context.Background()
	title, err := GenerateConversationTitle(ctx, "Test")

	if err != nil {
		t.Fatalf("GenerateConversationTitle failed: %v", err)
	}

	if title != "Go Programming" {
		t.Errorf("Quotes not removed: %s", title)
	}
}
