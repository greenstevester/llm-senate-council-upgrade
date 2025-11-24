# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

LLM Council is a 3-stage deliberation system where multiple LLMs collaboratively answer user questions through:
1. **Stage 1**: Parallel queries to all council models
2. **Stage 2**: Anonymized peer review and ranking (prevents bias)
3. **Stage 3**: Chairman synthesizes final answer from all context

## Development Commands

**Backend Options:**
- **Go backend** (Recommended): `./start-go.sh` - Fast, single binary, production-ready
- **Python backend** (Legacy): `./start.sh` - Original implementation

**Setup:**
```bash
# Backend - Go (recommended)
cd backend-go
go build -o llm-council

# Backend - Python (legacy)
uv sync

# Frontend dependencies
cd frontend && npm install

# Environment
cp .env.example .env
# Edit .env and add your OPENROUTER_API_KEY
```

**Running with Go backend (Recommended):**
```bash
# Option 1: Use start script
./start-go.sh

# Option 2: Run manually
# Terminal 1 - Go Backend (port 8001)
cd backend-go && ./llm-council

# Terminal 2 - Frontend (port 5173)
cd frontend && npm run dev
```

**Running with Python backend (Legacy):**
```bash
# Option 1: Use start script
./start.sh

# Option 2: Run manually
# Terminal 1 - Python Backend (port 8001)
uv run python -m backend.main

# Terminal 2 - Frontend (port 5173)
cd frontend && npm run dev
```

**Frontend commands:**
```bash
cd frontend
npm run dev      # Development server
npm run build    # Production build
npm run lint     # ESLint
npm run preview  # Preview production build
```

**Environment:**
- Create `.env` in project root with `OPENROUTER_API_KEY=sk-or-v1-...`
- Get API key at openrouter.ai

## Architecture

### Go Backend (`backend-go/`) - Recommended

The Go backend is a complete reimplementation with improved performance and simpler deployment.

**Module structure:**

- **`main.go`** (303 LOC): Gin-based HTTP server with CORS, all route handlers, SSE streaming
- **`config.go`** (47 LOC): Configuration loading, model lists, environment variables
- **`models.go`** (106 LOC): Complete type system with JSON serialization tags
- **`openrouter.go`** (123 LOC): OpenRouter HTTP client
  - `QueryModel()`: Single model query with context and timeout
  - `QueryModelsParallel()`: Goroutine-based parallel execution with errgroup
  - Thread-safe with sync.Mutex, graceful degradation on failures
- **`council.go`** (318 LOC): 3-stage orchestration
  - `Stage1CollectResponses()`: Parallel model queries
  - `Stage2CollectRankings()`: Anonymization and peer review
  - `Stage3SynthesizeFinal()`: Chairman synthesis
  - `ParseRankingFromText()`: Regex-based ranking extraction
  - `CalculateAggregateRankings()`: Statistical aggregation
  - `RunFullCouncil()`: Complete end-to-end orchestration
- **`storage.go`** (202 LOC): JSON file persistence
  - Same data format as Python backend (compatible with existing conversations)
  - All CRUD operations for conversations and messages

**Key advantages:**
- Single 27MB binary, no runtime dependencies
- Goroutine-based concurrency (faster than Python's asyncio)
- Type-safe with compile-time checks
- Lower memory footprint (~20-30MB vs ~100MB+)
- Instant startup (vs ~1s for Python)

### Python Backend (`backend/`) - Legacy

The backend uses **relative imports** throughout (e.g., `from .config import ...`). Always run as `python -m backend.main` from project root, never from within the backend directory.

**Module responsibilities:**

- **`config.py`**: Defines `COUNCIL_MODELS` list and `CHAIRMAN_MODEL`, loads API key from environment
- **`openrouter.py`**:
  - `query_model()`: Single async model query
  - `query_models_parallel()`: Parallel queries using `asyncio.gather()`
  - Returns dict with 'content' and optional 'reasoning_details'
  - Graceful degradation: returns None on failure

- **`council.py`**: Core 3-stage logic
  - `stage1_collect_responses()`: Parallel queries to all models
  - `stage2_collect_rankings()`:
    - Anonymizes responses as "Response A, B, C..."
    - Returns tuple: (rankings_list, label_to_model_dict)
    - Each ranking includes both raw text and `parsed_ranking` list
  - `stage3_synthesize_final()`: Chairman synthesizes from all context
  - `parse_ranking_from_text()`: Extracts "FINAL RANKING:" section with regex fallbacks
  - `calculate_aggregate_rankings()`: Computes average rank position
  - `generate_conversation_title()`: Uses gemini-2.5-flash for fast title generation

- **`storage.py`**: JSON-based persistence in `data/conversations/`
  - Each conversation: `{id, created_at, title, messages[]}`
  - Assistant messages contain: `{role, stage1, stage2, stage3}`
  - **Important**: Metadata (label_to_model, aggregate_rankings) is NOT persisted, only returned via API

- **`main.py`**: FastAPI app
  - CORS enabled for localhost:5173 and localhost:3000
  - POST `/api/conversations/{id}/message`: Batch endpoint returning all stages at once
  - POST `/api/conversations/{id}/message/stream`: Server-Sent Events streaming endpoint
  - Both endpoints return metadata in addition to stage results

### Frontend (`frontend/src/`)

React + Vite app with light mode theme (primary color: #4a90e2).

**Component structure:**

- **`App.jsx`**: Main orchestration, manages conversations and current conversation state
- **`ChatInterface.jsx`**: Multiline textarea (Enter to send, Shift+Enter for newline)
- **`Stage1.jsx`**: Tab view of individual model responses
- **`Stage2.jsx`**:
  - **Critical**: Shows RAW evaluation text from each model
  - De-anonymization happens CLIENT-SIDE for display (models received anonymous labels)
  - Shows "Extracted Ranking" below each evaluation for validation
  - Aggregate rankings displayed with average position
- **`Stage3.jsx`**: Final synthesis with green-tinted background (#f0fff0)

**Styling**:
- All markdown content must be wrapped in `<div className="markdown-content">`
- This class provides 12px padding and is defined in `index.css`

## Critical Implementation Details

### Backend Port
Backend runs on **port 8001** (not 8000 - user had another app on port 8000). Update both `backend/main.py` and `frontend/src/api.js` if changing ports.

### Stage 2 Prompt Format
Stage 2 ranking prompt requires strict format for reliable parsing:
1. Individual evaluations first
2. Must include "FINAL RANKING:" header (all caps with colon)
3. Numbered list: "1. Response C", "2. Response A", etc.
4. No text after ranking section

### De-anonymization Strategy
- Models receive: "Response A", "Response B", etc.
- Backend creates mapping: `{"Response A": "openai/gpt-5.1", ...}`
- Frontend displays model names in **bold** for readability
- This prevents bias during ranking while maintaining transparency

### Error Handling Philosophy
- Continue with successful responses if some models fail (graceful degradation)
- Never fail entire request due to single model failure
- Log errors but don't expose to user unless all models fail

### Model Configuration
Models are hardcoded in `backend/config.py`. To add/change council members or chairman, edit that file directly. Chairman can be same as or different from council members.

### Root main.py
The `main.py` in project root is just a placeholder from the uv template - the actual backend entrypoint is `backend/main.py`.

## Common Issues

1. **Module Import Errors**: Run backend as `python -m backend.main` from project root, not from backend directory
2. **CORS Issues**: If adding new frontend origin, update CORS middleware in `backend/main.py`
3. **Ranking Parse Failures**: Fallback regex extracts any "Response X" patterns if models don't follow format
4. **Missing Metadata**: Metadata is ephemeral (not persisted to JSON), only available in API responses

## Data Flow

```
User Query
    ↓
Stage 1: Parallel queries → [individual responses]
    ↓
Stage 2: Anonymize → Parallel ranking queries → [evaluations + parsed rankings]
    ↓
Aggregate Rankings Calculation → [sorted by avg position]
    ↓
Stage 3: Chairman synthesis with full context
    ↓
Return: {stage1, stage2, stage3, metadata}
    ↓
Frontend: Display with tabs + validation UI
```

The entire flow is async/parallel where possible to minimize latency.
