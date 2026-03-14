#!/bin/bash
set -e

# Nexus Docker Quick Start Script

echo "🚀 Nexus Docker Setup"
echo "================================"
echo ""

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed. Please install Docker first."
    echo "   Visit: https://docs.docker.com/get-docker/"
    exit 1
fi

# Check if Docker Compose is installed
if ! command -v docker-compose &> /dev/null; then
    echo "❌ Docker Compose is not installed. Please install Docker Compose first."
    echo "   Visit: https://docs.docker.com/compose/install/"
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

# Verify GROQ_API_KEY is set (use set -a to auto-export, avoiding arbitrary code execution)
set -a
source .env
set +a
if [ -z "$GROQ_API_KEY" ] || [ "$GROQ_API_KEY" = "your_groq_api_key_here" ]; then
    echo "❌ GROQ_API_KEY is not set in .env file"
    echo "   Get your API key from: https://console.groq.com"
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
echo "🔨 Building Docker images..."
echo "   This may take a few minutes on first run..."
docker-compose build

echo ""
echo "🚀 Starting services..."
docker-compose up -d

echo ""
echo "⏳ Waiting for services to start..."
sleep 10

echo ""
echo "🔍 Checking service health..."

# Check nexus
if curl -k -f -s https://localhost:8443/api/v1/health > /dev/null 2>&1; then
    echo "✅ Nexus is running (https://localhost:8443)"
else
    echo "⚠️  Nexus may not be ready yet. Check logs with:"
    echo "   docker-compose logs nexus"
fi

# Check QuickCom
if curl -f -s http://localhost:5000/api/health > /dev/null 2>&1; then
    echo "✅ QuickCom is running (http://localhost:5000)"
else
    echo "⚠️  QuickCom may not be ready yet. Check logs with:"
    echo "   docker-compose logs quickcom"
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
echo "   docker-compose logs -f          # View all logs"
echo "   docker-compose ps               # Check status"
echo "   docker-compose down             # Stop services"
echo "   docker-compose restart          # Restart services"
echo ""
echo "📖 Full documentation: README.docker.md"
echo ""

# Show your IP addresses
echo "💡 Your IP addresses:"
ip addr show | grep "inet " | grep -v "127.0.0.1" | awk '{print "   " $2}' | cut -d/ -f1

echo ""
echo "Happy chatting! 🎯"
