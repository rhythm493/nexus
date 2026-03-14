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
	"time"
)

// StdioTransport handles communication with an MCP server via stdio
type StdioTransport struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	stderr    io.ReadCloser
	scanner   *bufio.Scanner
	requestID atomic.Int32
	writeMu   sync.Mutex // protects stdin writes
	respMu    sync.Mutex // protects responses map
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

// registerResponse creates a buffered channel for the given request ID and stores it in the map.
func (t *StdioTransport) registerResponse(id int) chan *Response {
	ch := make(chan *Response, 1)
	t.respMu.Lock()
	t.responses[id] = ch
	t.respMu.Unlock()
	return ch
}

// resolveResponse sends the response on the channel for the given ID and removes it from the map.
func (t *StdioTransport) resolveResponse(id int, resp *Response) {
	t.respMu.Lock()
	ch, ok := t.responses[id]
	if ok {
		delete(t.responses, id)
	}
	t.respMu.Unlock()

	if ok {
		ch <- resp
	}
}

// cancelResponse removes the channel for the given ID from the map without sending a value.
func (t *StdioTransport) cancelResponse(id int) {
	t.respMu.Lock()
	delete(t.responses, id)
	t.respMu.Unlock()
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

		t.resolveResponse(resp.ID, &resp)
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

	// Register response channel before sending (buffered to prevent goroutine leaks)
	respChan := t.registerResponse(id)

	// Marshal and send request
	data, err := json.Marshal(req)
	if err != nil {
		t.cancelResponse(id)
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	slog.Debug("MCP request sending", "method", method, "id", id)

	t.writeMu.Lock()
	_, err = fmt.Fprintf(t.stdin, "%s\n", data)
	t.writeMu.Unlock()

	if err != nil {
		t.cancelResponse(id)
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
		t.cancelResponse(id)
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

	t.writeMu.Lock()
	_, err = fmt.Fprintf(t.stdin, "%s\n", data)
	t.writeMu.Unlock()

	return err
}

// Close gracefully terminates the MCP server process.
// It closes stdin first to signal EOF, waits up to 3 seconds for the process
// to exit, and kills it if the timeout expires.
func (t *StdioTransport) Close() error {
	t.closed = true

	// Close stdin to signal EOF to the child process
	t.stdin.Close()

	// Wait for the process to exit gracefully
	done := make(chan error, 1)
	go func() { done <- t.cmd.Wait() }()

	select {
	case <-done:
		// Process exited cleanly
	case <-time.After(3 * time.Second):
		// Process did not exit in time, kill it
		t.cmd.Process.Kill()
		<-done // Reap the zombie process
	}

	return nil
}
