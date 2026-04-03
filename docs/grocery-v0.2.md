# Grocery Mode v0.2 — Implementation Complete

## Overview
L4 agentic grocery shopping assistant. Agent builds optimized virtual carts across Blinkit, Zepto, and Instamart. User can review, swap items, and approve. Cart-first UI with collapsible chat advisor.

## Status: All Layers Complete

| Layer | Status | Key Changes |
|-------|--------|-------------|
| 1. LLM Infrastructure | ✅ Done | go-openai SDK, CLIProxyAPI (Gemini 3 Flash), rotating provider with connection error fallback |
| 2. Schema Improvements | ✅ Done | brand, product_url, per_unit_price_paise, quantity_value/unit, discount_pct, rating |
| 3. Cart System | ✅ Done | In-memory per conversation, state machine, event-sourced, 7 tools, optimizer |
| 4. Tools Cleanup | ✅ Done | Removed old grocery_search/compare_prices/location_set from grocery mode |
| 5. Flutter UI | ✅ Done | Cart-first layout, inline search, rich product cards, agent chips, draggable divider |

## Architecture
```
Flutter App ←SSE→ Nexus (Go) ←HTTP→ QuickCom (Node.js)
                    │                      │
              Cart Manager           Provider Registry
              go-openai SDK          Blinkit/Zepto/Instamart
              Gemini (CLIProxyAPI)   SQLite Cache + Scheduler
              Cart REST API          74 Darkstores (Pune)
```

## Key Files

### Go Server (`server/internal/`)
- `cart/manager.go` — Cart, CartItem, CartProduct, state machine, event log
- `cart/optimizer.go` — Single vs split provider optimization
- `cart/tools.go` — 7 cart tools (cart_add, cart_remove, cart_view, cart_optimize, cart_clear, price_check, find_deals)
- `api/cart_handlers.go` — REST: GET full cart, POST add/swap, DELETE item, GET search proxy
- `api/model_handler.go` — Speech model download endpoint
- `llm/adapter.go` — go-openai SDK type conversion
- `llm/openai.go` — SDK-based provider implementation
- `quickcom/client.go` — HTTP client (replaced WebSocket)

### QuickCom (`QuickCom/backend/src/`)
- `providers/blinkit/provider.ts` — page.evaluate(fetch) approach
- `providers/zepto/provider.ts` — Direct HTTP API
- `providers/instamart/provider.ts` — SPA navigation capture
- `cache/scheduler.ts` — Spread snapshots, pre-warm, cleanup
- `cache/config.ts` — 6-48h TTLs, 48 popular queries

### Flutter (`app/lib/`)
- `screens/chat_screen.dart` — Mode-based body swap (grocery vs chat)
- `widgets/cart_product_card.dart` — Rich product cards with provider selection
- `widgets/inline_search.dart` — Direct QuickCom search bar
- `widgets/agent_chip.dart` — Minimal LLM advice chips
- `services/cart_service.dart` — CartFullState, REST calls, SSE updates
- `models/cart_full_state.dart` — Full cart with product details

## Configuration

### CLIProxyAPI (`~/.cli-proxy-api/config.yaml`)
```yaml
host: "127.0.0.1"
port: 24080
auth-dir: "~/.cli-proxy-api"
api-keys: ["nexus-local"]
gemini-api-key:
  - api-key: "YOUR_GEMINI_KEY"
```

### QuickCom (`.env`)
```env
PORT=5000
DEFAULT_LAT=18.5204
DEFAULT_LON=73.8567
DEFAULT_LOCATION=Kothrud, Pune
CHROME_PATH=/usr/bin/google-chrome-stable
```

## Running
```bash
# 1. CLIProxyAPI (Gemini proxy)
cli-proxy-api -config ~/.cli-proxy-api/config.yaml &

# 2. QuickCom (grocery backend)
cd QuickCom/backend && PORT=5000 node dist/src/index.js &

# 3. Nexus (Go server)
cd server && go run ./cmd/server &

# 4. Flutter app
cd app && flutter run
```
