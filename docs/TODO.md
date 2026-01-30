# pocket-assistant - Implementation Status & TODO

## Implementation Status

### Phase 1: Project Setup ✅ COMPLETE
- [x] Monorepo structure created
- [x] Go module initialized (`server/go.mod`)
- [x] Flutter project structure (`app/pubspec.yaml`, `app/lib/`)
- [x] CLAUDE.md with AI instructions
- [x] Makefile with common commands
- [x] .gitignore configured

### Phase 2: Go Server Foundation ✅ COMPLETE
- [x] Config loading (`server/config/config.go`) - YAML + env vars
- [x] Basic HTTP server with health endpoint (`server/internal/api/handler.go`)
- [x] mTLS certificate generation script (`scripts/gen-certs.sh`)
- [x] mTLS server setup (`server/internal/tls/certs.go`)
- [x] mDNS service advertisement (`server/internal/mdns/advertise.go`)

### Phase 3: Gemini Integration ✅ COMPLETE
- [x] Gemini API client (`server/internal/gemini/client.go`)
- [x] Tool/function definition system
- [x] Chat completion with tool calls
- [x] Tool result handling loop

### Phase 4: MCP Host ✅ COMPLETE
- [x] MCP protocol types (`server/internal/mcp/protocol.go`)
- [x] stdio transport (`server/internal/mcp/transport.go`)
- [x] Server lifecycle management (`server/internal/mcp/host.go`)
- [x] Tool discovery from MCP servers
- [x] Tool execution via MCP
- [x] Sonos MCP config (`server/mcp-servers/sonos.json`)

### Phase 5: Flutter App ✅ COMPLETE (Code written, needs Flutter SDK to build)
- [x] Basic chat UI (`app/lib/screens/chat_screen.dart`)
- [x] mDNS discovery service (`app/lib/services/discovery_service.dart`)
- [x] mTLS client setup (`app/lib/services/cert_service.dart`)
- [x] API service with SSE (`app/lib/services/api_service.dart`)
- [x] Voice input service (`app/lib/services/voice_service.dart`)
- [x] Connection status indicator
- [x] Android manifest with permissions

### Phase 6: Integration & Polish ⏳ NOT STARTED
- [ ] End-to-end testing
- [ ] Error handling improvements
- [ ] Reconnection logic
- [ ] Loading states refinement
- [ ] iOS project setup

---

## Known Issues & Limitations

### Server
1. **No graceful MCP restart** - If an MCP server crashes, it's not restarted automatically
2. **Conversation storage is in-memory** - Lost on server restart
3. **No rate limiting** - Relies on Gemini API limits
4. **Single client assumption** - No multi-client conversation isolation

### Flutter App
1. **Not tested** - Flutter SDK not installed on dev machine
2. **iOS project missing** - Only Android skeleton created
3. **No offline mode** - Requires server connection
4. **No TTS** - Responses are text-only (no speech output)

### Security
1. **Self-signed certs** - Browsers/curl need `-k` flag
2. **No cert expiry handling** - 10-year validity, no rotation
3. **No PIN/password** - Anyone with certs can connect

---

## Next Steps (Priority Order)

### Immediate (Before First Test)
1. **Install Flutter SDK** and run `flutter pub get` in `app/`
2. **Generate certificates**: `make certs`
3. **Get Gemini API key** from https://makersuite.google.com/app/apikey
4. **Test server standalone**: `GEMINI_API_KEY=xxx make dev`
5. **Test with curl**: `make check-health`

### Short Term
1. **iOS project setup** - Run `flutter create` or manually add `app/ios/`
2. **Error handling** - Add retry logic for failed API calls
3. **Reconnection** - Auto-reconnect when server comes back
4. **Loading states** - Better UX during tool execution
5. **Conversation persistence** - Save to SQLite or file

### Medium Term
1. **Text-to-speech** - Add TTS for assistant responses
2. **Wake word** - Hands-free "Hey Assistant" activation
3. **More MCP servers** - Home Assistant, Spotify, etc.
4. **Settings screen** - Configure voice, theme, etc.
5. **Message history** - Persist and load previous conversations

### Long Term
1. **Offline LLM fallback** - Local model when no internet
2. **WebSocket** - Replace SSE for bidirectional streaming
3. **Multi-device** - Sync across phone/tablet
4. **Plugins** - User-installable MCP servers

---

## Testing Checklist

### Server Tests
```bash
# 1. Generate certs (one time)
make certs

# 2. Start server
GEMINI_API_KEY=your-key make dev

# 3. Health check
curl -k --cert certs/client.crt --key certs/client.key \
  https://localhost:8443/api/v1/health
# Expected: {"status":"ok","mcp_servers":["sonos"]}

# 4. List tools
curl -k --cert certs/client.crt --key certs/client.key \
  https://localhost:8443/api/v1/tools
# Expected: {"tools":[...]}

# 5. Chat (SSE)
curl -k --cert certs/client.crt --key certs/client.key \
  -X POST -H "Content-Type: application/json" \
  -d '{"message":"Hello"}' \
  https://localhost:8443/api/v1/chat
# Expected: SSE stream with data: lines

# 6. Check mDNS
avahi-browse -a | grep pocket-assistant
# Expected: Shows service advertisement
```

### Flutter Tests (requires Flutter SDK)
```bash
cd app
flutter pub get
flutter analyze  # Check for errors
flutter test     # Run unit tests (none yet)
flutter run      # Run on device/emulator
```

---

## Code Patterns & Conventions

### Go Server
- **Error handling**: Return errors up, log at top level
- **Logging**: Use `slog` package (structured logging)
- **Config**: Environment vars override YAML config
- **HTTP**: Standard library `net/http` with Go 1.22+ routing
- **Context**: Pass `context.Context` for cancellation

### Flutter App
- **State management**: Provider pattern
- **Services**: Singleton via Provider, extend `ChangeNotifier`
- **UI**: Material 3 with dark theme
- **Naming**: snake_case for files, PascalCase for classes

### MCP
- **Protocol**: JSON-RPC 2.0 over stdio
- **Config**: One JSON file per server in `mcp-servers/`
- **Tools**: Auto-discovered via `tools/list` method

---

## File Quick Reference

| File | Purpose | When to Edit |
|------|---------|--------------|
| `server/cmd/server/main.go` | Server entry point | Add new components |
| `server/internal/api/handler.go` | HTTP endpoints | Add/modify API |
| `server/internal/gemini/client.go` | LLM integration | Change prompts, model |
| `server/internal/mcp/host.go` | MCP management | Fix MCP issues |
| `app/lib/main.dart` | App entry point | Add providers |
| `app/lib/screens/chat_screen.dart` | Main UI | Modify chat interface |
| `app/lib/services/api_service.dart` | Server communication | Fix connection issues |
| `server/mcp-servers/*.json` | MCP configs | Add new MCP servers |
