# pocket-assistant Architecture

## Overview

pocket-assistant is a voice-controlled AI assistant that connects a mobile phone to a PC over the local network. It uses Gemini as the AI backend and MCP (Model Context Protocol) for tool integration.

## System Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                     PHONE (Flutter)                          │
│  ┌────────────────────────────────────────────────────────┐  │
│  │  Voice Input (STT) ──▶ Chat UI ◀── Response Display    │  │
│  │                           │                            │  │
│  │                    HTTP/SSE over mTLS                  │  │
│  └───────────────────────────┼────────────────────────────┘  │
└──────────────────────────────┼───────────────────────────────┘
                               │ mDNS discovery + mTLS
                               ▼
┌──────────────────────────────────────────────────────────────┐
│                      PC (Go Server)                          │
│  ┌────────────────────────────────────────────────────────┐  │
│  │                    Orchestrator                        │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │  │
│  │  │ Gemini API   │  │  MCP Host    │  │   mDNS +     │  │  │
│  │  │ Client       │  │  (spawns     │  │   mTLS       │  │  │
│  │  │              │  │   servers)   │  │   Server     │  │  │
│  │  └──────────────┘  └──────────────┘  └──────────────┘  │  │
│  │                           │                            │  │
│  │              ┌────────────┴────────────┐               │  │
│  │              ▼                         ▼               │  │
│  │     ┌──────────────┐          ┌──────────────┐         │  │
│  │     │  Sonos MCP   │          │  Future MCP  │         │  │
│  │     │  (stdio)     │          │  Servers...  │         │  │
│  │     └──────────────┘          └──────────────┘         │  │
│  └────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────┘
```

## Component Details

### Mobile App (Flutter)

**Responsibilities:**
- Voice input via platform speech-to-text
- Chat UI for displaying conversations
- mDNS discovery to find server on LAN
- mTLS client authentication
- SSE parsing for streaming responses

**Key Services:**
| Service | Purpose |
|---------|---------|
| `DiscoveryService` | mDNS-based server discovery |
| `CertService` | Manages mTLS certificates |
| `ApiService` | HTTP/SSE communication |
| `VoiceService` | Speech-to-text input |

**Data Flow:**
1. User speaks → Platform STT → Text
2. Text → HTTP POST to server
3. SSE events → UI updates
4. Tool calls displayed inline

### Go Server

**Responsibilities:**
- Accept authenticated HTTPS connections
- Advertise service via mDNS
- Route requests to Gemini API
- Manage MCP server processes
- Execute tools via MCP protocol

**Key Packages:**
| Package | Purpose |
|---------|---------|
| `internal/api` | HTTP handlers, SSE streaming |
| `internal/gemini` | Gemini API client with function calling |
| `internal/mcp` | MCP host, protocol, stdio transport |
| `internal/mdns` | mDNS service advertisement |
| `internal/tls` | mTLS configuration |

### MCP Integration

The server acts as an MCP host, spawning MCP servers as child processes and communicating via stdio.

**MCP Protocol (JSON-RPC 2.0):**
```
Server ──stdin──▶ MCP Process
Server ◀─stdout── MCP Process
```

**Lifecycle:**
1. Server reads `mcp-servers/*.json` configs
2. Spawns each MCP server process
3. Sends `initialize` handshake
4. Queries `tools/list` for available tools
5. Keeps process running for tool calls
6. Terminates on server shutdown

**Tool Execution:**
1. Gemini requests tool call
2. Server finds MCP server with that tool
3. Sends `tools/call` via stdio
4. Returns result to Gemini

### Security Model

**mTLS (Mutual TLS):**
- Server has certificate signed by CA
- Client has certificate signed by same CA
- Both verify each other
- No anonymous connections possible

**Certificate Hierarchy:**
```
CA (self-signed)
├── Server Certificate
│   └── Signs: pocket-assistant-server
│   └── SANs: localhost, *.local, local IPs
└── Client Certificate
    └── Signs: pocket-assistant-client
```

## Data Flow

### Chat Message Flow

```
1. Phone: User speaks "play jazz"
   └── STT converts to text

2. Phone: POST /api/v1/chat
   └── mTLS authenticates client
   └── Body: {"message": "play jazz"}

3. Server: Gemini API call
   └── System prompt + user message + tools
   └── Gemini decides to call "play" tool

4. Server: MCP tool execution
   └── Find Sonos MCP server
   └── Send: {"method": "tools/call", "params": {...}}
   └── Receive: {"result": {"status": "playing"}}

5. Server: Continue with Gemini
   └── Send tool result back to Gemini
   └── Gemini generates final response

6. Server → Phone: SSE stream
   └── data: {"type": "text", "content": "Playing jazz..."}
   └── data: {"type": "tool_call", "name": "play", ...}
   └── data: {"type": "tool_result", ...}
   └── data: {"type": "done"}

7. Phone: Update UI
   └── Show message bubbles
   └── Show tool indicators
```

### Server Discovery Flow

```
1. Phone: Start Bonsoir discovery
   └── Listen for _pocket-assistant._tcp.local

2. PC: mDNS advertisement
   └── Advertise hostname:8443

3. Phone: Resolve service
   └── Get IP address and port

4. Phone: Health check
   └── GET /api/v1/health with mTLS

5. Phone: Connected
   └── Save server for future use
```

## Configuration

### Server Config (`config/config.yaml`)

```yaml
port: 8443
service_name: pocket-assistant
certs_dir: ../certs
mcp_servers_dir: ./mcp-servers
log_level: info
```

### MCP Server Config (`mcp-servers/sonos.json`)

```json
{
  "name": "sonos",
  "command": "uvx",
  "args": ["sonos-mcp-server"],
  "env": {}
}
```

## Extensibility

### Adding New MCP Servers

1. Create config file in `mcp-servers/`
2. Server auto-discovers on startup
3. Tools automatically available to Gemini

### Adding API Endpoints

1. Add handler method in `internal/api/handler.go`
2. Register route in `Start()` method
3. Update client if needed

### Custom Voice Commands

The system uses Gemini's natural language understanding. No explicit command definitions needed - just describe capabilities in the system prompt.

## Performance Considerations

- **SSE vs WebSocket**: SSE is simpler for server→client streaming. WebSocket would be needed for bidirectional streaming.
- **Tool Latency**: Each MCP call adds ~100ms overhead
- **Gemini Latency**: API calls typically 1-3 seconds
- **mDNS Discovery**: Usually resolves within 1-2 seconds

## Future Enhancements

1. **WebSocket support**: For lower latency bidirectional communication
2. **Text-to-speech**: Server-side TTS for consistent voice
3. **Wake word**: Hands-free activation
4. **Multiple MCP servers**: Home Assistant, Spotify, etc.
5. **Offline mode**: Local LLM fallback
