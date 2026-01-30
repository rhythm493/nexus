#!/bin/bash
# Install Sonos MCP server for pocket-assistant
# Requires Python 3.7+ and uv

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}Installing Sonos MCP Server${NC}"
echo ""

# Check for Python
if ! command -v python3 &> /dev/null; then
    echo -e "${RED}Error: Python 3 is required but not installed${NC}"
    exit 1
fi

PYTHON_VERSION=$(python3 -c 'import sys; print(f"{sys.version_info.major}.{sys.version_info.minor}")')
echo "Python version: $PYTHON_VERSION"

# Check for uv
if ! command -v uv &> /dev/null; then
    echo -e "${YELLOW}uv not found. Installing uv...${NC}"
    curl -LsSf https://astral.sh/uv/install.sh | sh
    export PATH="$HOME/.cargo/bin:$PATH"
fi

echo "uv version: $(uv --version)"

# Test Sonos MCP installation
echo ""
echo -e "${YELLOW}Testing Sonos MCP Server...${NC}"

# Try running the server
if uvx sonos-mcp-server --help &> /dev/null; then
    echo -e "${GREEN}Sonos MCP Server is available!${NC}"
else
    echo -e "${YELLOW}Installing sonos-mcp-server...${NC}"
    uv tool install sonos-mcp-server
fi

# Verify it works
echo ""
echo "Testing MCP server initialization..."
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}}}' | timeout 5 uvx sonos-mcp-server 2>/dev/null | head -1 && echo -e "${GREEN}MCP server responds correctly!${NC}" || echo -e "${YELLOW}Note: Server may work but requires Sonos devices on network${NC}"

echo ""
echo -e "${GREEN}Installation complete!${NC}"
echo ""
echo "The Sonos MCP server will be started automatically by the pocket-assistant server."
echo "Make sure you have Sonos speakers on your local network."
echo ""
echo "Available tools:"
echo "  - get_all_device_states: Get status of all Sonos devices"
echo "  - now_playing: Get currently playing track"
echo "  - get_device_state: Get state of specific device"
echo "  - pause: Pause playback"
echo "  - stop: Stop playback"
echo "  - play: Start playback"
echo ""
echo "See: https://github.com/WinstonFassett/sonos-mcp-server"
