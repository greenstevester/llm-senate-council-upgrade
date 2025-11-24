package main

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Stage1CollectResponses collects individual responses from all council models.
// This is the first stage of the council process where each model independently
// answers the user's question. Returns a slice of responses, one per successful model.
func Stage1CollectResponses(ctx context.Context, userQuery string) ([]Stage1Response, error) {
	// Create messages slice with user query
	messages := []OpenRouterMessage{
		{Role: "user", Content: userQuery},
	}

	// Query all models in parallel
	responses, err := QueryModelsParallel(ctx, CouncilModels, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to query models: %w", err)
	}

	// Format results - only include successful responses
	var stage1Results []Stage1Response
	for model, response := range responses {
		if response != nil {
			stage1Results = append(stage1Results, Stage1Response{
				Model:    model,
				Response: response.Content,
			})
		}
	}

	return stage1Results, nil
}

// Stage2CollectRankings collects rankings from each model on anonymized responses.
// This is the second stage where models evaluate each other's responses without
// knowing which model produced which response. Returns rankings, a label-to-model
// mapping for de-anonymization, and any error encountered.
func Stage2CollectRankings(ctx context.Context, userQuery string, stage1Results []Stage1Response) ([]Stage2Ranking, map[string]string, error) {
	// Create anonymized labels (A, B, C...)
	labelToModel := make(map[string]string)
	var responsesText strings.Builder

	for i, result := range stage1Results {
		label := string(rune('A' + i))
		labelKey := fmt.Sprintf("Response %s", label)
		labelToModel[labelKey] = result.Model

		responsesText.WriteString(fmt.Sprintf("Response %s:\n%s\n\n", label, result.Response))
	}

	// Build ranking prompt
	rankingPrompt := fmt.Sprintf(`You are evaluating different responses to the following question:

Question: %s

Here are the responses from different models (anonymized):

%s

Your task:
1. First, evaluate each response individually. For each response, explain what it does well and what it does poorly.
2. Then, at the very end of your response, provide a final ranking.

IMPORTANT: Your final ranking MUST be formatted EXACTLY as follows:
- Start with the line "FINAL RANKING:" (all caps, with colon)
- Then list the responses from best to worst as a numbered list
- Each line should be: number, period, space, then ONLY the response label (e.g., "1. Response A")
- Do not add any other text or explanations in the ranking section

Example of the correct format for your ENTIRE response:

Response A provides good detail on X but misses Y...
Response B is accurate but lacks depth on Z...
Response C offers the most comprehensive answer...

FINAL RANKING:
1. Response C
2. Response A
3. Response B

Now provide your evaluation and ranking:`, userQuery, responsesText.String())

	// Create messages
	messages := []OpenRouterMessage{
		{Role: "user", Content: rankingPrompt},
	}

	// Query all models in parallel
	responses, err := QueryModelsParallel(ctx, CouncilModels, messages)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query models for rankings: %w", err)
	}

	// Format results
	var stage2Results []Stage2Ranking
	for model, response := range responses {
		if response != nil {
			fullText := response.Content
			parsed := ParseRankingFromText(fullText)
			stage2Results = append(stage2Results, Stage2Ranking{
				Model:         model,
				Ranking:       fullText,
				ParsedRanking: parsed,
			})
		}
	}

	return stage2Results, labelToModel, nil
}

// Stage3SynthesizeFinal synthesizes the final response using the chairman model.
// This is the final stage where the chairman reviews all responses and rankings
// to produce a comprehensive answer. Returns the synthesized response or an error.
func Stage3SynthesizeFinal(ctx context.Context, userQuery string, stage1Results []Stage1Response, stage2Results []Stage2Ranking) (*Stage3Response, error) {
	// Build comprehensive context with all stage1 results
	var stage1Text strings.Builder
	for _, result := range stage1Results {
		stage1Text.WriteString(fmt.Sprintf("Model: %s\nResponse: %s\n\n", result.Model, result.Response))
	}

	// Build stage2 rankings text
	var stage2Text strings.Builder
	for _, result := range stage2Results {
		stage2Text.WriteString(fmt.Sprintf("Model: %s\nRanking: %s\n\n", result.Model, result.Ranking))
	}

	// Create chairman prompt
	chairmanPrompt := fmt.Sprintf(`You are the Chairman of an LLM Council. Multiple AI models have provided responses to a user's question, and then ranked each other's responses.

Original Question: %s

STAGE 1 - Individual Responses:
%s

STAGE 2 - Peer Rankings:
%s

Your task as Chairman is to synthesize all of this information into a single, comprehensive, accurate answer to the user's original question. Consider:
- The individual responses and their insights
- The peer rankings and what they reveal about response quality
- Any patterns of agreement or disagreement

Provide a clear, well-reasoned final answer that represents the council's collective wisdom:`, userQuery, stage1Text.String(), stage2Text.String())

	// Create messages
	messages := []OpenRouterMessage{
		{Role: "user", Content: chairmanPrompt},
	}

	// Query chairman model
	response, err := QueryModel(ctx, ChairmanModel, messages, ModelQueryTimeout)
	if err != nil {
		return nil, fmt.Errorf("chairman model query failed: %w", err)
	}

	return &Stage3Response{
		Model:    ChairmanModel,
		Response: response.Content,
	}, nil
}

// ParseRankingFromText extracts the ranking from a model's response text.
// Looks for a "FINAL RANKING:" section and parses numbered responses (e.g., "1. Response A").
// Falls back to extracting any "Response X" patterns found in the text.
func ParseRankingFromText(rankingText string) []string {
	// Look for "FINAL RANKING:" section
	if strings.Contains(rankingText, "FINAL RANKING:") {
		parts := strings.Split(rankingText, "FINAL RANKING:")
		if len(parts) >= 2 {
			rankingSection := parts[1]

			// Try to extract numbered list format (e.g., "1. Response A")
			numberedPattern := regexp.MustCompile(`\d+\.\s*Response [A-Z]`)
			numberedMatches := numberedPattern.FindAllString(rankingSection, -1)
			if len(numberedMatches) > 0 {
				// Extract just the "Response X" part
				responsePattern := regexp.MustCompile(`Response [A-Z]`)
				var results []string
				for _, match := range numberedMatches {
					if resp := responsePattern.FindString(match); resp != "" {
						results = append(results, resp)
					}
				}
				return results
			}

			// Fallback: Extract all "Response X" patterns in order
			responsePattern := regexp.MustCompile(`Response [A-Z]`)
			matches := responsePattern.FindAllString(rankingSection, -1)
			if len(matches) > 0 {
				return matches
			}
		}
	}

	// Fallback: try to find any "Response X" patterns in order
	responsePattern := regexp.MustCompile(`Response [A-Z]`)
	matches := responsePattern.FindAllString(rankingText, -1)
	return matches
}

// CalculateAggregateRankings computes aggregate rankings across all models.
// Calculates the average rank position for each model based on peer rankings.
// Returns a slice of aggregate rankings sorted by average rank (lower is better).
func CalculateAggregateRankings(stage2Results []Stage2Ranking, labelToModel map[string]string) []AggregateRanking {
	// Track positions for each model
	modelPositions := make(map[string][]int)

	for _, ranking := range stage2Results {
		parsed := ranking.ParsedRanking

		for position, label := range parsed {
			if modelName, ok := labelToModel[label]; ok {
				modelPositions[modelName] = append(modelPositions[modelName], position+1) // position+1 because 0-indexed
			}
		}
	}

	// Calculate average position for each model
	var aggregate []AggregateRanking
	for model, positions := range modelPositions {
		if len(positions) > 0 {
			sum := 0
			for _, pos := range positions {
				sum += pos
			}
			avgRank := float64(sum) / float64(len(positions))

			aggregate = append(aggregate, AggregateRanking{
				Model:         model,
				AverageRank:   avgRank,
				RankingsCount: len(positions),
			})
		}
	}

	// Sort by average rank (lower is better)
	sort.Slice(aggregate, func(i, j int) bool {
		return aggregate[i].AverageRank < aggregate[j].AverageRank
	})

	return aggregate
}

// GenerateConversationTitle generates a short title for a conversation.
// Uses a fast model (gemini-2.5-flash) to create a 3-5 word summary of the user's query.
// Returns the generated title or an error if generation fails.
func GenerateConversationTitle(ctx context.Context, userQuery string) (string, error) {
	titlePrompt := fmt.Sprintf(`Generate a very short title (3-5 words maximum) that summarizes the following question.
The title should be concise and descriptive. Do not use quotes or punctuation in the title.

Question: %s

Title:`, userQuery)

	messages := []OpenRouterMessage{
		{Role: "user", Content: titlePrompt},
	}

	// Use gemini-2.5-flash for fast title generation
	response, err := QueryModel(ctx, "google/gemini-2.5-flash", messages, TitleGenTimeout)
	if err != nil {
		return "", fmt.Errorf("title generation failed: %w", err)
	}

	title := strings.TrimSpace(response.Content)

	// Clean up the title - remove quotes
	title = strings.Trim(title, "\"'")

	// Truncate if too long
	if len(title) > 50 {
		title = title[:47] + "..."
	}

	return title, nil
}

// RunFullCouncil runs the complete 3-stage council process.
// Orchestrates all three stages: parallel model queries, anonymized peer review,
// and chairman synthesis. Returns results from all stages plus metadata including
// rankings and label mappings, or an error if any critical stage fails.
func RunFullCouncil(ctx context.Context, userQuery string) ([]Stage1Response, []Stage2Ranking, Stage3Response, Metadata, error) {
	// Stage 1: Collect responses
	stage1Results, err := Stage1CollectResponses(ctx, userQuery)
	if err != nil {
		return nil, nil, Stage3Response{}, Metadata{}, fmt.Errorf("stage 1 failed: %w", err)
	}

	// If no models responded successfully, return error
	if len(stage1Results) == 0 {
		return nil, nil, Stage3Response{}, Metadata{},
			fmt.Errorf("all council models failed to respond")
	}

	// Stage 2: Collect rankings
	stage2Results, labelToModel, err := Stage2CollectRankings(ctx, userQuery, stage1Results)
	if err != nil {
		return nil, nil, Stage3Response{}, Metadata{}, fmt.Errorf("stage 2 failed: %w", err)
	}

	// Calculate aggregate rankings
	aggregateRankings := CalculateAggregateRankings(stage2Results, labelToModel)

	// Stage 3: Synthesize final answer
	stage3Result, err := Stage3SynthesizeFinal(ctx, userQuery, stage1Results, stage2Results)
	if err != nil {
		return nil, nil, Stage3Response{}, Metadata{}, fmt.Errorf("stage 3 failed: %w", err)
	}

	// Build metadata
	metadata := Metadata{
		LabelToModel:      labelToModel,
		AggregateRankings: aggregateRankings,
	}

	return stage1Results, stage2Results, *stage3Result, metadata, nil
}
