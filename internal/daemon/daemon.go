// Package daemon manages the Pi RPC daemon lifecycle
package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mtreilly/arc-ai/pkg/bridge"
)

// Daemon manages the Pi subprocess
type Daemon struct {
	config    *bridge.Config
	cmd       *exec.Cmd
	listener  net.Listener
	mu        sync.RWMutex
	started   time.Time
	requests  int
	piVersion string
	running   bool
}

// New creates a new daemon instance
func New(cfg *bridge.Config) *Daemon {
	return &Daemon{
		config: cfg,
	}
}

// Start launches the Pi daemon
func (d *Daemon) Start(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.running {
		return fmt.Errorf("daemon already running")
	}

	// Ensure socket directory exists
	socketDir := filepath.Dir(d.config.SocketPath)
	if err := os.MkdirAll(socketDir, 0755); err != nil {
		return fmt.Errorf("failed to create socket directory: %w", err)
	}

	// Remove old socket if exists
	if _, err := os.Stat(d.config.SocketPath); err == nil {
		os.Remove(d.config.SocketPath)
	}

	// Create Unix socket listener
	listener, err := net.Listen("unix", d.config.SocketPath)
	if err != nil {
		return fmt.Errorf("failed to create socket: %w", err)
	}
	d.listener = listener

	// Start Pi in RPC mode
	piCmd := exec.CommandContext(ctx, d.config.PiPath, "--mode", "rpc")
	piCmd.Env = append(os.Environ(),
		"PI_CODING_AGENT_DIR="+expandHome(d.config.ExtensionsDir),
	)

	// Start Pi (it will connect to our socket)
	if err := piCmd.Start(); err != nil {
		listener.Close()
		return fmt.Errorf("failed to start Pi: %w", err)
	}

	d.cmd = piCmd
	d.running = true
	d.started = time.Now()

	// Accept Pi connection
	go d.acceptConnections(ctx)

	// Wait a moment for Pi to connect
	time.Sleep(500 * time.Millisecond)

	return nil
}

// Stop shuts down the daemon
func (d *Daemon) Stop() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.running {
		return nil
	}

	// Stop accepting new connections
	if d.listener != nil {
		d.listener.Close()
	}

	// Kill Pi process
	if d.cmd != nil && d.cmd.Process != nil {
		d.cmd.Process.Kill()
		d.cmd.Wait()
	}

	// Clean up socket
	os.Remove(d.config.SocketPath)

	d.running = false
	return nil
}

// Status returns current daemon status
func (d *Daemon) Status() *bridge.Status {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return &bridge.Status{
		Running:        d.running,
		PID:            d.getPID(),
		PiVersion:      d.piVersion,
		Uptime:         time.Since(d.started),
		RequestsServed: d.requests,
	}
}

// Query sends a request to Pi and returns the response
func (d *Daemon) Query(ctx context.Context, req *bridge.Request) (*bridge.Response, error) {
	d.mu.RLock()
	if !d.running {
		d.mu.RUnlock()
		return nil, fmt.Errorf("daemon not running")
	}
	d.mu.RUnlock()

	// Connect to Pi via Unix socket
	conn, err := net.Dial("unix", d.config.SocketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to daemon: %w", err)
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

	d.mu.Lock()
	d.requests++
	d.mu.Unlock()

	return &resp, nil
}

// acceptConnections handles incoming Pi connections
func (d *Daemon) acceptConnections(ctx context.Context) {
	for {
		conn, err := d.listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				if d.running {
					fmt.Fprintf(os.Stderr, "Accept error: %v\n", err)
				}
				return
			}
		}

		go d.handleConnection(ctx, conn)
	}
}

// handleConnection processes a single Pi connection
func (d *Daemon) handleConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	// TODO: Implement bidirectional RPC with Pi
	// For now, this is a simple request-response

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	for {
		var req bridge.Request
		if err := decoder.Decode(&req); err != nil {
			if err != io.EOF {
				fmt.Fprintf(os.Stderr, "Decode error: %v\n", err)
			}
			return
		}

		resp := d.processRequest(ctx, &req)

		if err := encoder.Encode(resp); err != nil {
			fmt.Fprintf(os.Stderr, "Encode error: %v\n", err)
			return
		}
	}
}

// processRequest handles a single request
func (d *Daemon) processRequest(ctx context.Context, req *bridge.Request) *bridge.Response {
	start := time.Now()

	// Build Pi command with appropriate extensions
	args := []string{"--mode", "rpc", "--no-session"} // No session for one-shot

	if req.Model != "" {
		args = append(args, "--model", req.Model)
	}

	// Add extensions if specified
	for _, ext := range req.Tools {
		args = append(args, "--extension", ext)
	}

	// TODO: Actually spawn Pi and communicate via RPC
	// For now, return a mock response

	return &bridge.Response{
		ID:        req.ID,
		RequestID: req.ID,
		Text:      fmt.Sprintf("Processed: %s (with tools: %v)", req.Prompt, req.Tools),
		Model:     req.Model,
		Latency:   time.Since(start).Milliseconds(),
	}
}

func (d *Daemon) getPID() int {
	if d.cmd != nil && d.cmd.Process != nil {
		return d.cmd.Process.Pid
	}
	return 0
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

// IsRunning checks if daemon is running
func (d *Daemon) IsRunning() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.running
}
