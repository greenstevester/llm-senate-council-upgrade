# LLM Council - Go Backend

High-performance backend server for LLM Council. Handles parallel model queries, peer review orchestration, and conversation persistence.

## Quick Start

```bash
# From project root, build the backend
cd backend-go
go build -o llm-council

# Run the server (needs .env in parent directory)
./llm-council
```

Server starts on http://localhost:8001

## Prerequisites

- **Go 1.21 or higher** - Check with `go version`
- **OpenRouter API key** - Get at [openrouter.ai](https://openrouter.ai/)
- **`.env` file in project root** (parent directory) with:
  ```bash
  OPENROUTER_API_KEY=sk-or-v1-...
  ```

## Project Structure

```
backend-go/
├── main.go          # HTTP server (Gin) and route handlers (303 LOC)
├── council.go       # 3-stage orchestration logic (318 LOC)
├── openrouter.go    # OpenRouter API client with parallel queries (123 LOC)
├── storage.go       # JSON conversation persistence (202 LOC)
├── config.go        # Environment and model configuration (47 LOC)
├── models.go        # Data structures with JSON tags (106 LOC)
├── go.mod           # Go module dependencies
└── go.sum           # Dependency checksums
```

Total: ~1,100 lines of production Go code.

## Building & Running

```bash
# Build the binary
go build -o llm-council

# Run the server
./llm-council
```

**Or run without building:**
```bash
go run .
```

The server starts on http://localhost:8001 and looks for `.env` in the parent directory.

## Configuration

### Model Selection

Edit the council members and chairman in `config.go`:

```go
var CouncilModels = []string{
    "openai/gpt-5.1",
    "google/gemini-3-pro-preview",
    "anthropic/claude-sonnet-4.5",
    "x-ai/grok-4",
}

var ChairmanModel = "google/gemini-3-pro-preview"
```

After editing, rebuild: `go build -o llm-council`

### Environment Variables

The backend looks for `.env` in the parent directory (project root):

```bash
OPENROUTER_API_KEY=sk-or-v1-...
```

## Development

### Hot Reload

Install [Air](https://github.com/air-verse/air) for automatic recompilation:
```bash
go install github.com/air-verse/air@latest
air
```

### Code Quality

```bash
# Format code
go fmt ./...

# Run linter
go vet ./...

# View dependencies
go mod graph
```

### Testing

```bash
# Run tests
go test ./...

# With race detection
go test -race ./...

# With coverage
go test -cover ./...
```

## API Endpoints

### Conversation Management
- `GET /` - Health check (returns "LLM Council API")
- `GET /api/conversations` - List all conversations
- `POST /api/conversations` - Create new conversation
- `GET /api/conversations/:id` - Get conversation by ID

### Message Processing
- `POST /api/conversations/:id/message` - Send message (batch mode, returns all stages at once)
- `POST /api/conversations/:id/message/stream` - Send message (SSE streaming, updates in real-time)

**Request body:**
```json
{
  "content": "Your question here"
}
```

**Response (batch):**
```json
{
  "stage1": [...],
  "stage2": [...],
  "stage3": {...},
  "metadata": {
    "label_to_model": {...},
    "aggregate_rankings": [...]
  }
}
```

## Architecture

### Core Components

**main.go** - HTTP server with Gin framework
- CORS middleware for frontend communication
- Route handlers for all endpoints
- SSE streaming implementation

**council.go** - 3-stage orchestration
- `Stage1CollectResponses()` - Parallel model queries
- `Stage2CollectRankings()` - Anonymized peer review
- `Stage3SynthesizeFinal()` - Chairman synthesis
- Statistical ranking aggregation

**openrouter.go** - API client
- `QueryModel()` - Single model query with timeout
- `QueryModelsParallel()` - Goroutine-based parallel execution
- Thread-safe with `sync.Mutex`
- Graceful degradation on failures

**storage.go** - JSON persistence
- File-based storage in `data/conversations/`
- CRUD operations for conversations and messages
- Compatible data format with Python backend (if it existed)

**config.go** - Configuration
- Environment variable loading
- Model list definitions
- OpenRouter API key management

**models.go** - Type system
- Complete Go structs with JSON tags
- Type-safe throughout the application

### Key Design Decisions

- **Single package** - All code in `main` package for simplicity
- **Goroutines** - Native concurrency for parallel LLM queries
- **Graceful degradation** - Continues with successful responses if some models fail
- **File-based storage** - No database required, JSON files in `data/`
- **SSE streaming** - Real-time updates for better UX

## Dependencies

- `github.com/gin-gonic/gin` - HTTP web framework
- `github.com/gin-contrib/cors` - CORS middleware
- `github.com/joho/godotenv` - .env file parsing
- `golang.org/x/sync/errgroup` - Parallel execution with error handling

## Troubleshooting

### "Failed to load .env"
The backend expects `.env` in the parent directory (project root), not in `backend-go/`.

### Port 8001 already in use
```bash
# Find and kill the process
lsof -i :8001
kill -9 <PID>
```

### Build fails
```bash
# Clean and rebuild
go clean
rm llm-council
go mod download
go build -o llm-council
```

### API errors from OpenRouter
- Verify your API key is valid
- Check you have sufficient credits
- Ensure model names match OpenRouter's API (check [openrouter.ai/models](https://openrouter.ai/models))

## Performance Characteristics

- **Startup:** <10ms (vs ~1s for Python)
- **Memory:** ~30MB (vs ~100MB for Python)
- **Binary size:** ~27MB (single file, no dependencies)
- **Concurrency:** Native goroutines (no async/await overhead)
