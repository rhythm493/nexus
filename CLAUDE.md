# pocket-assistant

Voice-controlled AI assistant connecting phone to PC via LAN, using Groq API with MCP tool integration.

## Current Status: Fully Working

| Component | Status | Notes |
|-----------|--------|-------|
| Go Server | ✅ Working | Groq API (llama-4-scout), 30K TPM |
| Flutter App | ✅ Working | Tested on Moto G52 |
| MCP/Sonos | ✅ Working | 18 tools available |
| Voice Input | ✅ Working | Offline via sherpa-onnx (~70MB model) |
| mDNS Discovery | ⚠️ Partial | Works on PC, Android needs manual IP |
| TLS | ✅ Working | Server-only TLS, no client certs needed |

## Quick Start

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
│  │              ┌────────────┴────────────┐               │  │
│  │              ▼                         ▼               │  │
│  │     ┌──────────────┐          ┌──────────────┐         │  │
│  │     │  Sonos MCP   │          │  Future MCP  │         │  │
│  │     │  (18 tools)  │          │  Servers...  │         │  │
│  │     └──────────────┘          └──────────────┘         │  │
│  └────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────┘
```

## Tech Stack

| Component | Technology | Notes |
|-----------|------------|-------|
| Mobile App | Flutter (Dart) | Android tested, iOS needs setup |
| PC Server | Go 1.24 | Standard library + minimal deps |
| LLM | Groq API | llama-4-scout-17b, 30K TPM free tier |
| MCP | stdio transport | Sonos MCP via uv/Python |
| Auth | TLS only | No client certificates required |
| Discovery | mDNS/Avahi | Falls back to manual IP entry |
| Voice | sherpa-onnx | Offline, ~70MB model download |

## Project Structure

```
pocket-assistant/
├── server/                      # Go server
│   ├── cmd/server/main.go       # Entry point
│   ├── internal/
│   │   ├── api/                 # HTTP handlers + SSE streaming
│   │   ├── llm/                 # Groq API client (OpenAI-compatible)
│   │   ├── mcp/                 # MCP host, transport, protocol
│   │   ├── mdns/                # Service discovery
│   │   └── tls/                 # TLS certificate handling
│   ├── config/                  # YAML config loading
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
│   └── sonos-mcp-server/        # Sonos control (Python/uv)
│
├── certs/                       # Generated TLS certs (gitignored)
├── scripts/                     # Utility scripts
└── docs/                        # Documentation
```

## Server Configuration

### Config File: `server/config/config.yaml`

```yaml
port: 8443
service_name: pocket-assistant
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

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/chat` | Send message, receive SSE stream |
| GET | `/api/v1/health` | Health check + MCP server list |
| GET | `/api/v1/tools` | List available MCP tools |
| GET | `/api/v1/conversations/:id` | Get conversation history |

### SSE Response Format

```
data: {"type":"text","content":"Hello!"}
data: {"type":"tool_call","name":"get_all_device_states","args":{}}
data: {"type":"tool_result","name":"get_all_device_states","result":{...}}
data: {"type":"done"}
```

## MCP Integration

### Available Sonos Tools (18 total)

```
get_all_device_states    now_playing           get_device_state
pause                    stop                  play
next                     previous              get_queue
mode                     partymode             speaker_info
get_current_track_info   volume                skip
play_index               remove_index_from_queue  get_queue_length
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
Android: /data/data/com.example.pocket_assistant/app_flutter/sherpa-onnx-streaming-zipformer-en-2023-06-26/
```

### Files Downloaded

- `encoder-epoch-99-avg-1-chunk-16-left-128.int8.onnx` (~65MB)
- `decoder-epoch-99-avg-1-chunk-16-left-128.onnx` (~3MB)
- `joiner-epoch-99-avg-1-chunk-16-left-128.onnx` (~3MB)
- `tokens.txt` (~50KB)

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

## Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| LLM Provider | Groq | Fast, free tier, tool support |
| LLM Model | llama-4-scout | Highest free tier TPM (30K) |
| Streaming | SSE | Simpler than WebSocket for one-way |
| Voice | sherpa-onnx | Offline, no Google dependency |
| Auth | TLS only | Simplified from mTLS for LAN use |
| MCP Transport | stdio | Standard, works with all MCP servers |

## Dependencies

### Server (Go)
- `gopkg.in/yaml.v3` - Config parsing
- `github.com/google/uuid` - UUID generation
- `github.com/hashicorp/mdns` - mDNS advertisement

### App (Flutter)
- `sherpa_onnx: ^1.12.23` - Offline speech recognition
- `record: ^5.2.0` - Audio recording
- `bonsoir: ^5.1.4` - mDNS discovery
- `provider: ^6.1.1` - State management
- `dio: ^5.4.0` - HTTP client

## Changelog

### 2026-01-30
- Switched from Gemini to Groq API (rate limits)
- Changed model to llama-4-scout (30K TPM vs 12K)
- Fixed MCP notification bug (tools now discovered)
- Removed mTLS requirement (simpler connection)
- Added sherpa-onnx for offline voice recognition
- Fixed Flutter package compatibility issues
