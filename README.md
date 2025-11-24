# LLM Council

![llmcouncil](header.jpg)

A local web application that queries multiple LLMs simultaneously, has them anonymously review each other's responses, and synthesizes a final answer. Instead of asking one LLM, get the collective wisdom of your "LLM Council."

## What It Does

When you submit a query:

1. **Stage 1: Parallel Queries** - All council LLMs receive your question and respond independently
2. **Stage 2: Anonymous Peer Review** - Each LLM ranks the others' responses (identities hidden to prevent bias)
3. **Stage 3: Final Synthesis** - A Chairman LLM compiles all responses into one comprehensive answer

You see all individual responses, peer rankings, and the final synthesis in a ChatGPT-like interface.

## Quick Start (5 minutes)

**Prerequisites:** Go 1.21+, Node.js 18+, npm

```bash
# 1. Clone and enter directory
git clone <repo-url>
cd llm-senate-council-upgrade

# 2. Get OpenRouter API key at https://openrouter.ai/
cp .env.example .env
# Edit .env and add your key: OPENROUTER_API_KEY=sk-or-v1-...

# 3. Build backend and install frontend dependencies
cd backend-go && go build -o llm-council && cd ..
cd frontend && npm install && cd ..

# 4. Start everything (single command)
./start-go.sh
```

Open http://localhost:5173 and start chatting with your LLM Council.

## Vibe Code Alert

This project was 99% vibe coded as a fun Saturday hack for exploring and evaluating multiple LLMs side by side while [reading books together with LLMs](https://x.com/karpathy/status/1990577951671509438). It's provided as-is for inspiration. Code is ephemeral now and libraries are over - ask your LLM to modify it however you like.

## Detailed Setup

### Prerequisites

- **Go** 1.21 or higher ([install](https://go.dev/doc/install))
- **Node.js** 18+ and npm ([install](https://nodejs.org/))
- **OpenRouter API key** with credits ([get one](https://openrouter.ai/))

### 1. Install Dependencies

**Go Backend:**
```bash
cd backend-go
go mod download
go build -o llm-council
cd ..
```

**Frontend:**
```bash
cd frontend
npm install
cd ..
```

### 2. Configure API Key

Create `.env` in project root:

```bash
cp .env.example .env
```

Edit `.env` and add your OpenRouter API key:
```bash
OPENROUTER_API_KEY=sk-or-v1-your-actual-key-here
```

Get your API key at [openrouter.ai](https://openrouter.ai/). Make sure to purchase credits or enable automatic top-up.

### 3. Configure Models (Optional)

Default council uses GPT-4, Claude, Gemini, and Grok. To customize, edit `backend-go/config.go`:

```go
var CouncilModels = []string{
    "openai/gpt-5.1",
    "google/gemini-3-pro-preview",
    "anthropic/claude-sonnet-4.5",
    "x-ai/grok-4",
}

var ChairmanModel = "google/gemini-3-pro-preview"
```

After changes, rebuild: `cd backend-go && go build -o llm-council && cd ..`

## Running the Application

**Option 1: Automated Start (Recommended)**
```bash
./start-go.sh
```

This script starts both the Go backend (port 8001) and frontend (port 5173). Press Ctrl+C to stop both servers.

**Option 2: Manual Start**
```bash
# Terminal 1 - Backend
cd backend-go && ./llm-council

# Terminal 2 - Frontend
cd frontend && npm run dev
```

Then open http://localhost:5173 in your browser.

**Rebuilding After Code Changes:**
```bash
cd backend-go
go build -o llm-council
cd ..
```

## Architecture

### Backend (Go)

Single-binary server built with Gin framework:
- **Port:** 8001
- **Concurrency:** Native goroutines for parallel LLM queries
- **Storage:** JSON files in `data/conversations/`
- **API:** RESTful endpoints + Server-Sent Events for streaming

Key files:
- `main.go` - HTTP server and route handlers
- `council.go` - 3-stage orchestration logic
- `openrouter.go` - API client with parallel execution
- `storage.go` - JSON persistence layer

See `backend-go/README.md` for detailed architecture.

### Frontend (React)

Vite-powered SPA with tab-based interface:
- **Port:** 5173 (development)
- **Key components:** ChatInterface, Stage1/2/3 display, Conversation list
- **Styling:** Light theme, markdown rendering with syntax highlighting

### Tech Stack

- **Backend:** Go 1.21+, Gin web framework, OpenRouter API
- **Frontend:** React 19, Vite, react-markdown
- **Storage:** JSON files (no database required)
- **APIs:** OpenRouter for multi-model access

## Troubleshooting

### "Error: .env file not found"
Create `.env` in project root with `OPENROUTER_API_KEY=sk-or-v1-...`

### Backend fails to start
- Check if port 8001 is already in use: `lsof -i :8001`
- Verify API key is valid at openrouter.ai
- Ensure Go 1.21+ is installed: `go version`

### Frontend shows "Failed to fetch"
- Verify backend is running on http://localhost:8001
- Check browser console for CORS errors
- Try restarting both backend and frontend

### Build errors
```bash
# Clean and rebuild
cd backend-go
rm llm-council
go clean
go build -o llm-council
```

### Models not responding
- Check OpenRouter credits at openrouter.ai
- Verify model names in `backend-go/config.go` match OpenRouter's API
- Check backend logs for specific API errors

### Permission denied on start-go.sh
```bash
chmod +x start-go.sh
```

## Project Structure

```
llm-senate-council-upgrade/
├── backend-go/           # Go backend (recommended)
│   ├── main.go          # HTTP server
│   ├── council.go       # 3-stage logic
│   ├── openrouter.go    # API client
│   ├── storage.go       # JSON persistence
│   ├── config.go        # Model configuration
│   └── models.go        # Data structures
├── frontend/            # React frontend
│   ├── src/
│   │   ├── App.jsx
│   │   ├── ChatInterface.jsx
│   │   └── components/
│   └── package.json
├── data/                # Generated: conversation storage
├── .env                 # Your API key (create from .env.example)
├── .env.example         # Template
└── start-go.sh          # Launch script
```

## Contributing

This is a "vibe code" project - feel free to fork and modify as you see fit. No formal contribution process or support provided.
