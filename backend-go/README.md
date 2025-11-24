# LLM Council - Go Backend

This is the Go implementation of the LLM Council backend, replacing the Python FastAPI version.

## Project Structure

```
backend-go/
├── main.go          # HTTP server and route handlers (Gin framework)
├── config.go        # Configuration and environment variables
├── models.go        # Data structure definitions (structs)
├── openrouter.go    # OpenRouter API client with parallel queries
├── council.go       # 3-stage council orchestration logic
├── storage.go       # JSON-based conversation persistence
├── go.mod           # Go module dependencies
└── go.sum           # Dependency checksums
```

## Prerequisites

- Go 1.21 or higher
- OpenRouter API key

## Setup

1. **Ensure `.env` file exists in project root** (one directory up):
   ```bash
   OPENROUTER_API_KEY=sk-or-v1-...
   ```

2. **Install dependencies** (already done if you ran Phase 1):
   ```bash
   go mod download
   ```

## Building

```bash
# Build binary
go build -o llm-council

# The binary will be created in the current directory
```

## Running

```bash
# Option 1: Run directly with go run
go run .

# Option 2: Build and run binary
go build -o llm-council
./llm-council
```

The server will start on **http://localhost:8001**

## Development

### Running with auto-reload

Install Air for hot-reload during development:
```bash
go install github.com/air-verse/air@latest
air
```

### Code formatting

```bash
# Format all Go files
go fmt ./...

# Run Go linter
go vet ./...
```

### Testing

```bash
# Run all tests (when implemented)
go test ./...

# Run with race detector
go test -race ./...

# Run with coverage
go test -cover ./...
```

## Migration Status

**Phase 1: Setup - ✅ COMPLETE**
- [x] Project structure created
- [x] Go module initialized
- [x] Dependencies added (Gin, CORS, godotenv, errgroup, uuid)
- [x] All files scaffolded with TODOs
- [x] Project compiles successfully

**Phase 2-6: Implementation - ✅ COMPLETE**
- [x] Phase 2: Implement models.go and config.go
- [x] Phase 3: Implement openrouter.go (HTTP client)
- [x] Phase 4: Implement storage.go (JSON persistence)
- [x] Phase 5: Implement council.go (3-stage logic)
- [x] Phase 6: Implement main.go (HTTP handlers)

**Phase 7-8: Testing & Integration - 🚧 READY FOR TESTING**
- [ ] Phase 7: Testing with real API calls
- [ ] Phase 8: Integration with frontend and cutover

## API Endpoints

The Go backend provides:

- `GET /` - Health check
- `GET /api/conversations` - List all conversations
- `POST /api/conversations` - Create new conversation
- `GET /api/conversations/:id` - Get conversation by ID
- `POST /api/conversations/:id/message` - Send message (batch)
- `POST /api/conversations/:id/message/stream` - Send message (SSE streaming)

## Key Dependencies

- **github.com/gin-gonic/gin** - Web framework (FastAPI equivalent)
- **github.com/gin-contrib/cors** - CORS middleware
- **github.com/joho/godotenv** - Environment variable loading
- **golang.org/x/sync/errgroup** - Parallel execution with error handling

## Architecture Notes

- **Flat package structure**: All code in `main` package for simplicity
- **Parallel queries**: Uses goroutines + errgroup for Stage 1 & 2
- **Graceful degradation**: Continues if some models fail
- **SSE streaming**: Server-Sent Events for real-time updates
- **JSON storage**: File-based persistence in `data/conversations/`

## Next Steps

Continue with Phase 2 of the migration plan to implement:
1. Configuration loading (config.go)
2. Data models and JSON tags (models.go)
3. OpenRouter API client (openrouter.go)
4. Storage layer (storage.go)
5. Council orchestration (council.go)
6. HTTP handlers (main.go)

See the main project CLAUDE.md for detailed implementation guidance.
