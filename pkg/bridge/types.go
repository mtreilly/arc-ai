// Package bridge defines the RPC protocol between arc-ai (Go) and Pi (Node.js)
package bridge

import "time"

// Request represents a query to the AI
type Request struct {
	ID        string            `json:"id"`
	Prompt    string            `json:"prompt"`
	Context   string            `json:"context,omitempty"`
	Tools     []string          `json:"tools,omitempty"`     // Which Pi extensions to load
	Skills    []string          `json:"skills,omitempty"`    // Pi skills to use
	Session   string            `json:"session,omitempty"`   // Session ID for continuity
	Model     string            `json:"model,omitempty"`     // Model override
	Format    string            `json:"format,omitempty"`    // "text", "json", "markdown"
	Metadata  map[string]string `json:"metadata,omitempty"`  // Additional context
}

// Response represents the AI's response
type Response struct {
	ID        string   `json:"id"`
	RequestID string   `json:"request_id"`
	Text      string   `json:"text"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	Error     string   `json:"error,omitempty"`
	Model     string   `json:"model"`
	Tokens    int      `json:"tokens,omitempty"`
	Latency   int64    `json:"latency_ms"`
}

// ToolCall represents a tool invocation
type ToolCall struct {
	Tool      string `json:"tool"`
	Arguments string `json:"arguments"`
	Result    string `json:"result,omitempty"`
}

// Status represents daemon status
type Status struct {
	Running      bool      `json:"running"`
	PID          int       `json:"pid"`
	PiVersion    string    `json:"pi_version"`
	Extensions   []string  `json:"extensions"`
	Uptime       time.Duration `json:"uptime"`
	RequestsServed int     `json:"requests_served"`
}

// ExtensionInfo describes available Pi extensions
type ExtensionInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Tools       []string `json:"tools"`
	Installed   bool   `json:"installed"`
}

// Config for the bridge
type Config struct {
	PiPath       string   `json:"pi_path"`        // Path to pi binary
	ExtensionsDir string  `json:"extensions_dir"` // Where Pi extensions live
	SocketPath   string   `json:"socket_path"`    // Unix socket for RPC
	LogLevel     string   `json:"log_level"`
	DefaultModel string   `json:"default_model"`
}

// DefaultConfig returns sensible defaults
func DefaultConfig() *Config {
	return &Config{
		PiPath:        "pi",
		ExtensionsDir: "~/.pi/agent/extensions",
		SocketPath:    "~/.config/arc/ai/daemon.sock",
		LogLevel:      "info",
		DefaultModel:  "claude-sonnet-4-5-20250929",
	}
}