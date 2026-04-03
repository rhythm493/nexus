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

# --- Docker ---
docker-compose up -d                # Start all services
docker-compose logs -f              # View logs
docker-compose down                 # Stop services
```

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
- **No test files exist yet** — when adding tests, follow `*_test.go` / `*_test.dart` conventions
- **After editing Go code**, run `cd server && go build ./...` to verify compilation
- **After editing Dart code**, run `cd app && flutter analyze` to verify no errors
