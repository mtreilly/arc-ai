# arc-ai

Bridge between the Arc toolkit (Go) and Pi AI harness (Node.js).

## Overview

arc-ai connects your Go-based arc tools to the powerful Pi coding agent, enabling:

- **Unified AI interface** across all arc tools
- **Extension ecosystem** from Pi (27+ tools available)
- **Session management** for complex workflows
- **Unix-friendly** pipes and structured output

## Architecture

```
┌─────────────┐     RPC/Unix      ┌─────────────┐     Node.js       ┌─────────┐
│  arc tools  │◄───socket/JSON───►│   arc-ai    │◄───RPC mode─────►│   Pi    │
│  (Go)       │                   │   daemon    │                   │ (Node)  │
└─────────────┘                   └─────────────┘                   └─────────┘
     │                                   │
     │                            ┌──────┴──────┐
     │                            │  Extensions │
     │                            │  - tmux     │
     │                            │  - security │
     │                            │  - deps     │
     │                            └─────────────┘
```

## Installation

```bash
# Install arc-ai
cd arc-ai
go install

# Ensure Pi is installed
npm install -g @mariozechner/pi-coding-agent

# Start the daemon
arc-ai start
```

## Usage

### Start the daemon

```bash
arc-ai start                    # Run in foreground
arc-ai start &                 # Run in background
arc-ai status                   # Check if running
arc-ai stop                     # Stop daemon
```

### Ask questions

```bash
# Simple question
arc-ai ask "What is Go?"

# With piped input
cat code.go | arc-ai ask "Explain this"

# With tools enabled
cat errors.log | arc-ai ask "Analyze" --tools security,tmux

# From tmux pane
arc-tmux follow --pane dev:1.0 | arc-ai ask "What's wrong?"
```

### Using arc-ask (refactored)

```bash
# arc-ask now uses arc-ai bridge
arc-ask "Question"              # Uses daemon if available
arc-ask "Question" --tools tmux # Enable specific tools
```

## Configuration

Environment variables:

```bash
export ARC_AI_SOCKET="~/.config/arc/ai/daemon.sock"
export ARC_AI_EXTENSIONS="~/.pi/agent/extensions"
```

## Integration with Arc Tools

### arc-tmux
```bash
# Capture and analyze
arc-tmux follow --pane dev:1.0 | arc-ai ask "Debug this"
```

### arc-security
```bash
# Security audit with explanation
arc-security audit --json | arc-ai ask "Summarize vulnerabilities"
```

### arc-deps
```bash
# Dependency analysis
arc-deps check --json | arc-ai ask "Suggest updates"
```

## Protocol

The bridge uses JSON-RPC over Unix sockets:

**Request:**
```json
{
  "id": "unique-id",
  "prompt": "Question",
  "context": "Optional context",
  "tools": ["tmux", "security"],
  "format": "text"
}
```

**Response:**
```json
{
  "id": "unique-id",
  "text": "Answer",
  "model": "claude-sonnet-4-5",
  "latency_ms": 1234
}
```

## Development

### Project structure

```
arc-ai/
├── cmd/                    # CLI commands
├── internal/
│   ├── daemon/            # Pi process management
│   ├── client/            # Go client library
│   └── config/            # Configuration
├── pkg/bridge/            # Protocol types
└── main.go
```

### Testing

```bash
go test ./...
```

## Comparison

| Feature | arc-ask (old) | arc-ask (new) | Pi directly |
|---------|---------------|---------------|-------------|
| Unix pipes | ✅ | ✅ | ❌ |
| Extensions | ❌ | ✅ | ✅ |
| Sessions | ❌ | ✅ | ✅ |
| Startup time | Fast | Medium | Slow |
| Node.js dep | ❌ | ✅ (daemon) | ✅ |

## Roadmap

- [ ] Full RPC implementation
- [ ] Extension auto-discovery
- [ ] Session persistence
- [ ] Batch processing mode
- [ ] Streaming responses

## License

MIT
