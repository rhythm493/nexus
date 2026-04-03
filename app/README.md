# Nexus Mobile App

The Flutter/Dart mobile application for the Nexus AI assistant.

## Features

- **Voice-controlled interface** - Talk to your AI assistant using natural language
- **Multi-mode operation** - Switch between Chat, Grocery Shopping, and other modes
- **Real-time communication** - Secure SSE connection to backend server
- **Agentic AI integration** - See LLM reasoning and tool calls in real-time
- **Rich UI components** - Product cards, cart management, search interfaces

## Modes

### Chat Mode
Standard AI assistant with access to:
- Music and speaker controls (Sonos)
- Web search capabilities
- Radio streaming controls
- General knowledge and task assistance

### Grocery Shopping Mode
Agentic grocery shopping assistant:
- Add items to cart using natural language
- Compare prices across Blinkit, Zepto, and Instamart
- Manage shopping lists and cart optimization
- Voice-controlled cart management
- Rich product cards with provider selection

## Project Structure

```
lib/
├── main.dart              # App entry point
├── screens/               # UI screens
│   ├── chat_screen.dart   # Chat interface
│   └── grocery_screen.dart # Grocery shopping interface
├── services/              # Business logic services
│   ├── api_service.dart   # Backend communication
│   ├── cart_service.dart  # Grocery cart management
│   ├── mode_service.dart  # Mode switching logic
│   ├── voice_service.dart # Speech recognition
│   ├── discovery_service.dart # mDNS server discovery
│   └── cert_service.dart  # TLS certificate handling
├── models/                # Data models
│   ├── message.dart       # Chat messages
│   ├── cart_state.dart    # Simple cart state
│   └── cart_full_state.dart # Detailed cart with product info
├── widgets/               # Reusable UI components
│   ├── chat_bubble.dart   # Message display with LLM reasoning
│   ├── voice_button.dart  # Voice input control
│   ├── cart_product_card.dart # Product display with provider info
│   ├── cart_item_tile.dart # Cart item management
│   ├── cart_panel.dart    # Sidebar cart view
│   ├── agent_chip.dart    # Minimal LLM advice display
│   ├── inline_search.dart # Direct product search bar
│   ├── draggable_divider.dart # Resizable panel separator
│   └── optimization_banner.dart # Cart optimization suggestions
└── utils/                 # Utility functions
    └── performance_config.dart # Performance optimization settings
```

## Getting Started

See the [root README](../README.md) for complete setup instructions including backend setup.

## Architecture

The app communicates with the Go backend via:
- Secure Server-Sent Events (SSE) for real-time updates
- REST API for cart operations and grocery searches
- mDNS for automatic backend discovery on local network
- TLS 1.3 encryption for all communications

## State Management

Uses Provider pattern with ChangeNotifier:
- ModeService handles switching between Chat/Grocery modes
- CartService manages grocery cart state and backend synchronization
- Individual widgets listen to relevant state changes for updates

## Development

Run `flutter pub get` to install dependencies.
Use `flutter run` to launch on connected device or emulator.
Run `flutter test` to execute tests (when available).