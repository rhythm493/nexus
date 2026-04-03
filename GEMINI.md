# GEMINI.md - pocket-assistant

## Project Overview

**pocket-assistant** is a voice-controlled AI assistant that bridges a mobile phone (Flutter) and a PC (Go) over a local network (LAN). It uses **OpenAI-compatible LLM providers** (Groq, Cerebras, Ollama, LM Studio) and the **Model Context Protocol (MCP)** for extensible tool-calling.

### Key Architecture
- **Mobile App (Flutter):**
  - **Voice Input:** Offline STT via `sherpa-onnx`.
  - **Discovery:** mDNS discovery (`bonsoir`) to find the server.
  - **Security:** mTLS-secured communication with client-side cert import.
- **Server (Go):**
  - **Orchestrator:** Manages LLM interactions, mDNS advertisement, and MCP host.
  - **LLM Support:** Native support for Groq (default), Cerebras, Ollama, and LM Studio. Supports "rotating" providers for failover/load balancing.
  - **Assistant Modes:** "Sonos" (Music), "Grocery" (Price comparisons), and "Web Search" (Research).
- **Integrated Services:**
  - **Radio & Audio:** Liquidsoap-based radio engine with YouTube downloading (`yt-dlp`) and a local SQLite music library.
  - **External Integrations:** QuickCom WebSocket bridge (for groceries) and DuckDuckGo (web search).

---

## Building and Running

### Prerequisites
- **Go 1.24+**
- **Flutter 3.x**
- **OpenSSL** (for certificates)
- **Avahi/mDNS** (Linux)
- **Python 3.7+ with `uv`** (for Sonos MCP)
- **Liquidsoap** (for radio features)
- **yt-dlp** (for YouTube audio)

### Initialization (First-time Setup)
Installs dependencies, generates mTLS certificates, and installs the Sonos MCP server:
```bash
make init
```

### Server (Go)
1. **Set API Key:**
   ```bash
   export GROQ_API_KEY="your-groq-key" # Recommended
   # OR
   export CEREBRAS_API_KEY="your-cerebras-key"
   ```
2. **Run Server:**
   ```bash
   make server
   # or with hot reload/debug logs:
   make dev
   ```
3. **Key Endpoints:**
   - `POST /api/v1/chat`: SSE-based chat with tool calling.
   - `GET /api/v1/health`: Server & MCP status.
   - `GET /api/v1/radio/stream`: HTTP stream for the radio engine.
   - `GET /api/v1/qrcode`: Server discovery QR code for mobile.

### Mobile App (Flutter)
1. **Setup:** Transfer `certs/ca.crt`, `certs/client.crt`, and `certs/client.key` to the phone and import via App Settings.
2. **Run:** `make app`.

---

## Development Conventions

### Backend (`/server`)
- **LLM Providers:** Implemented in `internal/llm/`. Most use the OpenAI-compatible client.
- **MCP Host:** Manages lifecycle of child-process MCP servers (`internal/mcp/`).
- **Radio Engine:** Orchestrates `yt-dlp` and `liquidsoap` (`internal/radio/` and `internal/youtube/`).
- **Data Storage:** SQLite database in `data/library.db` for music and search indices.

### Frontend (`/app`)
- **State:** `Provider` package manages `ApiService`, `DiscoveryService`, and `VoiceService`.
- **UI:** Material 3 design with mode-based tabs.

### Discrepancy Note
*Note: While some older docs mention Gemini, the current codebase is optimized for Groq/Cerebras (OpenAI-compatible) for lower latency.*
