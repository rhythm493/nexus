# Nexus

Voice-controlled AI assistant that connects your phone to your PC over LAN. Talk to it, and it controls your Sonos speakers, plays music, searches the web, and more — all through natural language.

<div align="center">

https://github.com/rhythm493/nexus/raw/main/assets/demo.mp4

</div>

## What it does

Say "play Shape of You" → Nexus searches YouTube, downloads the audio, streams it through Liquidsoap, and auto-connects your Sonos speaker. One voice command, music plays.

Say "set volume to 30" → Nexus discovers your Sonos devices and adjusts the volume.

Say "queue Blinding Lights" → Adds to the queue without interrupting playback.

## Architecture

```
Phone (Flutter)  ──HTTPS──▶  PC (Go Server)  ──MCP──▶  Sonos Speakers
     │                            │
  Voice Input               ┌────┴────┐
  (sherpa-onnx)             │  Groq   │ ◀── Rotating LLM (8 slots)
                            │Cerebras │     ~162K TPM combined
                            └────┬────┘
                                 │
                    ┌────────────┼────────────┐
                    │            │            │
               Liquidsoap    yt-dlp       SQLite
               (stream)    (download)    (library)
                    │
               HTTP :8080  ──▶  Sonos tunes in
```

## Features

- **Voice control** — Offline speech recognition via sherpa-onnx (~70MB model)
- **Agentic AI** — LLM reasons, plans, calls tools, recovers from errors autonomously
- **Rotating LLM** — 8 model slots across Groq + Cerebras. Auto-fallback on rate limits (~162K TPM)
- **60+ Sonos tools** — Play, pause, volume, groups, queue, alarms (24 curated for the LLM)
- **Radio stream** — Liquidsoap-powered with Icy-Metadata. Sonos displays track titles
- **Song library** — SQLite + FTS5 search. Downloads cached, never re-downloaded
- **ReAct pattern** — LLM explains reasoning before each action (visible in UI)
- **Collapsible UI** — Tool calls and thinking bubbles collapse to keep chat clean
- **mDNS discovery** — Phone auto-discovers server on LAN
- **TLS encrypted** — Self-signed certificates, TLS 1.3
- **Grocery shopping** — Agentic assistant for adding items to cart, comparing prices across Blinkit/Zepto/Instamart, and managing shopping lists

## Quick Start

### Prerequisites

- Go 1.24+
- Flutter 3.x
- Liquidsoap 2.3+
- Node.js (for Sonos MCP server)
- Android phone on same WiFi

### 1. Clone and configure

```bash
git clone https://github.com/rhythm493/nexus.git
cd nexus

# Create .env with your API keys
cp .env.example .env
# Edit .env — add GROQ_API_KEY and CEREBRAS_API_KEY
# For grocery mode, also set up QuickCom (see docs/grocery-v0.2.md)
```

### 2. Start the server

```bash
cd server
go run ./cmd/server
```

### 3. Run the app

```bash
cd app
flutter run
```

The app auto-discovers the server via mDNS/UDP. If not, go to Settings and enter your PC's IP.

## Modes

| Mode | Tools | Description |
|------|-------|-------------|
| **Music & Speakers** | 24 tools | Sonos control, radio playback, YouTube search |
| **Grocery Shopping** | 15+ tools | Add items to cart, compare prices across Zepto/Blinkit/Instamart, manage shopping lists, voice-controlled grocery assistant |
| **Web Search** | 1 tool | DuckDuckGo web search |

## How Music Playback Works

```
User: "play Shape of You"
  │
  ▼
radio_play("Shape of You")
  ├── Search local library (SQLite FTS5)
  ├── If not found → search YouTube → download via yt-dlp → save as mp3
  ├── Push to Liquidsoap queue
  ├── Liquidsoap streams at http://LAN_IP:8080/stream
  ├── Auto-discover Sonos devices
  └── Connect Sonos to stream via sonos_play_url
  │
  ▼
Music plays on Sonos speaker
```

One tool call. Everything handled internally.

## Agentic Features

- **Tool call chaining** — LLM calls multiple tools in sequence, using results from previous calls
- **Error recovery** — If a tool fails, LLM reads the error and tries an alternative
- **ReAct reasoning** — LLM explains its thinking before each action
- **Guardrails** — Max 10 tool calls per request, 2KB result cap, input validation
- **Rate limit rotation** — When one LLM slot hits a rate limit, automatically tries the next

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Server | Go 1.24 |
| App | Flutter (Dart) |
| LLM | Groq + Cerebras (rotating) |
| Voice | sherpa-onnx (offline) |
| Streaming | Liquidsoap |
| Database | SQLite + FTS5 |
| Speakers | Sonos via MCP |
| Discovery | mDNS + UDP broadcast |
| Security | TLS 1.3 (self-signed) |

## License

MIT
# Trigger CI
