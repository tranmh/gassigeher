#!/bin/bash
# Gassigeher - Build and Test Script for Linux/Mac
# Usage: ./bat.sh

set -e  # Exit on error

echo "========================================"
echo "Gassigeher - Build and Test"
echo "========================================"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}[ERROR] Go is not installed or not in PATH${NC}"
    exit 1
fi

echo "[1/4] Checking Go version..."
go version
echo ""

echo "[2/4] Downloading dependencies..."
if go mod download; then
    echo -e "${GREEN}[OK] Dependencies downloaded${NC}"
else
    echo -e "${RED}[ERROR] Failed to download dependencies${NC}"
    exit 1
fi
echo ""

echo "[3/4] Building application..."
if go build -o gassigeher ./cmd/server; then
    chmod +x gassigeher
    echo -e "${GREEN}[OK] Build successful: gassigeher (pure Go SQLite)${NC}"
else
    echo -e "${RED}[ERROR] Build failed${NC}"
    exit 1
fi
echo ""

echo "[4/4] Running tests..."
if go test -v -cover ./...; then
    echo -e "${GREEN}[OK] All tests passed${NC}"
else
    echo -e "${YELLOW}[WARNING] Some tests failed${NC}"
fi
echo ""

echo "========================================"
echo "Build and Test Complete!"
echo "========================================"
echo ""
echo "To run the application:"
echo "  ./gassigeher"
echo ""
echo "To run with custom port:"
echo "  PORT=3000 ./gassigeher"
echo ""
