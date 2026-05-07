# GEMINI.md - Nexus (pocket-assistant)

## Project Overview

**Nexus** (formerly pocket-assistant) is a voice-controlled agentic AI assistant connecting a phone (Flutter) to a PC (Go) over the local network. It features a **rotating LLM architecture** (Gemini 3 Flash via CLIProxyAPI + Groq + Cerebras), MCP tool integration, and a dedicated **L4 agentic grocery shopping** system.

### Key Architecture
- **Mobile App (Flutter):**
  - **Voice Input:** Offline streaming STT via `sherpa-onnx`.
  - **Discovery:** mDNS discovery (service: `_nexus._tcp`) or manual IP entry.
  - **Interface:** Dynamic mode switching (Chat vs. Cart-first Grocery UI).
- **Server (Go):**
  - **Orchestrator:** Manages LLM tool-calling (ReAct), SSE streaming, and MCP hosts.
  - **LLM Support:** Primary LLM is **Gemini 3 Flash** (via `CLIProxyAPI` on port 24080), with rotating fallback to Groq and Cerebras for high availability.
  - **Assistant Modes:** "Sonos" (Music), "Grocery" (L4 Agentic Cart), and "Web Search".
- **Integrated Services:**
  - **Radio Engine:** Liquidsoap-based stream with `yt-dlp` downloading, crossfading, and Icy-Metadata.
  - **Grocery (QuickCom):** TypeScript-based REST API service (Puppeteer/Node) supporting Blinkit, Zepto, and Instamart.
  - **Data Storage:** SQLite with FTS5 for music library and search caching.

---

## Building and Running

### Prerequisites
- **Go 1.24+**
- **Flutter 3.x**
- **Node.js & Puppeteer** (for QuickCom)
- **Liquidsoap & yt-dlp** (for radio features)
- **Python 3.7+ with `uv`** (for Sonos MCP)

### Initialization
Installs dependencies, generates TLS certificates, and syncs MCP servers:
```bash
make init
```

### Server (Go)
1. **Set API Keys:**
   ```bash
   # Add to your .env file
   GEMINI_API_KEY="your-key"
   GROQ_API_KEY="your-key"
   ```
2. **Run Server:**
   ```bash
   make server
   # or with debug logs:
   make dev
   ```
3. **Key Endpoints:**
   - `POST /api/v1/chat`: SSE-based agentic chat.
   - `GET /api/v1/health`: Server, LLM, and MCP status.
   - `GET /api/v1/radio/stream`: HTTP audio stream (port 8080).

### Mobile App (Flutter)
1. **Security:** The app uses TLS 1.3. No client certificate import is required for standard LAN use.
2. **Discovery:** The app will auto-discover the `nexus` service. If mDNS fails, enter the PC IP manually in Settings.
3. **Run:** `make app`.

---

## Development Conventions

### Agent Backend (`/server`)
- **LLM Rotation:** Logic in `internal/llm/` handles the Gemini → Groq → Cerebras fallback.
- **Cart Manager:** In-memory, event-sourced state machine for the grocery system.
- **MCP Host:** Communicates over stdio with tools like `sonos-ts-mcp`.

### Frontend (`/app`)
- **State Management:** `Provider` pattern for services (Voice, Discovery, API).
- **Cart-First UI:** Optimized layout for Grocery mode with draggable dividers and product cards.

---

## Security & Connectivity
- **TLS Only:** Communication is secured via TLS, but mTLS requirements were simplified for easier LAN discovery.
- **IP Filtering:** mDNS advertisements automatically filter out Docker (172.x.x.x) and VPN IPs to ensure the phone finds the correct LAN address.

