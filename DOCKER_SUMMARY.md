# Docker Configuration Summary

## 📦 What Was Created

### Core Files
1. **`docker-compose.yml`** - Orchestrates all services
2. **`server/Dockerfile`** - Builds Go server image
3. **`.env.example`** - Environment variables template
4. **`README.docker.md`** - Comprehensive Docker documentation

### Helper Scripts
1. **`docker-start.sh`** - One-command setup and launch
2. **`docker-health-check.sh`** - Verify all services are healthy
3. **`server/.dockerignore`** - Exclude unnecessary files from build

## 🎯 Quick Start (3 Steps)

```bash
# 1. Setup environment
cp .env.example .env
nano .env  # Add your GROQ_API_KEY

# 2. Start everything
./docker-start.sh

# 3. Connect your Flutter app to https://<your-ip>:8443
```

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────┐
│                 Docker Compose Network                  │
│            nexus-network (bridge)            │
│                                                         │
│  ┌──────────────────┐       ┌─────────────────────┐    │
│  │    QuickCom      │       │  Nexus   │    │
│  │   (Node.js)      │       │    (Go Server)      │    │
│  ├──────────────────┤       ├─────────────────────┤    │
│  │ - Puppeteer      │◄──────┤ - Groq LLM          │    │
│  │ - Chrome         │  WS   │ - MCP Tools         │    │
│  │ - Grocery APIs   │       │ - Liquidsoap        │    │
│  │                  │       │ - yt-dlp            │    │
│  │ Port: 10000      │       │ - SQLite Library    │    │
│  │ (mapped to 5000) │       │                     │    │
│  └──────────────────┘       │ Ports: 8443, 8080   │    │
│                             └─────────────────────┘    │
│                                                         │
└─────────────────────────────────────────────────────────┘
           │                           │
           │                           │
           ▼                           ▼
     localhost:5000             localhost:8443 (HTTPS)
     (Internal WebSocket)      localhost:8080 (Radio)
```

## 📊 Services Overview

### Nexus
- **Image**: Built from `server/Dockerfile`
- **Base**: `golang:1.24-alpine` → `alpine:latest`
- **Size**: ~200MB (multi-stage build)
- **Memory**: 256MB-1GB
- **Dependencies**:
  - Liquidsoap (audio streaming)
  - ffmpeg (audio processing)
  - yt-dlp (YouTube downloads)
  - Python 3 + uv (for MCP servers)
  - Node.js (for MCP servers)

### QuickCom
- **Image**: Built from `../QuickCom/Dockerfile`
- **Base**: `node:20`
- **Size**: ~1.5GB (includes Chrome)
- **Memory**: 512MB-2GB
- **Dependencies**:
  - Google Chrome (headless)
  - Puppeteer (browser automation)
  - Express (web framework)
  - WebSocket server

## 💾 Persistent Storage

All data persists across container restarts:

| Volume | Purpose | Typical Size |
|--------|---------|--------------|
| `library-data` | Downloaded audio files | 1-10GB |
| `library-db` | SQLite database | 10-100MB |
| `bin-data` | yt-dlp binary | ~20MB |
| `certs-data` | TLS certificates | <1MB |
| `streams-data` | Liquidsoap buffers | ~100MB |
| `quickcom-cache` | Puppeteer cache | ~200MB |

## 🔧 Configuration

### Environment Variables (.env)

```env
# Required
GROQ_API_KEY=gsk_your_actual_key_here

# Optional
LOG_LEVEL=info
```

### Config File (server/config/config.yaml)

Mount custom config:
```yaml
# docker-compose.yml (already configured)
volumes:
  - ./server/config/config.yaml:/app/config/config.yaml:ro
```

Edit `server/config/config.yaml` to customize:
- LLM provider/model
- Assistant modes
- Radio settings
- Tool filters

## 🌐 Ports Exposed

| Port | Service | Protocol | Description |
|------|---------|----------|-------------|
| 8443 | nexus | HTTPS | Main API (chat, tools, modes) |
| 8080 | nexus | HTTP | Radio stream (Liquidsoap) |
| 5000 | quickcom | WebSocket | Grocery shopping API |
| 5900 | quickcom | VNC | Debug Puppeteer (optional) |

## 🚀 Common Commands

### Daily Use
```bash
# Start services
docker-compose up -d

# Check status
docker-compose ps

# View logs
docker-compose logs -f

# Health check
./docker-health-check.sh

# Stop services
docker-compose down
```

### Development
```bash
# Rebuild after code changes
docker-compose build
docker-compose up -d

# Restart specific service
docker-compose restart nexus

# Access shell
docker-compose exec nexus sh

# View resource usage
docker stats
```

### Maintenance
```bash
# Update images
docker-compose pull
docker-compose up -d

# Backup library
docker run --rm -v nexus_library-db:/data \
  -v $(pwd):/backup alpine \
  tar czf /backup/library-backup.tar.gz /data

# Clean up unused resources
docker system prune -a

# View volume usage
docker system df -v
```

## 🔒 Security Features

1. **TLS Encryption**: Auto-generated self-signed certificates
2. **Network Isolation**: Services run in isolated Docker network
3. **Resource Limits**: Memory and CPU limits prevent DoS
4. **Read-only Mounts**: Config files mounted read-only
5. **No Privileged Mode**: Containers run unprivileged

## 🎯 Production Deployment

### Recommended Setup

1. **Reverse Proxy**: Use Nginx/Caddy with Let's Encrypt
2. **Firewall**: Only expose necessary ports
3. **Monitoring**: Add Prometheus + Grafana
4. **Backups**: Automated volume backups
5. **Updates**: Regular `docker-compose pull`

### Example Nginx Config

```nginx
server {
    listen 443 ssl http2;
    server_name assistant.example.com;

    ssl_certificate /etc/letsencrypt/live/example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/example.com/privkey.pem;

    location / {
        proxy_pass https://localhost:8443;
        proxy_ssl_verify off;
    }

    location /stream {
        proxy_pass http://localhost:8080;
        proxy_buffering off;
    }
}
```

## 🐛 Troubleshooting

### QuickCom Won't Start

**Symptoms**: Container exits immediately

**Solutions**:
1. Check memory: QuickCom needs 512MB+
2. Verify logs: `docker-compose logs quickcom`
3. Test Chrome: `docker-compose exec quickcom google-chrome-stable --version`

### Nexus Can't Connect to QuickCom

**Symptoms**: "Failed to connect to QuickCom" in logs

**Solutions**:
1. Check QuickCom health: `curl http://localhost:5000/api/health`
2. Test network: `docker-compose exec nexus ping quickcom`
3. Verify services are on same network: `docker network inspect nexus_nexus-network`

### Radio Not Working

**Symptoms**: No stream at localhost:8080

**Solutions**:
1. Check Liquidsoap logs: `docker-compose logs nexus | grep liquidsoap`
2. Verify Liquidsoap is running: `docker-compose exec nexus ps aux | grep liquidsoap`
3. Test telnet connection: `docker-compose exec nexus nc -zv localhost 1234`

### High Memory Usage

**Symptoms**: QuickCom using 2GB+ memory

**Solutions**:
1. This is normal for Puppeteer + Chrome
2. Adjust resource limits in docker-compose.yml
3. Restart periodically: `docker-compose restart quickcom`

## 📈 Performance Tuning

### For Low-Memory Servers (<2GB RAM)

```yaml
services:
  quickcom:
    environment:
      - NODE_OPTIONS=--max-old-space-size=512
    deploy:
      resources:
        limits:
          memory: 1G
```

### For High-Traffic Deployments

```yaml
services:
  nexus:
    deploy:
      replicas: 3
      resources:
        limits:
          cpus: '2.0'
          memory: 2G
```

## 🎓 Next Steps

1. **Read Full Docs**: See [README.docker.md](README.docker.md)
2. **Customize Config**: Edit `server/config/config.yaml`
3. **Add MCP Servers**: Clone into `mcp-repos/`
4. **Setup Monitoring**: Add Prometheus/Grafana
5. **Production Hardening**: Nginx, real TLS, firewall

## 📚 Additional Resources

- **Main README**: [CLAUDE.md](CLAUDE.md)
- **Docker Guide**: [README.docker.md](README.docker.md)
- **Flutter Optimizations**: [app/OPTIMIZATIONS.md](app/OPTIMIZATIONS.md)
- **Quick Start**: `./docker-start.sh`
- **Health Check**: `./docker-health-check.sh`

## ✅ Verification Checklist

After setup, verify everything works:

- [ ] Services running: `docker-compose ps`
- [ ] Nexus healthy: `curl -k https://localhost:8443/api/v1/health`
- [ ] QuickCom healthy: `curl http://localhost:5000/api/health`
- [ ] Radio stream accessible: `curl -I http://localhost:8080/stream`
- [ ] Flutter app connects successfully
- [ ] Can send chat messages
- [ ] Radio plays music
- [ ] Grocery search works (if enabled)

## 🎉 You're All Set!

Your Nexus is now running in Docker with:
- ✅ Automatic restarts
- ✅ Persistent storage
- ✅ Health monitoring
- ✅ Resource limits
- ✅ Network isolation
- ✅ Production-ready configuration

Happy chatting! 🚀
