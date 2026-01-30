# pocket-assistant API Reference

All endpoints require mTLS client certificate authentication.

Base URL: `https://<server-ip>:8443`

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
| `error` | Error message |
| `done` | Stream complete |

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

## Authentication

All requests must include a valid client certificate signed by the server's CA.

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
