.PHONY: all server app certs install-sonos-mcp clean dev help

# Default target
all: help

# Run the Go server
server:
	cd server && go run ./cmd/server

# Run the Flutter app
app:
	cd app && flutter run

# Build the Go server
build-server:
	cd server && go build -o ../bin/pocket-assistant ./cmd/server

# Build the Flutter app for Android
build-android:
	cd app && flutter build apk --release

# Build the Flutter app for iOS
build-ios:
	cd app && flutter build ios --release

# Generate mTLS certificates
certs:
	./scripts/gen-certs.sh

# Install Sonos MCP server
install-sonos-mcp:
	./scripts/install-sonos-mcp.sh

# Install all dependencies
deps:
	cd server && go mod download
	cd app && flutter pub get

# Run server in development mode with hot reload
dev:
	cd server && LOG_LEVEL=debug go run ./cmd/server

# Run tests
test:
	cd server && go test ./...
	cd app && flutter test

# Clean build artifacts
clean:
	rm -rf bin/
	cd server && go clean
	cd app && flutter clean

# Format code
fmt:
	cd server && go fmt ./...
	cd app && dart format lib/

# Lint code
lint:
	cd server && go vet ./...
	cd app && flutter analyze

# Check mDNS advertisement
check-mdns:
	avahi-browse -a | grep pocket-assistant || echo "Service not found"

# Test server health endpoint
check-health:
	curl -k --cert certs/client.crt --key certs/client.key \
		https://localhost:8443/api/v1/health

# Initialize a new project (run once)
init: deps certs install-sonos-mcp
	@echo "Project initialized! Set GEMINI_API_KEY and run 'make dev'"

# Show help
help:
	@echo "pocket-assistant - Voice-controlled AI assistant"
	@echo ""
	@echo "Usage:"
	@echo "  make server          Run Go server"
	@echo "  make app             Run Flutter app"
	@echo "  make dev             Run server in debug mode"
	@echo "  make certs           Generate mTLS certificates"
	@echo "  make install-sonos-mcp  Install Sonos MCP server"
	@echo "  make deps            Install all dependencies"
	@echo "  make build-server    Build server binary"
	@echo "  make build-android   Build Android APK"
	@echo "  make build-ios       Build iOS app"
	@echo "  make test            Run all tests"
	@echo "  make fmt             Format code"
	@echo "  make lint            Lint code"
	@echo "  make clean           Clean build artifacts"
	@echo "  make check-mdns      Check mDNS advertisement"
	@echo "  make check-health    Test server health endpoint"
	@echo "  make init            Initialize project (first time setup)"
	@echo "  make help            Show this help"
