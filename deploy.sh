#!/bin/bash
# Deploy Nexus + QuickCom to remote VM via Tailscale

set -e

VM_IP="100.67.37.67"
VM_USER="ubuntu"
DEPLOY_DIR="/opt/nexus"

echo "Deploying to VM at $VM_IP..."

# Copy docker-compose.yml and .env to VM
scp -o StrictHostKeyChecking=no -i ~/.ssh/id_ed25519 \
    docker-compose.yml \
    $VM_USER@$VM_IP:$DEPLOY_DIR/ 2>&1 || echo "Copy failed, may need to create $DEPLOY_DIR"

# SSH into VM and deploy
ssh -o StrictHostKeyChecking=no -i ~/.ssh/id_ed25519 $VM_USER@$VM_IP << 'ENDSSH'
set -e

# Create deploy directory if not exists
sudo mkdir -p /opt/nexus
sudo chown ubuntu:ubuntu /opt/nexus

# Login to GitHub Container Registry (requires GITHUB_TOKEN env var)
# echo $GITHUB_TOKEN | docker login ghcr.io -u $GITHUB_USER --password-stdin

# Pull latest images (when using ghcr.io)
# docker pull ghcr.io/rhythm493/pocket-assistant/nexus:latest

# Or build locally on VM (slower first time)
cd /opt/nexus
docker compose pull || docker compose build

# Start services
docker compose up -d

echo "Deployment complete!"
docker compose ps
ENDSSH
