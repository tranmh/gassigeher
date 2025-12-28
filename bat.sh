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

echo "[1/6] Checking Go version..."
go version
echo ""

echo "[2/6] Compiling SCSS..."
if command -v npm &> /dev/null; then
    # Check if node_modules exists, install if not
    if [ ! -d "node_modules" ]; then
        echo "Installing npm dependencies..."
        npm install
    fi
    if npm run sass:prod; then
        echo -e "${GREEN}[OK] SCSS compiled successfully${NC}"
    else
        echo -e "${RED}[ERROR] SCSS compilation failed${NC}"
        exit 1
    fi
else
    echo -e "${YELLOW}[SKIP] npm not found - SCSS compilation skipped${NC}"
    echo -e "${YELLOW}       Make sure main.css exists or install npm${NC}"
fi
echo ""

echo "[3/6] Downloading dependencies..."
if go mod download; then
    echo -e "${GREEN}[OK] Dependencies downloaded${NC}"
else
    echo -e "${RED}[ERROR] Failed to download dependencies${NC}"
    exit 1
fi
echo ""

echo "[4/6] Building application..."
# Get version info for ldflags
VERSION="2.0"
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS="-X github.com/tranmh/gassigeher/internal/version.Version=${VERSION}"
LDFLAGS="${LDFLAGS} -X github.com/tranmh/gassigeher/internal/version.GitCommit=${GIT_COMMIT}"
LDFLAGS="${LDFLAGS} -X github.com/tranmh/gassigeher/internal/version.BuildTime=${BUILD_TIME}"

if go build -ldflags "${LDFLAGS}" -o gassigeher ./cmd/server; then
    chmod +x gassigeher
    echo -e "${GREEN}[OK] Build successful: gassigeher v${VERSION} (${GIT_COMMIT})${NC}"
else
    echo -e "${RED}[ERROR] Build failed${NC}"
    exit 1
fi
echo ""

echo "[5/6] Running Go tests (short mode - skips slow property tests)..."
if go test -short -cover ./...; then
    echo -e "${GREEN}[OK] All Go tests passed${NC}"
else
    echo -e "${YELLOW}[WARNING] Some Go tests failed${NC}"
fi
echo ""
echo "Note: Run 'go test ./...' without -short for full property-based tests (~5 min)"
echo ""

echo "[6/6] Running frontend tests..."
if command -v npm &> /dev/null; then
    # Check if node_modules exists, install if not
    if [ ! -d "node_modules" ]; then
        echo "Installing npm dependencies..."
        npm install
    fi
    if npm test; then
        echo -e "${GREEN}[OK] All frontend tests passed${NC}"
    else
        echo -e "${YELLOW}[WARNING] Some frontend tests failed${NC}"
    fi
else
    echo -e "${YELLOW}[SKIP] npm not found - frontend tests skipped${NC}"
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
