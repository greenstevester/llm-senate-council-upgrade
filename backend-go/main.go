package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Global bills cache instance
var billsCache *BillsCache

func main() {
	// Load configuration
	LoadConfig()

	// Initialize bills cache
	billsCache = NewBillsCache(BillsCacheTTL)

	// Create Gin router
	router := gin.Default()

	// Request size limit middleware
	router.Use(func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxRequestBodySize)
		c.Next()
	})

	// CORS middleware with dynamic origin validation
	router.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			// In production, use environment-configured origins
			if len(CORSAllowedOrigins) > 0 && CORSAllowedOrigins[0] != "" {
				for _, allowedOrigin := range CORSAllowedOrigins {
					if origin == allowedOrigin {
						return true
					}
				}
				return false
			}
			// In development, allow any localhost/127.0.0.1 origin
			return len(origin) > 0 && (
				len(origin) >= 16 && origin[:16] == "http://localhost" ||
				len(origin) >= 14 && origin[:14] == "http://127.0.0")
		},
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type"},
		AllowCredentials: true,
	}))

	// Routes
	router.GET("/", healthCheck)
	router.GET("/api/conversations", listConversationsHandler)
	router.POST("/api/conversations", createConversationHandler)
	router.GET("/api/conversations/:id", getConversationHandler)
	router.POST("/api/conversations/:id/message", sendMessageHandler)
	router.POST("/api/conversations/:id/message/stream", sendMessageStreamHandler)
	router.GET("/api/bills", getBillsHandler)
	router.POST("/api/fetch-url", fetchURLHandler)

	// Start server
	log.Println("Starting LLM Council backend on port 8001...")
	if err := router.Run(":8001"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// healthCheck returns a simple health check response.
// GET / - Returns service status information.
func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "LLM Council API",
	})
}

// listConversationsHandler lists all conversations with metadata only.
// GET /api/conversations - Returns array of conversation metadata sorted by date.
func listConversationsHandler(c *gin.Context) {
	conversations, err := ListConversations()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to list conversations: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, conversations)
}

// createConversationHandler creates a new conversation.
// POST /api/conversations - Generates a new UUID and creates an empty conversation.
func createConversationHandler(c *gin.Context) {
	// Generate new UUID
	conversationID := uuid.New().String()

	// Create conversation
	conversation, err := CreateConversation(conversationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to create conversation: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, conversation)
}

// getConversationHandler gets a specific conversation by ID.
// GET /api/conversations/:id - Returns full conversation including all messages.
func getConversationHandler(c *gin.Context) {
	conversationID := c.Param("id")

	conversation, err := GetConversation(conversationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to get conversation: %v", err),
		})
		return
	}

	if conversation == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Conversation not found",
		})
		return
	}

	c.JSON(http.StatusOK, conversation)
}

// sendMessageHandler sends a message and runs the 3-stage council process.
// POST /api/conversations/:id/message - Runs full council and returns all stages at once.
// Use sendMessageStreamHandler for SSE streaming version.
func sendMessageHandler(c *gin.Context) {
	conversationID := c.Param("id")

	// Parse request
	var request SendMessageRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	// Check if conversation exists
	conversation, err := GetConversation(conversationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to get conversation: %v", err),
		})
		return
	}
	if conversation == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Conversation not found",
		})
		return
	}

	// Check if this is the first message
	isFirstMessage := len(conversation.Messages) == 0

	// Add user message
	if err := AddUserMessage(conversationID, request.Content); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to add user message: %v", err),
		})
		return
	}

	// Generate title if first message (run in background)
	if isFirstMessage {
		go func() {
			ctx := context.Background()
			title, err := GenerateConversationTitle(ctx, request.Content)
			if err != nil {
				log.Printf("Failed to generate title: %v", err)
				// Use default title on error
				UpdateConversationTitle(conversationID, "New Conversation")
			} else {
				UpdateConversationTitle(conversationID, title)
			}
		}()
	}

	// Run the 3-stage council process
	ctx := context.Background()
	stage1, stage2, stage3, metadata, err := RunFullCouncil(ctx, request.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Council process failed: %v", err),
		})
		return
	}

	// Add assistant message
	if err := AddAssistantMessage(conversationID, stage1, stage2, stage3); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to add assistant message: %v", err),
		})
		return
	}

	// Return response
	c.JSON(http.StatusOK, SendMessageResponse{
		Stage1:   stage1,
		Stage2:   stage2,
		Stage3:   stage3,
		Metadata: metadata,
	})
}

// sendMessageStreamHandler sends a message and streams the 3-stage council process via SSE.
// POST /api/conversations/:id/message/stream - Streams progress events as each stage completes.
// Events: stage1_start, stage1_complete, stage2_start, stage2_complete, stage3_start, stage3_complete, complete.
func sendMessageStreamHandler(c *gin.Context) {
	conversationID := c.Param("id")

	// Parse request
	var request SendMessageRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	// Check if conversation exists
	conversation, err := GetConversation(conversationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to get conversation: %v", err),
		})
		return
	}
	if conversation == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Conversation not found",
		})
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	// Check if this is the first message
	isFirstMessage := len(conversation.Messages) == 0

	// Add user message
	if err := AddUserMessage(conversationID, request.Content); err != nil {
		sendSSEError(c, fmt.Sprintf("Failed to add user message: %v", err))
		return
	}

	ctx := context.Background()

	// Start title generation in background if first message
	var titleChan chan string
	if isFirstMessage {
		titleChan = make(chan string, 1)
		go func() {
			title, err := GenerateConversationTitle(ctx, request.Content)
			if err != nil {
				log.Printf("Failed to generate title: %v", err)
				UpdateConversationTitle(conversationID, "New Conversation")
			} else {
				UpdateConversationTitle(conversationID, title)
				titleChan <- title
			}
			close(titleChan)
		}()
	}

	// Stage 1
	sendSSEEvent(c, gin.H{"type": "stage1_start"})
	stage1, err := Stage1CollectResponses(ctx, request.Content)
	if err != nil {
		sendSSEError(c, fmt.Sprintf("Stage 1 failed: %v", err))
		return
	}
	sendSSEEvent(c, gin.H{"type": "stage1_complete", "data": stage1})

	// Stage 2
	sendSSEEvent(c, gin.H{"type": "stage2_start"})
	stage2, labelToModel, err := Stage2CollectRankings(ctx, request.Content, stage1)
	if err != nil {
		sendSSEError(c, fmt.Sprintf("Stage 2 failed: %v", err))
		return
	}
	aggregateRankings := CalculateAggregateRankings(stage2, labelToModel)
	sendSSEEvent(c, gin.H{
		"type": "stage2_complete",
		"data": stage2,
		"metadata": gin.H{
			"label_to_model":      labelToModel,
			"aggregate_rankings":  aggregateRankings,
		},
	})

	// Stage 3
	sendSSEEvent(c, gin.H{"type": "stage3_start"})
	stage3, err := Stage3SynthesizeFinal(ctx, request.Content, stage1, stage2)
	if err != nil {
		sendSSEError(c, fmt.Sprintf("Stage 3 failed: %v", err))
		return
	}
	sendSSEEvent(c, gin.H{"type": "stage3_complete", "data": stage3})

	// Wait for title if it was being generated
	if titleChan != nil {
		if title := <-titleChan; title != "" {
			sendSSEEvent(c, gin.H{"type": "title_complete", "data": gin.H{"title": title}})
		}
	}

	// Save complete assistant message (check for nil first)
	if stage3 == nil {
		sendSSEError(c, "Stage 3 returned no result")
		return
	}
	if err := AddAssistantMessage(conversationID, stage1, stage2, *stage3); err != nil {
		sendSSEError(c, fmt.Sprintf("Failed to save message: %v", err))
		return
	}

	// Send completion event
	sendSSEEvent(c, gin.H{"type": "complete"})
}

// sendSSEEvent sends a Server-Sent Event.
// Marshals data to JSON and writes as SSE format with "data: " prefix.
func sendSSEEvent(c *gin.Context, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Failed to marshal SSE event: %v", err)
		return
	}
	c.Writer.WriteString(fmt.Sprintf("data: %s\n\n", string(jsonData)))
	c.Writer.Flush()
}

// sendSSEError sends an error event via SSE.
// Convenience wrapper for sending error-type SSE events.
func sendSSEError(c *gin.Context, message string) {
	sendSSEEvent(c, gin.H{"type": "error", "message": message})
}

// getBillsHandler fetches and returns all bills before parliament
// GET /api/bills - Returns all bills with caching
// Query params: ?refresh=true (force cache refresh)
func getBillsHandler(c *gin.Context) {
	// Check for refresh parameter
	forceRefresh := c.Query("refresh") == "true"

	// Try to get from cache first (unless refresh requested)
	if !forceRefresh {
		if cachedBills, ok := billsCache.Get(); ok {
			log.Printf("Returning %d bills from cache", len(cachedBills))
			c.JSON(http.StatusOK, BillsResponse{
				Bills:       cachedBills,
				CurrentPage: 1,
				TotalPages:  CalculateTotalPages(len(cachedBills)),
				HasNextPage: false,
				LastUpdated: billsCache.GetLastUpdated(),
			})
			return
		}
	}

	// Fetch fresh data
	log.Println("Fetching fresh bills data from APH website...")
	ctx := context.Background()
	bills, err := FetchAllBills(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to fetch bills: %v", err),
		})
		return
	}

	// Update cache
	billsCache.Set(bills)
	log.Printf("Cached %d bills", len(bills))

	// Return response
	c.JSON(http.StatusOK, BillsResponse{
		Bills:       bills,
		CurrentPage: 1,
		TotalPages:  CalculateTotalPages(len(bills)),
		HasNextPage: false,
		LastUpdated: time.Now(),
	})
}

// fetchURLHandler fetches and extracts content from a given URL
// POST /api/fetch-url - Body: {"url": "https://..."}
func fetchURLHandler(c *gin.Context) {
	// Parse request
	var request struct {
		URL string `json:"url" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	// Fetch content
	ctx := context.Background()
	content, err := FetchURLContent(ctx, request.URL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to fetch URL content: %v", err),
		})
		return
	}

	// Return content
	c.JSON(http.StatusOK, gin.H{
		"content": content,
	})
}
