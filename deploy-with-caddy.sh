#!/bin/bash
# Deploy script with Caddy reverse proxy for automatic HTTPS

set -e

# Configuration
VM_IP="100.67.37.67"  # Tailscale IP of VM1
DOMAIN="pocket-assistant-nexus.duckdns.org"

echo "=== Deploying Nexus with Caddy HTTPS ==="
echo "Target VM: $VM_IP"
echo "Domain: $DOMAIN"
echo ""

# Check if Docker Compose is available
echo "Checking Docker Compose..."
ssh ubuntu@$VM_IP "which docker-compose || which docker compose"

# Upload the necessary files
echo "Uploading Caddyfile and docker-compose.yml..."
scp Caddyfile ubuntu@$VM_IP:/tmp/Caddyfile
scp docker-compose.yml ubuntu@$VM_IP:/tmp/docker-compose.yml

# Deploy on VM
ssh ubuntu@$VM_IP << 'ENDSSH'
set -e

cd /tmp

# Create directories if needed
mkdir -p /home/ubuntu/nexus

# Move files to proper location
mv Caddyfile /home/ubuntu/nexus/
mv docker-compose.yml /home/ubuntu/nexus/

cd /home/ubuntu/nexus

# Stop existing containers
echo "Stopping existing containers..."
docker-compose down || true

# Pull latest image
echo "Pulling latest Nexus image..."
docker pull ghcr.io/rhythm493/nexus:main

# Start with Caddy
echo "Starting Caddy + Nexus..."
docker-compose up -d

# Wait for services to start
echo "Waiting for services to start..."
sleep 10

# Check status
echo ""
echo "=== Container Status ==="
docker-compose ps

echo ""
echo "=== Testing Caddy ==="
curl -s -o /dev/null -w "HTTP Status: %{http_code}\n" https://pocket-assistant-nexus.duckdns.org/api/v1/health -k || echo "Caddy not ready yet, may need a few minutes for Let's Encrypt"

echo ""
echo "=== Logs ==="
docker-compose logs --tail=50

ENDSSH

echo ""
echo "=== Deployment Complete ==="
echo "Your Nexus server should be available at: https://$DOMAIN"
echo "Note: Let's Encrypt certificate may take a few minutes to provision"
