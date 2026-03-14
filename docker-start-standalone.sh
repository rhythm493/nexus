#!/bin/bash
set -e

# Pocket Assistant Standalone Docker Setup (without QuickCom)

echo "🚀 Pocket Assistant Standalone Setup"
echo "===================================="
echo ""
echo "⚠️  Note: Running without QuickCom (grocery shopping disabled)"
echo ""

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed. Please install Docker first."
    exit 1
fi

# Check if Docker Compose is installed
if ! command -v docker &> /dev/null || ! docker compose version &> /dev/null; then
    echo "❌ Docker Compose is not available. Please install Docker Compose."
    exit 1
fi

echo "✅ Docker and Docker Compose are installed"
echo ""

# Check if .env file exists
if [ ! -f .env ]; then
    echo "⚠️  No .env file found. Creating from .env.example..."
    cp .env.example .env
    echo ""
    echo "📝 Please edit .env and add your Groq API key:"
    echo "   GROQ_API_KEY=your_actual_key_here"
    echo ""
    read -p "Press Enter after updating .env, or Ctrl+C to exit..."
fi

# Verify GROQ_API_KEY is set
source .env
if [ -z "$GROQ_API_KEY" ] || [ "$GROQ_API_KEY" = "your_groq_api_key_here" ]; then
    echo "❌ GROQ_API_KEY is not set in .env file"
    exit 1
fi

echo "✅ GROQ_API_KEY is configured"
echo ""

# Create config.yaml if it doesn't exist
if [ ! -f server/config/config.yaml ]; then
    echo "📝 Creating config.yaml from example..."
    cp server/config/config.example.yaml server/config/config.yaml
    echo "✅ Created server/config/config.yaml"
fi

echo ""
echo "🔨 Building Docker image..."
docker compose -f docker-compose.standalone.yml build

echo ""
echo "🚀 Starting services..."
docker compose -f docker-compose.standalone.yml up -d

echo ""
echo "⏳ Waiting for services to start..."
sleep 10

echo ""
echo "🔍 Checking service health..."

# Check pocket-assistant
if curl -k -f -s https://localhost:8443/api/v1/health > /dev/null 2>&1; then
    echo "✅ Pocket Assistant is running (https://localhost:8443)"
else
    echo "⚠️  Pocket Assistant may not be ready yet. Check logs with:"
    echo "   docker compose -f docker-compose.standalone.yml logs pocket-assistant"
fi

echo ""
echo "═══════════════════════════════════════════"
echo "🎉 Setup Complete!"
echo "═══════════════════════════════════════════"
echo ""
echo "📱 Connect your Flutter app to:"
echo "   https://<your-ip>:8443"
echo ""
echo "🎵 Radio stream available at:"
echo "   http://<your-ip>:8080/stream"
echo ""
echo "📊 Useful commands:"
echo "   docker compose -f docker-compose.standalone.yml logs -f"
echo "   docker compose -f docker-compose.standalone.yml ps"
echo "   docker compose -f docker-compose.standalone.yml down"
echo ""
echo "💡 Your IP addresses:"
ip addr show | grep "inet " | grep -v "127.0.0.1" | awk '{print "   " $2}' | cut -d/ -f1

echo ""
echo "🛒 To enable grocery shopping (QuickCom):"
echo "   1. Run: ./fix-quickcom.sh"
echo "   2. Then: docker compose up -d"
echo ""
echo "Happy chatting! 🎯"
