#!/bin/bash
set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${CYAN}"
echo "   ╔══════════════════════════════════════╗"
echo "   ║          🍪  COOKIT  Docker          ║"
echo "   ╚══════════════════════════════════════╝"
echo -e "${NC}"

if [ -z "$1" ]; then
    echo -e "${RED}Usage: ./run.sh /path/to/cookie/folder${NC}"
    exit 1
fi

COOKIE_PATH="$(cd "$1" && pwd)"

if [ ! -d "$COOKIE_PATH" ]; then
    echo -e "${RED}✗ Path does not exist: $COOKIE_PATH${NC}"
    exit 1
fi

if [ ! -f ".env" ]; then
    echo -e "${RED}✗ .env file not found. Create one with GEMINI_API_KEY=your_key${NC}"
    exit 1
fi

source .env

if [ -z "$GEMINI_API_KEY" ]; then
    echo -e "${RED}✗ GEMINI_API_KEY is not set in .env${NC}"
    exit 1
fi

if [[ "$OSTYPE" == "darwin"* ]]; then
    if ! command -v xquartz &> /dev/null && [ ! -d "/Applications/Utilities/XQuartz.app" ]; then
        echo -e "${YELLOW}⚠  XQuartz is required for headed browser on macOS${NC}"
        echo -e "${CYAN}   Install it: brew install --cask xquartz${NC}"
        echo -e "${CYAN}   Then restart your machine and run this script again${NC}"
        exit 1
    fi

    echo -e "${CYAN}  Setting up X11 display for macOS...${NC}"
    xhost +localhost 2>/dev/null || true
fi

echo -e "${GREEN}  ✓ Cookie path: $COOKIE_PATH${NC}"
echo -e "${GREEN}  ✓ Building Docker image...${NC}"

docker build -t cookit:latest .

echo -e "${GREEN}  ✓ Image built. Launching...${NC}"
echo ""

docker run -it --rm \
    --name cookit \
    -e GEMINI_API_KEY="$GEMINI_API_KEY" \
    -e DISPLAY=host.docker.internal:0 \
    -v "$COOKIE_PATH":/cookies:ro \
    -v "$(pwd)/data":/app/data \
    -v /tmp/.X11-unix:/tmp/.X11-unix \
    -p 8420:8420 \
    cookit:latest /cookies
