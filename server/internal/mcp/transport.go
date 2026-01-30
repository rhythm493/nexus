package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"sync"
	"sync/atomic"
)

// StdioTransport handles communication with an MCP server via stdio
type StdioTransport struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	stderr    io.ReadCloser
	scanner   *bufio.Scanner
	requestID atomic.Int32
	mu        sync.Mutex
	responses map[int]chan *Response
	closed    bool
}

// NewStdioTransport creates a new stdio transport for an MCP server
func NewStdioTransport(ctx context.Context, command string, args []string, env []string) (*StdioTransport, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	if len(env) > 0 {
		cmd.Env = append(cmd.Environ(), env...)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start MCP server: %w", err)
	}

	t := &StdioTransport{
		cmd:       cmd,
		stdin:     stdin,
		stdout:    stdout,
		stderr:    stderr,
		scanner:   bufio.NewScanner(stdout),
		responses: make(map[int]chan *Response),
	}

	// Start reading responses
	go t.readLoop()

	// Start reading stderr for debugging
	go t.readStderr()

	return t, nil
}

// readLoop continuously reads responses from stdout
func (t *StdioTransport) readLoop() {
	for t.scanner.Scan() {
		line := t.scanner.Text()
		if line == "" {
			continue
		}

		slog.Debug("MCP response received", "line", line)

		var resp Response
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			slog.Error("Failed to parse MCP response", "error", err, "line", line)
			continue
		}

		t.mu.Lock()
		if ch, ok := t.responses[resp.ID]; ok {
			ch <- &resp
			delete(t.responses, resp.ID)
		}
		t.mu.Unlock()
	}

	if err := t.scanner.Err(); err != nil && !t.closed {
		slog.Error("MCP stdout scanner error", "error", err)
	}
}

// readStderr logs stderr output from the MCP server
func (t *StdioTransport) readStderr() {
	scanner := bufio.NewScanner(t.stderr)
	for scanner.Scan() {
		slog.Debug("MCP stderr", "output", scanner.Text())
	}
}

// Send sends a request and waits for a response
func (t *StdioTransport) Send(ctx context.Context, method string, params interface{}) (*Response, error) {
	id := int(t.requestID.Add(1))
	req := NewRequest(id, method, params)

	// Create response channel
	respChan := make(chan *Response, 1)
	t.mu.Lock()
	t.responses[id] = respChan
	t.mu.Unlock()

	// Marshal and send request
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	slog.Debug("MCP request sending", "method", method, "id", id)

	t.mu.Lock()
	_, err = fmt.Fprintf(t.stdin, "%s\n", data)
	t.mu.Unlock()

	if err != nil {
		return nil, fmt.Errorf("failed to write request: %w", err)
	}

	// Wait for response
	select {
	case resp := <-respChan:
		if resp.Error != nil {
			return nil, fmt.Errorf("MCP error %d: %s", resp.Error.Code, resp.Error.Message)
		}
		return resp, nil
	case <-ctx.Done():
		t.mu.Lock()
		delete(t.responses, id)
		t.mu.Unlock()
		return nil, ctx.Err()
	}
}

// Notify sends a notification (no response expected)
func (t *StdioTransport) Notify(method string, params interface{}) error {
	// Notifications don't have an ID
	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
	}
	if params != nil {
		req["params"] = params
	}

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	slog.Debug("MCP notification sending", "method", method)

	t.mu.Lock()
	_, err = fmt.Fprintf(t.stdin, "%s\n", data)
	t.mu.Unlock()

	return err
}

// Close terminates the MCP server process
func (t *StdioTransport) Close() error {
	t.closed = true
	t.stdin.Close()
	return t.cmd.Process.Kill()
}
