# Docker Deployment Guide

This guide explains how to run Nexus using Docker and Docker Compose.

## Prerequisites

- Docker Engine 20.10+
- Docker Compose 2.0+
- At least 4GB RAM available
- Groq API key (free from https://console.groq.com)

## Quick Start

### 1. Set Up Environment Variables

```bash
# Copy the example environment file
cp .env.example .env

# Edit .env and add your Groq API key
nano .env
```

Add your key:
```env
GROQ_API_KEY=gsk_your_actual_groq_api_key_here
```

### 2. Build and Start Services

```bash
# Build and start all services
docker-compose up -d

# View logs
docker-compose logs -f

# View logs for specific service
docker-compose logs -f nexus
docker-compose logs -f quickcom
```

### 3. Verify Services

Check that both services are running:

```bash
# Check status
docker-compose ps

# Test nexus API
curl -k https://localhost:8443/api/v1/health

# Test QuickCom WebSocket server
curl http://localhost:5000/api/health
```

### 4. Connect Flutter App

Update your Flutter app to connect to:
- **Server URL**: `https://<your-host-ip>:8443`
- Use manual connection (mDNS may not work across Docker networks)

## Architecture

```
┌─────────────────────────────────────────────────┐
│              Docker Network                     │
│         nexus-network                │
│                                                 │
│  ┌───────────────┐      ┌──────────────────┐   │
│  │  QuickCom     │      │ Nexus │   │
│  │  (Node.js)    │◄─────┤   (Go Server)    │   │
│  │               │  WS  │                  │   │
│  │  Port: 5000   │      │  Port: 8443/8080 │   │
│  └───────────────┘      └──────────────────┘   │
│         │                        │              │
│         │                        │              │
└─────────┼────────────────────────┼──────────────┘
          │                        │
          ▼                        ▼
    Port 5000                Port 8443 (HTTPS)
    (Internal)               Port 8080 (Radio Stream)
```

## Services Overview

### Nexus (Go Server)
- **Ports**:
  - `8443`: HTTPS API (chat, tools, modes)
  - `8080`: HTTP radio stream (Liquidsoap)
- **Components**:
  - Groq LLM integration
  - MCP tool orchestration
  - Liquidsoap radio engine
  - YouTube download (yt-dlp)
  - SQLite library
- **Memory**: 256MB-1GB

### QuickCom (Node.js Server)
- **Ports**:
  - `5000`: WebSocket API (grocery shopping)
  - `5900`: VNC (for debugging Puppeteer)
- **Components**:
  - Puppeteer browser automation
  - Blinkit/Zepto/Instamart integration
  - Chrome headless browser
- **Memory**: 512MB-2GB

## Persistent Data

All data is stored in Docker volumes:

- `library-data`: Downloaded audio files
- `library-db`: SQLite database
- `bin-data`: yt-dlp binary
- `certs-data`: TLS certificates
- `streams-data`: Liquidsoap stream buffers
- `quickcom-cache`: Puppeteer browser cache

### Backup Volumes

```bash
# Backup library database
docker run --rm -v nexus_library-db:/data -v $(pwd):/backup alpine tar czf /backup/library-backup.tar.gz /data

# Restore library database
docker run --rm -v nexus_library-db:/data -v $(pwd):/backup alpine tar xzf /backup/library-backup.tar.gz -C /
```

## Configuration

### Custom Configuration File

Mount a custom `config.yaml`:

```yaml
# docker-compose.yml (already configured)
services:
  nexus:
    volumes:
      - ./server/config/config.yaml:/app/config/config.yaml:ro
```

Edit `server/config/config.yaml` to customize modes, LLM settings, etc.

### MCP Servers

To add custom MCP servers:

```bash
# Clone MCP server into mcp-repos/
cd mcp-repos
git clone https://github.com/modelcontextprotocol/servers
cd servers/src/sonos
uv sync

# Add JSON config to server/mcp-servers/
cat > ../../server/mcp-servers/sonos.json <<EOF
{
  "name": "sonos",
  "command": "uv",
  "args": ["run", "--directory", "/app/mcp-repos/servers/src/sonos", "sonos"],
  "env": {}
}
EOF

# Rebuild
docker-compose up -d --build
```

## Management Commands

### Start/Stop

```bash
# Start all services
docker-compose up -d

# Stop all services
docker-compose down

# Stop and remove volumes (WARNING: deletes all data)
docker-compose down -v
```

### Logs and Debugging

```bash
# Follow logs for all services
docker-compose logs -f

# Follow logs for nexus only
docker-compose logs -f nexus

# View last 100 lines
docker-compose logs --tail=100 nexus

# Debug QuickCom via VNC (requires VNC viewer)
# Connect to: localhost:5900
```

### Resource Usage

```bash
# View resource usage
docker stats

# Limit resources in docker-compose.yml
services:
  nexus:
    deploy:
      resources:
        limits:
          memory: 1G
          cpus: '1.0'
```

### Rebuilding

```bash
# Rebuild after code changes
docker-compose build

# Rebuild specific service
docker-compose build nexus

# Rebuild without cache
docker-compose build --no-cache
```

### Shell Access

```bash
# Access nexus shell
docker-compose exec nexus sh

# Access quickcom shell
docker-compose exec quickcom bash

# Run one-off command
docker-compose exec nexus ls -la /app/data/library
```

## Networking

### Access from LAN

To access from other devices on your network:

1. Find your host IP:
   ```bash
   ip addr show | grep "inet "
   ```

2. Open firewall ports:
   ```bash
   sudo ufw allow 8443/tcp
   sudo ufw allow 8080/tcp
   ```

3. Connect Flutter app to: `https://<host-ip>:8443`

### mDNS Discovery

For mDNS to work:

1. Uncomment `mdns-reflector` in `docker-compose.yml`
2. Run with host networking:
   ```bash
   docker-compose up -d mdns-reflector
   ```

Note: Host networking bypasses Docker network isolation.

## Troubleshooting

### QuickCom fails to start

**Error**: Puppeteer can't launch Chrome

**Solution**:
- Check memory (QuickCom needs at least 512MB)
- Verify Chrome installation: `docker-compose exec quickcom google-chrome-stable --version`

### Nexus can't connect to QuickCom

**Error**: `Failed to connect to QuickCom`

**Solution**:
```bash
# Check QuickCom health
curl http://localhost:5000/api/health

# Check network connectivity
docker-compose exec nexus wget -O- http://quickcom:10000/api/health
```

### Radio stream not working

**Error**: Liquidsoap fails to start

**Solution**:
- Check logs: `docker-compose logs nexus | grep -i liquidsoap`
- Verify Liquidsoap is installed: `docker-compose exec nexus liquidsoap --version`
- Check script syntax: `docker-compose exec nexus liquidsoap --check /app/scripts/radio.liq`

### TLS certificate errors

**Error**: `x509: certificate signed by unknown authority`

**Solution**:
- Certificates are auto-generated on first run
- Flutter app automatically accepts self-signed certs
- For manual testing: `curl -k https://localhost:8443/api/v1/health`

### Out of disk space

**Error**: No space left on device

**Solution**:
```bash
# Check volume sizes
docker system df -v

# Clean up old images/containers
docker system prune -a

# Limit library size in config.yaml
library:
  max_size_mb: 5120  # 5GB
```

## Production Deployment

### Reverse Proxy (Nginx)

```nginx
server {
    listen 443 ssl;
    server_name nexus.example.com;

    ssl_certificate /etc/letsencrypt/live/example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/example.com/privkey.pem;

    location / {
        proxy_pass https://localhost:8443;
        proxy_ssl_verify off;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    location /stream {
        proxy_pass http://localhost:8080;
        proxy_buffering off;
    }
}
```

### Security Considerations

1. **Change default ports** in production
2. **Use real TLS certificates** (Let's Encrypt)
3. **Set strong API keys**
4. **Enable firewall** (ufw, iptables)
5. **Regular updates**: `docker-compose pull && docker-compose up -d`
6. **Monitor logs** for suspicious activity

### Performance Tuning

```yaml
# docker-compose.yml
services:
  nexus:
    deploy:
      resources:
        limits:
          cpus: '2.0'
          memory: 2G
        reservations:
          cpus: '0.5'
          memory: 512M
    restart: always
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

## Updating

```bash
# Pull latest changes
cd /home/rhythm493/Projects/Local/nexus
git pull

# Rebuild and restart
docker-compose down
docker-compose build --no-cache
docker-compose up -d

# Verify
docker-compose ps
docker-compose logs -f
```

## Uninstall

```bash
# Stop and remove containers
docker-compose down

# Remove volumes (WARNING: deletes all data)
docker-compose down -v

# Remove images
docker rmi nexus_nexus
docker rmi nexus_quickcom

# Clean up everything
docker system prune -a --volumes
```

## Support

- **Issues**: https://github.com/rhythm493/nexus/issues
- **Logs**: Always include output of `docker-compose logs` when reporting issues
- **Health Check**: `curl -k https://localhost:8443/api/v1/health`
