#!/bin/bash

# Nexus Docker Health Check Script

echo "🏥 Nexus Health Check"
echo "================================="
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}❌ Docker is not running${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Docker is running${NC}"
echo ""

# Check container status
echo "📦 Container Status:"
docker-compose ps

echo ""
echo "🔍 Service Health Checks:"
echo ""

# Check Nexus
echo -n "Nexus (https://localhost:8443): "
if response=$(curl -k -s -w "\n%{http_code}" https://localhost:8443/api/v1/health 2>/dev/null); then
    http_code=$(echo "$response" | tail -n1)
    if [ "$http_code" -eq 200 ]; then
        echo -e "${GREEN}✅ Healthy${NC}"
        body=$(echo "$response" | head -n-1)
        if echo "$body" | jq '.' >/dev/null 2>&1; then
            echo "$body" | jq '.'
        else
            echo "$body"
        fi
    else
        echo -e "${RED}❌ Unhealthy (HTTP $http_code)${NC}"
    fi
else
    echo -e "${RED}❌ Not responding${NC}"
fi

echo ""

# Check QuickCom
echo -n "QuickCom (http://localhost:5000): "
if response=$(curl -s -w "\n%{http_code}" http://localhost:5000/api/health 2>/dev/null); then
    http_code=$(echo "$response" | tail -n1)
    if [ "$http_code" -eq 200 ]; then
        echo -e "${GREEN}✅ Healthy${NC}"
        body=$(echo "$response" | head -n-1)
        if echo "$body" | jq '.' >/dev/null 2>&1; then
            echo "$body" | jq '.'
        else
            echo "$body"
        fi
    else
        echo -e "${RED}❌ Unhealthy (HTTP $http_code)${NC}"
    fi
else
    echo -e "${RED}❌ Not responding${NC}"
fi

echo ""

# Check Radio Stream
echo -n "Radio Stream (http://localhost:8080/stream): "
if curl -s -I http://localhost:8080/stream 2>/dev/null | grep -q "200\|ICY 200"; then
    echo -e "${GREEN}✅ Streaming${NC}"
else
    echo -e "${YELLOW}⚠️  Not streaming (may be normal if no music playing)${NC}"
fi

echo ""
echo "💾 Volume Usage:"
docker system df -v | grep nexus | head -5

echo ""
echo "🔧 Resource Usage:"
docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}" | grep -E "NAME|nexus|quickcom"

echo ""
echo "📊 Recent Logs (last 10 lines per service):"
echo ""
echo "--- Nexus ---"
docker-compose logs --tail=10 nexus 2>/dev/null | tail -10

echo ""
echo "--- QuickCom ---"
docker-compose logs --tail=10 quickcom 2>/dev/null | tail -10

echo ""
echo "═══════════════════════════════════════════"
echo "💡 Troubleshooting Commands:"
echo "═══════════════════════════════════════════"
echo "View full logs:           docker-compose logs -f"
echo "Restart services:         docker-compose restart"
echo "Rebuild services:         docker-compose build && docker-compose up -d"
echo "Access shell:            docker-compose exec nexus sh"
echo "Check QuickCom via VNC:  Connect VNC to localhost:5900"
echo ""
