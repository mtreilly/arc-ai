// Package client provides a Go client for the arc-ai daemon
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mtreilly/arc-ai/pkg/bridge"
)

// Client connects to the arc-ai daemon
type Client struct {
	socketPath string
	timeout    time.Duration
}

// New creates a new client
func New(socketPath string) *Client {
	if socketPath == "" {
		socketPath = "~/.config/arc/ai/daemon.sock"
	}
	return &Client{
		socketPath: expandHome(socketPath),
		timeout:    30 * time.Second,
	}
}

// WithTimeout sets the request timeout
func (c *Client) WithTimeout(timeout time.Duration) *Client {
	c.timeout = timeout
	return c
}

// Ask sends a simple question to the AI
func (c *Client) Ask(ctx context.Context, prompt string) (string, error) {
	req := &bridge.Request{
		ID:      generateID(),
		Prompt:  prompt,
		Format:  "text",
	}

	resp, err := c.Query(ctx, req)
	if err != nil {
		return "", err
	}

	if resp.Error != "" {
		return "", fmt.Errorf("AI error: %s", resp.Error)
	}

	return resp.Text, nil
}

// AskWithContext sends a question with context (e.g., piped input)
func (c *Client) AskWithContext(ctx context.Context, prompt, context string) (string, error) {
	req := &bridge.Request{
		ID:      generateID(),
		Prompt:  prompt,
		Context: context,
		Format:  "text",
	}

	resp, err := c.Query(ctx, req)
	if err != nil {
		return "", err
	}

	return resp.Text, nil
}

// AskWithTools sends a question with specific tools enabled
func (c *Client) AskWithTools(ctx context.Context, prompt string, tools []string) (string, error) {
	req := &bridge.Request{
		ID:     generateID(),
		Prompt: prompt,
		Tools:  tools,
		Format: "text",
	}

	resp, err := c.Query(ctx, req)
	if err != nil {
		return "", err
	}

	return resp.Text, nil
}

// Query sends a raw request to the daemon
func (c *Client) Query(ctx context.Context, req *bridge.Request) (*bridge.Response, error) {
	// Check if daemon is running
	if !c.IsDaemonRunning() {
		return nil, fmt.Errorf("arc-ai daemon not running. Start with: arc-ai start")
	}

	// Set timeout
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Connect to daemon
	conn, err := net.Dial("unix", c.socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to arc-ai daemon: %w", err)
	}
	defer conn.Close()

	// Send request
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(req); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Read response
	decoder := json.NewDecoder(conn)
	var resp bridge.Response
	if err := decoder.Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return &resp, nil
}

// IsDaemonRunning checks if the daemon is available
func (c *Client) IsDaemonRunning() bool {
	_, err := os.Stat(c.socketPath)
	return err == nil
}

// Status queries the daemon for its status
func (c *Client) Status(ctx context.Context) (*bridge.Status, error) {
	if !c.IsDaemonRunning() {
		return nil, fmt.Errorf("daemon not running")
	}

	// For now, return minimal status
	// In full implementation, query daemon for real status
	return &bridge.Status{
		Running: c.IsDaemonRunning(),
	}, nil
}

// generateID creates a simple request ID
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}
