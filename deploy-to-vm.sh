#!/bin/bash
# Deploy Nexus to OCI VM via Tailscale
# Usage: ./deploy-to-vm.sh [tailscale_ip]

set -e

VM_IP="${1:-100.67.37.67}"
echo "Deploying Nexus to $VM_IP..."

# Build image locally
echo "Building Nexus Docker image..."
docker build -t nexus:latest -f server/Dockerfile server/

# Save image to tar file
echo "Saving image to tar..."
docker save nexus:latest | gzip > /tmp/nexus.tar.gz

# Copy to VM
echo "Copying image to VM..."
scp -o StrictHostKeyChecking=no -i ~/.ssh/id_ed25519 /tmp/nexus.tar.gz ubuntu@$VM_IP:/tmp/

# Load image on VM and run
echo "Loading image on VM and starting container..."
ssh -o StrictHostKeyChecking=no -i ~/.ssh/id_ed25519 ubuntu@$VM_IP << 'ENDSSH'
# Load image
zcat /tmp/nexus.tar.gz | docker load

# Stop existing container if running
docker stop nexus 2>/dev/null || true
docker rm nexus 2>/dev/null || true

# Create volumes if not exist
docker volume create nexus-library-data 2>/dev/null || true
docker volume create nexus-library-db 2>/dev/null || true
docker volume create nexus-certs-data 2>/dev/null || true
docker volume create nexus-streams-data 2>/dev/null || true

# Run Nexus container
docker run -d \
  --name nexus \
  --restart unless-stopped \
  -p 8443:8443 \
  -p 8080:8080 \
  -e LOG_LEVEL=info \
  -e PORT=8443 \
  -e RADIO_ENABLED=true \
  -e YOUTUBE_ENABLED=true \
  -e RADIO_LIQUIDSOAP_PATH=liquidsoap \
  -e RADIO_STREAM_PORT=8080 \
  -v nexus-library-data:/app/data/library \
  -v nexus-library-db:/app/data \
  -v nexus-certs-data:/app/certs \
  -v nexus-streams-data:/app/streams \
  nexus:latest

# Cleanup
rm -f /tmp/nexus.tar.gz

echo "Nexus deployed successfully!"
docker ps | grep nexus
ENDSSH

# Cleanup local
rm -f /tmp/nexus.tar.gz

echo "Deployment complete!"
echo "Nexus should be accessible at: https://$VM_IP:8443"
