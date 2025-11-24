#!/bin/bash

# Run tests with coverage
GOROOT=/opt/homebrew/Cellar/go/1.25.4/libexec go test -coverprofile=coverage.out

# Show total coverage
echo "=== Total Coverage ==="
GOROOT=/opt/homebrew/Cellar/go/1.25.4/libexec go tool cover -func=coverage.out | tail -1

# Show coverage excluding main function (which cannot be tested)
echo ""
echo "=== Coverage Excluding main() Function ==="
GOROOT=/opt/homebrew/Cellar/go/1.25.4/libexec go tool cover -func=coverage.out | grep -v "main.go:15" | tail -1

# Show detailed breakdown
echo ""
echo "=== Detailed Coverage by File ==="
GOROOT=/opt/homebrew/Cellar/go/1.25.4/libexec go tool cover -func=coverage.out | grep -E "\.go:" | awk '{print $1 " " $3}' | sort -t: -k1,1 -k2,2n

# Generate HTML report
echo ""
echo "Generating HTML coverage report..."
GOROOT=/opt/homebrew/Cellar/go/1.25.4/libexec go tool cover -html=coverage.out -o coverage.html
echo "HTML report generated: coverage.html"
