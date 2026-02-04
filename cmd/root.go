// Package cmd implements the arc-ai CLI
package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/mtreilly/arc-ai/internal/client"
	"github.com/mtreilly/arc-ai/internal/daemon"
	"github.com/mtreilly/arc-ai/pkg/bridge"
)

var (
	socketPath string
	configPath string
)

// Execute runs the root command
func Execute() error {
	rootCmd := newRootCmd()
	return rootCmd.Execute()
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "arc-ai",
		Short: "Arc AI bridge - Connects arc toolkit to Pi harness",
		Long: `arc-ai manages the Pi AI harness daemon and provides a bridge between
the Go-based arc toolkit and the Node.js Pi coding agent.

It enables arc tools to leverage Pi's powerful AI capabilities, extensions,
and skill system while maintaining Unix-friendly interfaces.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.PersistentFlags().StringVar(&socketPath, "socket", "", "Path to daemon socket (default: ~/.config/arc/ai/daemon.sock)")
	cmd.PersistentFlags().StringVar(&configPath, "config", "", "Path to config file")

	cmd.AddCommand(newStartCmd())
	cmd.AddCommand(newStopCmd())
	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newAskCmd())
	cmd.AddCommand(newQueryCmd())

	return cmd
}

func getConfig() *bridge.Config {
	cfg := bridge.DefaultConfig()
	if socketPath != "" {
		cfg.SocketPath = socketPath
	}
	return cfg
}

func getClient() *client.Client {
	cfg := getConfig()
	return client.New(cfg.SocketPath)
}

// newStartCmd creates the start command
func newStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start the arc-ai daemon",
		Long:  `Starts the Pi harness daemon in the background.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := getConfig()
			d := daemon.New(cfg)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Handle shutdown signals
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigChan
				fmt.Fprintln(os.Stderr, "\nShutting down...")
				d.Stop()
				cancel()
			}()

			if err := d.Start(ctx); err != nil {
				return fmt.Errorf("failed to start daemon: %w", err)
			}

			fmt.Println("arc-ai daemon started")
			fmt.Printf("Socket: %s\n", cfg.SocketPath)
			fmt.Println("Press Ctrl+C to stop")

			// Wait for shutdown
			<-ctx.Done()
			return nil
		},
	}
}

// newStopCmd creates the stop command
func newStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the arc-ai daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := getConfig()
			d := daemon.New(cfg)
			
			// Try to connect and stop gracefully first
			c := client.New(cfg.SocketPath)
			if c.IsDaemonRunning() {
				// TODO: Send shutdown RPC
			}

			// Force stop
			if err := d.Stop(); err != nil {
				return fmt.Errorf("failed to stop daemon: %w", err)
			}

			fmt.Println("arc-ai daemon stopped")
			return nil
		},
	}
}

// newStatusCmd creates the status command
func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check daemon status",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := getClient()
			
			if !c.IsDaemonRunning() {
				fmt.Println("arc-ai daemon: not running")
				fmt.Println("Start with: arc-ai start")
				return nil
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			status, err := c.Status(ctx)
			if err != nil {
				fmt.Printf("arc-ai daemon: running (cannot get status: %v)\n", err)
				return nil
			}

			fmt.Println("arc-ai daemon: running")
			fmt.Printf("Socket: %s\n", getConfig().SocketPath)
			if status.PID > 0 {
				fmt.Printf("PID: %d\n", status.PID)
			}
			if status.Uptime > 0 {
				fmt.Printf("Uptime: %v\n", status.Uptime)
			}
			if status.RequestsServed > 0 {
				fmt.Printf("Requests served: %d\n", status.RequestsServed)
			}

			return nil
		},
	}
}

// newAskCmd creates the ask command (simple Q&A)
func newAskCmd() *cobra.Command {
	var (
		tools []string
		model string
	)

	cmd := &cobra.Command{
		Use:   "ask [question]",
		Short: "Ask the AI a question",
		Long: `Ask the AI a question. Reads from stdin if piped.

This is a convenience wrapper for common use cases.
For full control, use 'arc-ai query'.`,
		Example: `  # Simple question
  arc-ai ask "What is Go?"

  # With piped input
  cat code.go | arc-ai ask "Explain this code"

  # With tools enabled
  cat errors.log | arc-ai ask "What's wrong?" --tools tmux,security`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := getClient()

			if !c.IsDaemonRunning() {
				return fmt.Errorf("arc-ai daemon not running. Start with: arc-ai start")
			}

			// Read stdin if piped
			var input string
			stat, _ := os.Stdin.Stat()
			if (stat.Mode() & os.ModeCharDevice) == 0 {
				data, _ := io.ReadAll(os.Stdin)
				input = string(data)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			var answer string
			var err error

			if len(tools) > 0 {
				answer, err = c.AskWithTools(ctx, args[0], tools)
			} else if input != "" {
				answer, err = c.AskWithContext(ctx, args[0], input)
			} else {
				answer, err = c.Ask(ctx, args[0])
			}

			if err != nil {
				return err
			}

			fmt.Println(answer)
			return nil
		},
	}

	cmd.Flags().StringSliceVar(&tools, "tools", nil, "Enable specific tools")
	cmd.Flags().StringVar(&model, "model", "", "Model to use")

	return cmd
}

// newQueryCmd creates the query command (raw RPC)
func newQueryCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "query",
		Short:  "Send raw query to daemon (advanced)",
		Long:   `Send a raw JSON request to the daemon. Useful for scripting.`,
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not implemented yet")
		},
	}
}
