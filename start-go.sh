#!/bin/bash

# LLM Council - Start script (Go backend)

echo "Starting LLM Council with Go backend..."
echo ""

# Check if .env exists
if [ ! -f .env ]; then
    echo "❌ Error: .env file not found"
    echo "Please create .env file with your OPENROUTER_API_KEY"
    echo "See .env.example for template"
    exit 1
fi

# Build Go backend if needed
if [ ! -f backend-go/llm-council ]; then
    echo "Building Go backend..."
    cd backend-go
    export GOROOT=/opt/homebrew/Cellar/go/1.25.4/libexec
    go build -o llm-council
    cd ..
fi

# Start Go backend
echo "Starting Go backend on http://localhost:8001..."
cd backend-go
./llm-council &
BACKEND_PID=$!
cd ..

# Wait a bit for backend to start
sleep 2

# Start frontend
echo "Starting frontend on http://localhost:5173..."
cd frontend
npm run dev &
FRONTEND_PID=$!
cd ..

echo ""
echo "✓ LLM Council is running with Go backend!"
echo "  Backend:  http://localhost:8001"
echo "  Frontend: http://localhost:5173"
echo ""
echo "Press Ctrl+C to stop both servers"

# Wait for Ctrl+C
trap "kill $BACKEND_PID $FRONTEND_PID 2>/dev/null; exit" SIGINT SIGTERM
wait
