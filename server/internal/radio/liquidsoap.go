// Package radio provides Liquidsoap-based radio streaming functionality.
package radio

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"
)

// Common errors
var (
	ErrNotConnected = errors.New("not connected to liquidsoap")
	ErrClosed       = errors.New("connection closed")
)

// Status contains the current Liquidsoap status.
type Status struct {
	Alive       bool
	Version     string
	Uptime      string
	QueueLength int
}

// Client provides telnet-based control of Liquidsoap.
type Client struct {
	host string
	port int
	conn net.Conn
	mu   sync.Mutex

	// Timeouts
	dialTimeout      time.Duration
	reconnectTimeout time.Duration
	commandTimeout   time.Duration

	// Reconnection guard
	reconnecting bool
}

// NewClient creates a new Liquidsoap telnet client.
func NewClient(host string, port int) *Client {
	return &Client{
		host:             host,
		port:             port,
		dialTimeout:      5 * time.Second,
		reconnectTimeout: 3 * time.Second,
		commandTimeout:   10 * time.Second,
	}
}

// Connect establishes a telnet connection to Liquidsoap.
// It will retry a few times if the initial connection fails.
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Close existing connection if any
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}

	addr := fmt.Sprintf("%s:%d", c.host, c.port)

	var lastErr error
	maxRetries := 3
	retryDelay := 500 * time.Millisecond

	for i := 0; i < maxRetries; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		dialer := net.Dialer{
			Timeout: c.dialTimeout,
		}

		conn, err := dialer.DialContext(ctx, "tcp", addr)
		if err == nil {
			c.conn = conn
			slog.Info("Connected to Liquidsoap", "addr", addr)
			return nil
		}

		lastErr = err
		slog.Debug("Connection attempt failed",
			"attempt", i+1,
			"addr", addr,
			"error", err,
		)

		if i < maxRetries-1 {
			time.Sleep(retryDelay)
			retryDelay *= 2 // Exponential backoff
		}
	}

	return fmt.Errorf("failed to connect to liquidsoap at %s: %w", addr, lastErr)
}

// Close closes the telnet connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return nil
	}

	err := c.conn.Close()
	c.conn = nil
	slog.Info("Disconnected from Liquidsoap")
	return err
}

// IsConnected returns true if connected to Liquidsoap.
func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn != nil
}

// Push adds a file to the Liquidsoap queue.
// Returns the request ID assigned by Liquidsoap.
func (c *Client) Push(filePath string) (string, error) {
	// Use request_queue.push command (Liquidsoap 2.x syntax)
	cmd := fmt.Sprintf("request_queue.push %s", filePath)

	response, err := c.command(cmd)
	if err != nil {
		return "", fmt.Errorf("push failed: %w", err)
	}

	// Response format: "<request_id>"
	requestID := strings.TrimSpace(response)
	if requestID == "" {
		return "", errors.New("push returned empty response")
	}

	slog.Debug("Pushed to queue", "file", filePath, "requestID", requestID)
	return requestID, nil
}

// Skip skips the currently playing track.
func (c *Client) Skip() error {
	_, err := c.command("request_queue.skip")
	if err != nil {
		return fmt.Errorf("skip failed: %w", err)
	}
	slog.Debug("Skipped current track")
	return nil
}

// Flush clears the entire queue.
func (c *Client) Flush() error {
	_, err := c.command("request_queue.flush_and_skip")
	if err != nil {
		return fmt.Errorf("flush failed: %w", err)
	}
	slog.Debug("Flushed queue")
	return nil
}

// Metadata returns the current track metadata.
// Returns nil map if no track is playing.
func (c *Client) Metadata() (map[string]string, error) {
	response, err := c.command("request.on_air")
	if err != nil {
		return nil, fmt.Errorf("metadata failed: %w", err)
	}

	// Parse request ID from response
	requestID := strings.TrimSpace(response)
	if requestID == "" {
		return nil, nil // No track playing
	}

	// Get metadata for the request
	metaResponse, err := c.command(fmt.Sprintf("request.metadata %s", requestID))
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata for request %s: %w", requestID, err)
	}

	return parseMetadata(metaResponse), nil
}

// Status returns the current Liquidsoap status.
func (c *Client) Status() (*Status, error) {
	status := &Status{}

	// Check if alive
	version, err := c.command("version")
	if err != nil {
		return nil, fmt.Errorf("status check failed: %w", err)
	}
	status.Alive = true
	status.Version = strings.TrimSpace(version)

	// Get uptime
	uptime, err := c.command("uptime")
	if err == nil {
		status.Uptime = strings.TrimSpace(uptime)
	}

	// Get queue length
	queueStatus, err := c.command("queue.queue")
	if err == nil {
		// Count lines in queue response
		lines := strings.Split(strings.TrimSpace(queueStatus), "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				status.QueueLength++
			}
		}
	}

	return status, nil
}

// QueueList returns the list of items currently in the queue.
func (c *Client) QueueList() ([]string, error) {
	response, err := c.command("queue.queue")
	if err != nil {
		return nil, fmt.Errorf("failed to list queue: %w", err)
	}

	var items []string
	lines := strings.Split(strings.TrimSpace(response), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			items = append(items, line)
		}
	}

	return items, nil
}

// command sends a command to Liquidsoap and reads the response.
// Commands end with newline, responses end with "END\r\n".
// Auto-reconnects if the connection was closed.
func (c *Client) command(cmd string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Try to reconnect if not connected
	if c.conn == nil {
		if err := c.reconnectLocked(); err != nil {
			return "", err
		}
	}

	// Try to send command, reconnect once if it fails
	response, err := c.sendCommandLocked(cmd)
	if err != nil {
		// Connection might be stale, try to reconnect once
		slog.Debug("Command failed, attempting reconnect", "error", err)
		if reconnErr := c.reconnectLocked(); reconnErr != nil {
			return "", fmt.Errorf("reconnect failed after command error: %w", reconnErr)
		}
		// Retry the command after reconnect
		response, err = c.sendCommandLocked(cmd)
		if err != nil {
			return "", err
		}
	}

	return response, nil
}

// reconnectLocked reconnects to Liquidsoap. Must be called with mutex held.
func (c *Client) reconnectLocked() error {
	// Guard against concurrent reconnect attempts
	if c.reconnecting {
		return fmt.Errorf("reconnection already in progress")
	}
	c.reconnecting = true
	defer func() { c.reconnecting = false }()

	// Close existing connection if any
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}

	addr := fmt.Sprintf("%s:%d", c.host, c.port)

	// Use a short timeout for reconnection to avoid blocking under mutex
	conn, err := net.DialTimeout("tcp", addr, c.reconnectTimeout)
	if err != nil {
		return fmt.Errorf("failed to reconnect to %s: %w", addr, err)
	}

	c.conn = conn
	slog.Debug("Reconnected to Liquidsoap", "addr", addr)
	return nil
}

// sendCommandLocked sends a command and reads the response. Must be called with mutex held.
func (c *Client) sendCommandLocked(cmd string) (string, error) {
	if c.conn == nil {
		return "", ErrNotConnected
	}

	// Set write deadline
	if err := c.conn.SetWriteDeadline(time.Now().Add(c.commandTimeout)); err != nil {
		return "", fmt.Errorf("failed to set write deadline: %w", err)
	}

	// Send command with newline
	cmdBytes := []byte(cmd + "\n")
	if _, err := c.conn.Write(cmdBytes); err != nil {
		return "", fmt.Errorf("failed to send command: %w", err)
	}

	// Set read deadline
	if err := c.conn.SetReadDeadline(time.Now().Add(c.commandTimeout)); err != nil {
		return "", fmt.Errorf("failed to set read deadline: %w", err)
	}

	// Read response until "END\r\n"
	reader := bufio.NewReader(c.conn)
	var response strings.Builder

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read response: %w", err)
		}

		// Liquidsoap ends responses with "END\r\n" or just "END\n"
		trimmed := strings.TrimSpace(line)
		if trimmed == "END" {
			break
		}

		response.WriteString(line)
	}

	return strings.TrimSpace(response.String()), nil
}

// parseMetadata parses Liquidsoap metadata response into a map.
// Format: key="value" on each line.
func parseMetadata(response string) map[string]string {
	metadata := make(map[string]string)

	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Find the first '=' separator
		idx := strings.Index(line, "=")
		if idx == -1 {
			continue
		}

		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])

		// Remove surrounding quotes from value
		if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
			value = value[1 : len(value)-1]
		}

		metadata[key] = value
	}

	return metadata
}
