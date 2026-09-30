# pocket-assistant API Reference

Dev mode (no TLS): `http://<server-ip>:8443`
Production: `https://<domain>` (TLS terminated by mailcow's nginx in front of Nexus;
production is `https://pocket-assistant-nexus.duckdns.org`). Nexus itself never serves
TLS and publishes no ports — it is reachable only from the local Docker network.

## Endpoints

### Health Check

Check server status and connected MCP servers.

```
GET /api/v1/health
```

**Response:**
```json
{
  "status": "ok",
  "mcp_servers": ["sonos"]
}
```

### List Tools

Get all available tools from connected MCP servers.

```
GET /api/v1/tools
```

**Response:**
```json
{
  "tools": [
    {
      "name": "get_all_device_states",
      "description": "Get the current state of all Sonos devices",
      "inputSchema": {
        "type": "object",
        "properties": {}
      }
    },
    {
      "name": "play",
      "description": "Start playback on a Sonos device",
      "inputSchema": {
        "type": "object",
        "properties": {
          "device": {
            "type": "string",
            "description": "Device name or room name"
          }
        }
      }
    }
  ]
}
```

### Chat

Send a message and receive a streaming response.

```
POST /api/v1/chat
Content-Type: application/json
```

**Request:**
```json
{
  "message": "Play some jazz music",
  "conversation_id": "optional-uuid"
}
```

**Response:** Server-Sent Events (SSE)

```
data: {"type": "text", "content": "I'll play some jazz for you."}

data: {"type": "tool_call", "name": "play", "args": {"device": "Living Room"}}

data: {"type": "tool_result", "name": "play", "result": {"status": "playing"}}

data: {"type": "text", "content": "Jazz is now playing in the Living Room."}

data: {"type": "metadata", "result": {"model":"gemini-2.5-flash","enriched_name":"Google: Gemini 2.5 Flash","finish_reason":"stop","prompt_tokens":2439,"completion_tokens":2,"total_tokens":2441,"latency_ms":1512,"prompt_price":"0.0000003","completion_price":"0.0000025","estimated_cost":"0.000001"}}

data: {"type": "done"}
```

**Response Headers:**
- `X-Conversation-ID`: The conversation ID (generated if not provided)

**Event Types:**

| Type | Description |
|------|-------------|
| `text` | Text response from the assistant |
| `tool_call` | Tool being called (includes name and args) |
| `tool_result` | Result from tool execution |
| `thinking` | LLM reasoning before a tool call (ReAct pattern) |
| `metadata` | Token usage, model info, latency, pricing — emitted before `done` |
| `error` | Error message |
| `done` | Stream complete |

**Metadata Fields:**

| Field | Description |
|-------|-------------|
| `model` | Raw model ID from the API response |
| `enriched_name` | Human-readable model name from OpenRouter (e.g. "Google: Gemini 2.5 Flash") |
| `finish_reason` | Why the model stopped: `stop`, `length`, `content_filter`, or `tool_calls` |
| `prompt_tokens` | Token count of the prompt |
| `completion_tokens` | Token count of the completion |
| `total_tokens` | Total token count |
| `latency_ms` | End-to-end latency in milliseconds |
| `prompt_price` | Price per token for prompts (from OpenRouter cache) |
| `completion_price` | Price per token for completions (from OpenRouter cache) |
| `estimated_cost` | Calculated cost: `(prompt_tokens × prompt_price + completion_tokens × completion_price) / 1000` |

**Note:** `finish_reason: "length"` means the response was truncated. The Flutter app shows a "Continue" retry button. `finish_reason: "content_filter"` means the response was blocked by the API's content filter.

### Get Conversation

Retrieve conversation history.

```
GET /api/v1/conversations/:id
```

**Response:**
```json
{
  "id": "uuid",
  "messages": [
    {
      "role": "user",
      "content": "Play some jazz",
      "timestamp": "2024-01-15T10:30:00Z"
    },
    {
      "role": "assistant",
      "content": "Jazz is now playing in the Living Room.",
      "timestamp": "2024-01-15T10:30:02Z"
    }
  ]
}
```

### List Providers

Get available LLM providers that can be selected in the app.

```
GET /api/v1/providers
```

**Response:**
```json
{
  "providers": [
    {
      "name": "gemini",
      "display_name": "Google Gemini",
      "requires_auth": true,
      "default_url": "https://generativelanguage.googleapis.com/v1beta/openai"
    },
    {
      "name": "groq",
      "display_name": "Groq",
      "requires_auth": true,
      "default_url": "https://api.groq.com/openai/v1"
    },
    {
      "name": "ollama",
      "display_name": "Ollama",
      "requires_auth": false,
      "default_url": "http://localhost:11434/v1"
    }
  ]
}
```

### List Provider Models

Get available models for a provider, enriched with metadata from OpenRouter cache.

```
GET /api/v1/providers/{name}/models
```

**Parameters:**
- `name`: Provider name (e.g. `gemini`, `groq`, `cerebras`, `ollama`, `lmstudio`, `cliproxy`)

**Response:**
```json
{
  "models": [
    {
      "id": "models/gemini-2.5-flash",
      "name": "models/gemini-2.5-flash",
      "display_name": "Google: Gemini 2.5 Flash",
      "description": "Stable version of Gemini 2.5 Flash...",
      "context_size": 1048576,
      "owned_by": "google",
      "prompt_price": "0.0000003",
      "completion_price": "0.0000025"
    }
  ]
}
```

**Note:** `display_name`, `description`, `context_size`, `prompt_price`, and `completion_price` are populated from the OpenRouter model cache (synced on server startup, refreshed every 12h). Not all models will have enrichment data.

### List All Models (Cache)

Get the full OpenRouter model cache with provider mappings. Used by Flutter's `flutter_cache_manager` for offline caching and per-chat metadata enrichment.

```
GET /api/v1/models
```

**Headers:**
- `ETag`: SHA256 hash of the response body (for conditional revalidation)
- `Cache-Control: public, max-age=3600`

**Response:**
```json
{
  "models": [
    {
      "id": "google/gemini-2.5-flash",
      "display_name": "Google: Gemini 2.5 Flash",
      "description": "Stable version of Gemini 2.5 Flash...",
      "context_length": 1048576,
      "prompt_price": "0.0000003",
      "completion_price": "0.0000025"
    }
  ],
  "mappings": {
    "gemini": {
      "models/gemini-2.5-flash": "google/gemini-2.5-flash"
    },
    "groq": {
      "llama-3.3-70b-versatile": "meta-llama/llama-3.3-70b-instruct"
    }
  },
  "updated_at": "2026-05-09T12:00:00Z"
}
```

**Notes:**
- Client can store the `ETag` and send `If-None-Match` for conditional requests
- `flutter_cache_manager` handles this automatically
- Models are refreshed from OpenRouter every 12 hours
- `mappings` are the same provider-to-OpenRouter-ID mappings used by the server-side enricher

## Authentication

**curl example:**
```bash
curl -k \
  --cert certs/client.crt \
  --key certs/client.key \
  https://localhost:8443/api/v1/health
```

**Note:** The `-k` flag is needed because we're using a self-signed CA. The mTLS still provides security.

## Error Responses

**400 Bad Request:**
```json
{
  "error": "Invalid request body"
}
```

**401 Unauthorized:**
- Missing or invalid client certificate

**404 Not Found:**
```json
{
  "error": "Conversation not found"
}
```

**500 Internal Server Error:**
```json
{
  "error": "Internal server error"
}
```

## SSE Client Example (JavaScript)

```javascript
const eventSource = new EventSource('/api/v1/chat', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ message: 'Hello' })
});

eventSource.onmessage = (event) => {
  const data = JSON.parse(event.data);

  switch (data.type) {
    case 'text':
      console.log('Assistant:', data.content);
      break;
    case 'tool_call':
      console.log('Calling tool:', data.name, data.args);
      break;
    case 'tool_result':
      console.log('Tool result:', data.result);
      break;
    case 'done':
      eventSource.close();
      break;
  }
};
```

## Rate Limits

No explicit rate limits. However, Gemini API has its own rate limits (free tier: 15 RPM, 1M tokens/day).

## WebSocket Alternative

Future versions may support WebSocket for bidirectional streaming. Currently, SSE is used for server-to-client streaming.
