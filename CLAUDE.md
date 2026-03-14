# Nexus

Voice-controlled agentic AI assistant connecting phone to PC via LAN. Rotating LLM (Groq + Cerebras) with MCP tool integration.

## Current Status: Fully Working

| Component | Status | Notes |
|-----------|--------|-------|
| Go Server | ✅ Working | Rotating LLM: 8 slots across Groq + Cerebras (~162K TPM) |
| Flutter App | ✅ Working | Tested on Moto G52, adb over WiFi |
| MCP/Sonos | ✅ Working | 24 curated tools (trimmed from 60) via sonos-ts-mcp |
| Radio/Liquidsoap | ✅ Working | Minimal pipeline: queue → mksafe → mp3 output |
| YouTube | ✅ Working | yt-dlp download, converts to mp3 for Liquidsoap |
| Library | ✅ Working | SQLite + FTS5 for cached songs |
| Voice Input | ✅ Working | Offline via sherpa-onnx (~70MB model) |
| mDNS Discovery | ✅ Working | Service: `_nexus._tcp`, UDP fallback, filters VPN/Docker IPs |
| TLS | ✅ Working | TLS 1.3, self-signed certs |
| Agentic Features | ✅ Working | ReAct thinking, guardrails, error recovery, self-sufficient tools |

## Key Architecture Decisions
- **radio_play is self-sufficient**: One call = search + download + stream + auto-connect Sonos
- **Liquidsoap must use mp3**: opus/webm decode fails silently. Don't add normalize() or metadata.map() — breaks playback
- **Filter ALL 172.x.x.x IPs**: Docker uses various subnets, not just 172.17
- **API keys from .env only**: Never in config.yaml or code. loadDotenv() in main.go
- **Tool results capped at 2KB**: Prevents context overflow from large queue/search results

## Quick Start

### Option 1: Docker (Recommended for Production)

```bash
# 1. Setup and start all services
./docker-start.sh

# 2. Run app on phone
cd app && flutter run

# 3. In app: Settings > Manual > Enter PC IP:8443
```

See [README.docker.md](README.docker.md) for full Docker documentation.

### Option 2: Local Development

```bash
# 1. Start server (from server/ directory)
cd server
go run ./cmd/server

# 2. Run app on phone (from app/ directory)
cd app
flutter run

# 3. In app: Settings > Manual > Enter PC IP:8443
```

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                     PHONE (Flutter)                          │
│  ┌────────────────────────────────────────────────────────┐  │
│  │  Voice Input ──▶ Chat UI ◀── Response Display          │  │
│  │  (sherpa-onnx)      │                                  │  │
│  │                   HTTPS                                │  │
│  │              (self-signed TLS)                         │  │
│  └───────────────────────────┼────────────────────────────┘  │
└──────────────────────────────┼───────────────────────────────┘
                               │ Manual IP / mDNS
                               ▼
┌──────────────────────────────────────────────────────────────┐
│                      PC (Go Server)                          │
│  ┌────────────────────────────────────────────────────────┐  │
│  │                    Orchestrator                        │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │  │
│  │  │  Groq API    │  │  MCP Host    │  │   mDNS +     │  │  │
│  │  │  (LLM)       │  │  (tools)     │  │   TLS        │  │  │
│  │  └──────────────┘  └──────────────┘  └──────────────┘  │  │
│  │                           │                            │  │
│  │  ┌──────────────┐  ┌──────────────┴──────────────┐     │  │
│  │  │  Radio       │  │         MCP Servers         │     │  │
│  │  │  Engine      │  │  ┌────────┐  ┌──────────┐   │     │  │
│  │  └──────┬───────┘  │  │ Sonos  │  │ Future   │   │     │  │
│  │         │          │  │(60+tools│  │ Servers  │   │     │  │
│  │  ┌──────▼───────┐  │  └────────┘  └──────────┘   │     │  │
│  │  │  Library     │  └─────────────────────────────┘     │  │
│  │  │  (SQLite)    │                                      │  │
│  │  └──────┬───────┘                                      │  │
│  │         │                                              │  │
│  │  ┌──────▼───────┐  ┌──────────────┐                    │  │
│  │  │  YouTube     │  │  Liquidsoap  │                    │  │
│  │  │  (yt-dlp)    │  │  (crossfade) │                    │  │
│  │  └──────────────┘  └──────┬───────┘                    │  │
│  │                           │                            │  │
│  │                    ┌──────▼───────┐                    │  │
│  │                    │ HTTP :8080   │◀── Sonos tunes once│  │
│  │                    │ /stream      │    (Icy-Metadata)  │  │
│  │                    └──────────────┘                    │  │
│  └────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────┘
```

## Tech Stack

| Component | Technology | Notes |
|-----------|------------|-------|
| Mobile App | Flutter (Dart) | Android tested, iOS needs setup |
| PC Server | Go 1.24 | Standard library + minimal deps |
| LLM | Groq API | llama-4-scout-17b, 30K TPM free tier |
| MCP | stdio transport | Sonos via sonos-ts-mcp (TypeScript/Node) |
| Radio | Liquidsoap | Crossfade, normalization, Icy-Metadata |
| Library | SQLite + FTS5 | Local song cache with full-text search |
| Auth | TLS only | No client certificates required |
| Discovery | mDNS/Avahi | Falls back to manual IP entry |
| Voice | sherpa-onnx | Offline, ~70MB model download |

## Project Structure

```
nexus/
├── server/                      # Go server
│   ├── cmd/server/main.go       # Entry point
│   ├── internal/
│   │   ├── api/                 # HTTP handlers + SSE streaming
│   │   ├── binutil/             # yt-dlp binary management
│   │   ├── library/             # SQLite track storage + FTS5 search
│   │   ├── llm/                 # Groq API client (OpenAI-compatible)
│   │   ├── mcp/                 # MCP host, transport, protocol
│   │   ├── mdns/                # Service discovery
│   │   ├── radio/               # Liquidsoap engine, queue, tools
│   │   ├── tls/                 # TLS certificate handling
│   │   └── youtube/             # YouTube search + download
│   ├── config/                  # YAML config loading
│   ├── scripts/                 # radio.liq (Liquidsoap config)
│   ├── data/                    # Runtime data (gitignored)
│   │   ├── library/             # Downloaded audio files
│   │   ├── bin/                 # yt-dlp binary
│   │   └── library.db           # SQLite database
│   └── mcp-servers/             # MCP server JSON configs
│
├── app/                         # Flutter app
│   ├── lib/
│   │   ├── main.dart            # App entry, providers
│   │   ├── screens/             # ChatScreen
│   │   ├── services/            # Api, Discovery, Voice services
│   │   ├── models/              # Message, SSEEvent
│   │   └── widgets/             # ChatBubble, VoiceButton
│   └── android/                 # Android-specific config
│
├── mcp-repos/                   # Cloned MCP servers
│   └── sonos-ts-mcp/            # Sonos control (TypeScript/Node)
│
├── certs/                       # Generated TLS certs (gitignored)
└── docs/                        # Documentation
```

## Server Configuration

### Config File: `server/config/config.yaml`

```yaml
port: 8443
service_name: nexus
certs_dir: ../certs
mcp_servers_dir: ./mcp-servers
llm_api_key: "gsk_your_groq_api_key_here"
log_level: info
```

### Environment Variables (override config)

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `GROQ_API_KEY` | Yes | - | Groq API key |
| `SERVER_PORT` | No | 8443 | HTTPS server port |
| `LOG_LEVEL` | No | info | debug/info/warn/error |
| `YOUTUBE_ENABLED` | No | true | Enable YouTube integration |
| `RADIO_ENABLED` | No | true | Enable Liquidsoap radio |
| `RADIO_LIQUIDSOAP_PATH` | No | /usr/bin/liquidsoap | Path to Liquidsoap |
| `RADIO_STREAM_PORT` | No | 8080 | HTTP stream port |
| `LIBRARY_DATA_DIR` | No | ./data/library | Audio file storage |
| `LIBRARY_DATABASE_PATH` | No | ./data/library.db | SQLite database |

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/chat` | Send message, receive SSE stream |
| GET | `/api/v1/health` | Health check + MCP server list |
| GET | `/api/v1/tools` | List available MCP tools |
| GET | `/api/v1/conversations/:id` | Get conversation history |
| GET | `/api/v1/youtube/search?q=` | Search YouTube, return video info |
| POST | `/api/v1/youtube/download` | Download audio from YouTube |
| GET | `/api/v1/radio/status` | Get radio status (current track, queue) |
| POST | `/api/v1/radio/play` | Play a song immediately |
| POST | `/api/v1/radio/queue` | Add song to queue |
| POST | `/api/v1/radio/skip` | Skip to next track |
| GET | `/api/v1/radio/stream` | Redirect to Liquidsoap stream |

### SSE Response Format

```
data: {"type":"text","content":"Hello!"}
data: {"type":"tool_call","name":"get_all_device_states","args":{}}
data: {"type":"tool_result","name":"get_all_device_states","result":{...}}
data: {"type":"done"}
```

## MCP Integration

### Available Sonos Tools (60+ via sonos-ts-mcp)

Key tools include:
```
# Playback
sonos_play, sonos_pause, sonos_stop, sonos_next, sonos_previous
sonos_play_url (play HTTP streams with metadata)

# Volume
sonos_set_volume, sonos_get_volume, sonos_set_mute

# Queue
sonos_get_queue, sonos_add_to_queue, sonos_clear_queue

# Discovery
sonos_discover, sonos_list_devices

# Groups
sonos_get_zone_groups, sonos_join_group, sonos_party_mode
```

### Adding New MCP Servers

1. Clone/install the MCP server:
   ```bash
   cd mcp-repos
   git clone https://github.com/user/some-mcp-server
   cd some-mcp-server && uv sync
   ```

2. Add config to `server/mcp-servers/<name>.json`:
   ```json
   {
     "name": "my-server",
     "command": "uv",
     "args": ["run", "--directory", "/path/to/mcp-server", "mcp", "run", "server.py"],
     "env": {}
   }
   ```

3. Restart server - tools auto-discovered

### MCP Protocol Notes

- Uses JSON-RPC 2.0 over stdio
- `notifications/initialized` is a notification (no response expected)
- Tools discovered via `tools/list` after initialization

## Voice Input (sherpa-onnx)

### How It Works

1. First mic button press downloads model (~70MB) from GitHub
2. Model: `sherpa-onnx-streaming-zipformer-en-2023-06-26`
3. Runs completely offline after download
4. Real-time streaming recognition

### Model Location

```
Android: /data/data/com.example.nexus/app_flutter/sherpa-onnx-streaming-zipformer-en-2023-06-26/
```

### Files Downloaded

- `encoder-epoch-99-avg-1-chunk-16-left-128.int8.onnx` (~65MB)
- `decoder-epoch-99-avg-1-chunk-16-left-128.onnx` (~3MB)
- `joiner-epoch-99-avg-1-chunk-16-left-128.onnx` (~3MB)
- `tokens.txt` (~50KB)

## Radio Integration (Liquidsoap)

### How It Works

1. User says "play Shape of You"
2. LLM calls `radio_play("Shape of You")`
3. Radio engine searches library (cached songs)
4. If miss, searches YouTube via yt-dlp
5. Downloads audio (opus/m4a) - no transcoding needed
6. Adds track to library database
7. Pushes to Liquidsoap queue via telnet
8. Liquidsoap normalizes, crossfades, and streams
9. Sonos displays track title via Icy-Metadata

### LLM Radio Tools

| Tool | Description |
|------|-------------|
| `radio_play(query)` | Search + download + play immediately |
| `radio_queue(query)` | Add to end of queue |
| `radio_skip()` | Skip to next track (with crossfade) |
| `radio_status()` | Current track, queue, state |
| `library_search(query)` | Search cached songs |
| `youtube_search(query)` | Search YouTube (doesn't play) |

### Dual Port Setup

- **Port 8443 (HTTPS)**: API endpoints, chat, tool calls
- **Port 8080 (HTTP)**: Liquidsoap stream with Icy-Metadata

### Radio API Usage

```bash
# Check radio status
curl -k https://localhost:8443/api/v1/radio/status

# Play a song
curl -k -X POST https://localhost:8443/api/v1/radio/play \
  -H "Content-Type: application/json" \
  -d '{"query":"shape of you ed sheeran"}'

# Queue a song
curl -k -X POST https://localhost:8443/api/v1/radio/queue \
  -H "Content-Type: application/json" \
  -d '{"query":"blinding lights"}'

# Skip current track
curl -k -X POST https://localhost:8443/api/v1/radio/skip

# Stream audio (via Liquidsoap)
curl "http://localhost:8080/stream" -o test.mp3
```

### Configuration

```yaml
radio:
  enabled: true
  liquidsoap_path: /usr/bin/liquidsoap
  script_path: ./scripts/radio.liq
  telnet_host: 127.0.0.1
  telnet_port: 1234
  stream_port: 8080
  crossfade_secs: 3.0
  auto_start: true

library:
  data_dir: ./data/library
  database_path: ./data/library.db
  max_size_mb: 10240

youtube:
  enabled: true
  bin_dir: ./data/bin       # yt-dlp auto-downloaded here
  output_dir: ./data/library
```

### Liquidsoap Features

- **Request queue**: Dynamic track pushing via telnet
- **Normalization**: -14 LUFS loudness normalization
- **Crossfade**: 3-second smooth transitions
- **Icy-Metadata**: "Artist - Title" shown on Sonos/players

### Dependencies

- **liquidsoap**: Audio streaming server (system package)
- **yt-dlp**: Auto-downloaded to `data/bin/` on first use
- **ffmpeg**: Used by yt-dlp for audio extraction

## Groq API Details

### Current Model

```go
model: "meta-llama/llama-4-scout-17b-16e-instruct"  // 30K TPM
```

### Rate Limits (Free Tier)

| Model | TPM | RPM |
|-------|-----|-----|
| llama-4-scout-17b | 30K | 30 |
| llama-3.3-70b-versatile | 12K | 30 |
| llama-3.1-8b-instant | 6K | 30 |

### Potential Enhancement: Model Rotation

Could implement fallback to other models on 429 errors for ~48K combined TPM.

## Common Issues & Fixes

### "Speech recognition not available"
- This message appears if Google STT fails, but sherpa-onnx should work
- Long-press mic button, wait for model download on first use

### mDNS not working on Android
- Android 13+ has mDNS restrictions
- Use manual connection: Settings > Manual > Enter IP:8443
- Generate QR code: `qrencode -t UTF8 "https://192.168.0.129:8443"`

### mDNS showing wrong IP (Tailscale/Docker)
- Fixed: Server now filters out non-LAN IPs (Tailscale 100.x.x.x, Docker 172.17.x.x)
- Only 192.168.x.x and 10.x.x.x IPs are advertised

### mDNS service name too long
- mDNS service types limited to 15 characters
- Service name changed from `pocket-assistant` to `nexus`

### MCP tools not loading
- Fixed: `notifications/initialized` must be sent as notification (no ID)
- Check server logs for "Discovered MCP tools" message

### Rate limit errors
- Switched from llama-3.3-70b (12K TPM) to llama-4-scout (30K TPM)
- Wait a few seconds between rapid messages if needed

## Development Commands

```bash
# Server
cd server && go run ./cmd/server           # Run server
cd server && go build ./...                 # Check compilation
cd server && LOG_LEVEL=debug go run ./cmd/server  # Debug mode

# App
cd app && flutter pub get                   # Get dependencies
cd app && flutter run                       # Run on connected device
cd app && flutter build apk --debug         # Build APK

# Testing
curl -k https://localhost:8443/api/v1/health
curl -k https://localhost:8443/api/v1/tools
curl -k -X POST https://localhost:8443/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{"message":"hello","conversation_id":"test"}'

# MCP Testing
cd mcp-repos/sonos-mcp-server
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | uv run mcp run server.py
```

## File Reference

| Task | Primary File |
|------|--------------|
| Change LLM model | `server/internal/llm/client.go:100` |
| Add API endpoint | `server/internal/api/handler.go` |
| Fix MCP issues | `server/internal/mcp/host.go`, `transport.go` |
| Add MCP server | `server/mcp-servers/<name>.json` |
| Modify chat UI | `app/lib/screens/chat_screen.dart` |
| Fix voice input | `app/lib/services/voice_service.dart` |
| Change TLS config | `server/internal/tls/certs.go` |
| Radio engine | `server/internal/radio/engine.go` |
| Radio tools | `server/internal/radio/tools.go` |
| Liquidsoap config | `server/scripts/radio.liq` |
| Library database | `server/internal/library/db.go` |
| YouTube download | `server/internal/youtube/downloader.go` |
| yt-dlp management | `server/internal/binutil/ytdlp.go` |
| mDNS advertisement | `server/internal/mdns/advertise.go` |
| App discovery | `app/lib/services/discovery_service.dart` |
| Sonos control | `mcp-repos/sonos-ts-mcp/src/mcp/handlers/` |

## Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| LLM Provider | Groq | Fast, free tier, tool support |
| LLM Model | llama-4-scout | Highest free tier TPM (30K) |
| Streaming | SSE | Simpler than WebSocket for one-way |
| Voice | sherpa-onnx | Offline, no Google dependency |
| Auth | TLS only | Simplified from mTLS for LAN use |
| MCP Transport | stdio | Standard, works with all MCP servers |
| Audio Streaming | Liquidsoap | Crossfade, normalization, Icy-Metadata |
| Song Cache | SQLite + FTS5 | Fast local search, no external deps |
| Audio Format | opus/m4a | Native YouTube format, no transcode |

## Dependencies

### Server (Go)
- `gopkg.in/yaml.v3` - Config parsing
- `github.com/google/uuid` - UUID generation
- `github.com/hashicorp/mdns` - mDNS advertisement
- `modernc.org/sqlite` - Pure Go SQLite driver (no CGO)

### App (Flutter)
- `sherpa_onnx: ^1.12.23` - Offline speech recognition
- `record: ^5.2.0` - Audio recording
- `bonsoir: ^5.1.4` - mDNS discovery
- `provider: ^6.1.1` - State management
- `dio: ^5.4.0` - HTTP client

## Docker Deployment

Full Docker support with docker-compose orchestration:

### Services
- **nexus**: Go server with Liquidsoap, yt-dlp, MCP
- **quickcom**: Node.js server with Puppeteer for grocery shopping
- **mdns-reflector**: Optional mDNS service discovery (requires host networking)

### Features
- Multi-stage builds for optimized images
- Persistent volumes for library, database, and certs
- Health checks and auto-restart
- Resource limits (nexus: 256MB-1GB, quickcom: 512MB-2GB)
- Production-ready with Nginx reverse proxy support

### Files
- `docker-compose.yml` - Service orchestration
- `server/Dockerfile` - Go server image
- `../QuickCom/Dockerfile` - Node.js server image (already existed)
- `.env.example` - Environment variables template
- `docker-start.sh` - Quick start script
- `README.docker.md` - Comprehensive Docker guide

### Quick Commands
```bash
# Start everything
./docker-start.sh

# View logs
docker-compose logs -f

# Stop services
docker-compose down

# Rebuild after changes
docker-compose build && docker-compose up -d
```

## Changelog

### 2026-01-31 (Radio Integration)
- Replaced blocking YouTube transcoding with always-on Liquidsoap radio
  - Instant playback with crossfading and Icy-Metadata
  - Sonos displays proper "Artist - Title" metadata
- Added SQLite library with FTS5 for cached song search
- New LLM tools: `radio_play`, `radio_queue`, `radio_skip`, `radio_status`, `library_search`
- yt-dlp auto-downloads to `data/bin/` (persistent across reboots)
- No more ffmpeg transcoding - Liquidsoap handles encoding
- New packages: `internal/radio/`, `internal/library/`, `internal/binutil/`

### 2026-01-31 (Bug Fixes)
- **Liquidsoap 2.3 compatibility**:
  - Changed `icy_metadata=true` to `metaint=8192` (new syntax)
  - Added `fallible=true` for empty queue handling
  - Changed `fallback()` to `mksafe(queue)` for reliable playback
  - Fixed telnet commands: `queue.*` → `request_queue.*`
- **Telnet auto-reconnect**: Connection now auto-reconnects if dropped
- **mDNS IP filtering**: Filters out Tailscale (100.x.x.x) and Docker (172.17.x.x) IPs
  - Only advertises LAN IPs (192.168.x.x, 10.x.x.x)
- **Service name fix**: Changed `pocket-assistant` (16 chars) to `nexus` (5 chars)
  - mDNS service types limited to 15 characters
  - Updated both server config and Flutter app discovery

### 2026-01-31 (Earlier)
- YouTube to Sonos integration (replaced by radio system above)
- Switched to sonos-ts-mcp (TypeScript, 60+ tools)

### 2026-01-30
- Switched from Gemini to Groq API (rate limits)
- Changed model to llama-4-scout (30K TPM vs 12K)
- Fixed MCP notification bug (tools now discovered)
- Removed mTLS requirement (simpler connection)
- Added sherpa-onnx for offline voice recognition
- Fixed Flutter package compatibility issues
