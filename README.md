# pocket-assistant

Voice-controlled AI assistant that connects your phone to your PC via LAN, using Gemini API with MCP (Model Context Protocol) tool integration.

## Features

- **Voice Control**: Speak commands on your phone
- **Local Network**: Secure mTLS connection over LAN
- **Auto-Discovery**: mDNS service discovery (no IP configuration needed)
- **Extensible**: Add new capabilities via MCP servers
- **Privacy**: All processing happens locally or through your own API key

## Quick Start

### Prerequisites

- Go 1.21+
- Flutter 3.x
- OpenSSL (for certificate generation)
- Avahi (mDNS - usually pre-installed on Linux)
- Python 3.7+ with uv (for Sonos MCP)

### Setup

1. **Clone and initialize**
   ```bash
   git clone <repo-url>
   cd pocket-assistant
   make init
   ```

2. **Set your Gemini API key**
   ```bash
   export GEMINI_API_KEY="your-api-key"
   ```
   Get a free API key at https://makersuite.google.com/app/apikey

3. **Start the server**
   ```bash
   make server
   ```

4. **Transfer certificates to phone**
   - Copy `certs/ca.crt`, `certs/client.crt`, `certs/client.key` to your phone
   - Import them in the app

5. **Run the app**
   ```bash
   make app
   ```

## Architecture

```
Phone (Flutter) ──── mTLS over LAN ────▶ PC (Go Server)
                                              │
                                              ├── Gemini API
                                              │
                                              └── MCP Servers
                                                   └── Sonos
```

## Documentation

- [Setup Guide](docs/SETUP.md)
- [API Reference](docs/API.md)
- [Architecture](docs/ARCHITECTURE.md)

## Current MCP Integrations

- **Sonos**: Control Sonos speakers (play, pause, volume, etc.)

## Adding MCP Servers

Create a JSON config in `server/mcp-servers/`:

```json
{
  "name": "my-server",
  "command": "uvx",
  "args": ["my-mcp-server"]
}
```

The server auto-discovers new MCP configs on startup.

## License

MIT
