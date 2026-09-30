# AGENTS.md — Guide for AI Coding Agents

This is a voice-controlled AI assistant ("Nexus") with a **Go 1.24 server** and **Flutter/Dart mobile app**.
See `CLAUDE.md` for full architecture, design decisions, and component status.

## Build / Lint / Test Commands

All commands run from the repo root. Use `make` targets or run manually:

```bash
# --- Go server ---
make server                         # Run server
make dev                            # Run with LOG_LEVEL=debug
make build-server                   # Build binary to bin/
cd server && go build ./...         # Check compilation
cd server && go vet ./...           # Lint (only linter used)
cd server && go test ./...          # Run all tests (none exist yet)
cd server && go test ./... -run TestFoo -v   # Run single test by name
cd server && go fmt ./...           # Format code
cd server && go mod tidy            # Clean up go.mod/go.sum

# --- Flutter app ---
make app                            # Run on connected device
make build-android                  # Build release APK
cd app && flutter analyze           # Lint (uses analysis_options.yaml)
cd app && flutter test              # Run all widget tests (none exist yet)
cd app && flutter test test/foo_test.dart   # Run single test file
cd app && dart format lib/          # Format code
cd app && flutter pub get           # Install dependencies

# --- All ---
make lint                           # Go vet + Flutter analyze
make fmt                            # Go fmt + Dart format
make test                           # Go test + Flutter test
make deps                           # Install all dependencies
make clean                          # Clean all build artifacts

# --- Docker (trading-eu, production) ---
docker compose -f stacks/nexus-trading-eu.yml up -d         # Start Nexus
docker compose -f stacks/nexus-trading-eu.yml logs -f       # View Nexus logs

# --- Docker (legacy Traefik stacks, superseded) ---
docker compose -f stacks/traefik-stack.yml up -d              # Start Traefik
docker compose -f stacks/nexus-stack-traefik.yml up -d        # Start Nexus
docker compose -f stacks/traefik-stack.yml logs -f            # View Traefik logs
docker compose -f stacks/nexus-stack-traefik.yml logs -f      # View Nexus logs
```

## Deployment: trading-eu (production)

Nexus runs on `trading-eu` (Contabo `vmi3461756`, France) — **not** on the OCI VMs.
No Portainer: plain `docker compose`.

```bash
ssh trading-eu
cd /opt/nexus && sudo docker compose up -d
sudo docker exec nexus wget -qO- http://127.0.0.1:8443/api/v1/health
```

- Compose lives at `/opt/nexus/docker-compose.yml`, generated from `stacks/nexus-trading-eu.yml`.
- Secrets come from `/opt/nexus/.env` (mode `600`, root-owned) — **never** from the repo's
  `.env`, which is gitignored. The compose file uses `${VAR:?msg}` so a missing key fails at
  `up` time rather than silently degrading the `rotating` provider.
- **No published ports.** Nexus listens on 8443 HTTP-only and is reachable only via the
  mailcow Docker network at `172.22.1.106`, behind mailcow's nginx on
  `pocket-assistant-nexus.duckdns.org` (shared LE cert; config at
  `/opt/mailcow-dockerized/data/conf/nginx/nexus.conf`).
- **QuickCom is a sibling service** on the same host, reached over the `nexus_net` bridge
  at `QUICKCOM_URL=http://ts-quickcom:10000`. See `DEPLOYMENT_LOCAL.md` for the full
  cutover checklist and the host layout.

## CI Pipeline (`.github/workflows/ci.yml`)

Four jobs run on push/PR to `main`:
1. **go**: `go build ./...` → `go vet ./...` → `go test ./...` (tests allowed to fail)
2. **dart-analyze**: `dart analyze --fatal-warnings`
3. **flutter-build-debug**: Debug APK build + artifact upload
4. **flutter-build-release**: Release APK on tags only

## Go Code Style

**Module:** `github.com/rhythm493/pocket-assistant/server` (Go 1.24)
**Philosophy:** Standard library + minimal deps. No CGO.

### Imports
Group with blank lines — stdlib, then third-party, then project imports:
```go
import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/rhythm493/pocket-assistant/server/config"
)
```

### Naming
- Packages: lowercase single word (`api`, `llm`, `cart`, `radio`)
- Types/Functions: PascalCase exported (`NewServer`, `ExecuteTool`), camelCase private
- Receivers: short single letter (`s *Server`, `h *Host`)
- Constants: PascalCase (`DefaultSystemPrompt`)

### Error Handling
- Always wrap with context: `fmt.Errorf("doing X: %w", err)`
- Check with `errors.Is()` / `errors.As()`
- Non-critical failures: `slog.Warn("msg", "error", err)` then continue
- Custom errors: implement `Error() string` (e.g., `RateLimitError`)

### Logging
Use `log/slog` exclusively — never `fmt.Println` or `log.Println`:
```go
slog.Info("server started", "port", port)
slog.Error("failed to load", "error", err)
slog.Debug("tool result", "name", name, "size", len(result))
```

### HTTP Handlers
- Methods on `*Server` receiver: `func (s *Server) handleXxx(w http.ResponseWriter, r *http.Request)`
- JSON: `json.NewEncoder(w).Encode(resp)` / `json.NewDecoder(r.Body).Decode(&req)`
- Errors: `http.Error(w, "message", http.StatusBadRequest)`

### Concurrency
- `sync.RWMutex` for read-heavy shared state
- `context.Context` for cancellation
- `atomic.Int32` for counters

## Dart/Flutter Code Style

**App name:** `nexus`, SDK `>=3.0.0 <4.0.0`
**Linter:** `analysis_options.yaml` enforces `prefer_const_constructors`, `prefer_const_declarations`, `avoid_print`

### Imports
Group with blank lines — `dart:`, then `package:`, then relative:
```dart
import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../models/message.dart';
```

### Naming
- Classes: PascalCase (`ApiService`, `ChatScreen`)
- Private members: leading underscore (`_isConnected`)
- Methods: camelCase (`checkHealth`)
- Files: `snake_case.dart`

### State Management
- `ChangeNotifier` + Provider pattern
- `notifyListeners()` on state changes
- `context.read<T>()` for one-time access, `Consumer<T>` for reactive UI

### Logging
- Use `debugPrint()` — never `print()` (linter enforces `avoid_print`)

### Widget Patterns
- `const` constructors with `super.key`
- `Material 3` theming: `ThemeData(useMaterial3: true)`
- Factory constructors for JSON: `factory Message.fromJson(Map<String, dynamic> json)`

## Critical Rules (from CLAUDE.md)

- **API keys from `.env` only** — never in config.yaml or code
- **Liquidsoap must use mp3** — opus/webm decode fails silently; don't add `normalize()` or `metadata.map()`
- **Tool results capped at 2KB** — truncate large outputs
- **Filter ALL 172.x.x.x IPs** — Docker uses various subnets
- **SSE events** — line-buffer in Flutter parser (large events can span TCP chunks)
- **Model metadata from OpenRouter** — `server/internal/models/cache.go` fetches on startup, caches in SQLite, refreshes every 12h. Provider model IDs mapped in `server/internal/models/enricher.go`
- **`GET /api/v1/models`** — Returns full model cache + provider mappings as JSON with `ETag` header. Used by Flutter's `flutter_cache_manager` for conditional revalidation
- **Chat SSE includes `"metadata"` event** — Emitted before `"done"` with `model`, `enriched_name`, `finish_reason`, `prompt_tokens`, `completion_tokens`, `total_tokens`, `latency_ms`, `prompt_price`, `completion_price`, `estimated_cost`
- **Flutter model cache** — Uses `flutter_cache_manager` with automatic ETag-based revalidation. `ModelCacheService` provides `lookup(provider, modelId)` and `getModelsForProvider(provider)` with the same fuzzy-matching logic as server-side enricher
- **`model_cache_service.dart`** — Downloads model cache via `DefaultCacheManager().getSingleFile(url)` on app startup/reconnect. Auto-refreshes when server data changes
- **`_AssistantBubble` has expandable metadata** — Shows "Show details" / "Hide details" button that reveals token count, latency, cost, and finish reason warnings. Truncated responses show a "Continue" retry button
- **Model info bar** — `ModelInfoBar` widget above chat input shows current model display name + context size
- **Model switch detection** — When a response uses a different model than the previous turn, a `"Switched to ..."` system divider is inserted
- **`make app` passes `--dart-define`** — reads `NEXUS_SERVER_URL` from `../.env` for dev server URL
- **QuickCom prices are paise integers** — `Product` in `server/internal/quickcom/client.go` models QuickCom's `UnifiedProduct` field names (`pricePaise`, `mrpPaise`, `perUnitPricePaise`, `quantityValue`). It does **not** model the cache's narrower `CachedProduct` (`price`/`originalPrice` as strings). If QuickCom ever returns the latter shape, Go decodes it to `0` silently and grocery results show price 0 to both the user and the LLM. Any change to `POST /api/search`'s response shape must be mirrored here and re-checked against the QuickCom repo.
- **No test files exist yet** — when adding tests, follow `*_test.go` / `*_test.dart` conventions
- **After editing Go code**, run `cd server && go build ./...` to verify compilation
- **After editing Dart code**, run `cd app && flutter analyze` to verify no errors
