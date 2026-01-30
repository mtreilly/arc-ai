// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yourorg/arc-sdk/output"
)

// NewRootCmd creates the root command for arc-ai.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "arc-ai",
		Short: "AI-powered tools",
		Long: `AI-powered development tools.

Generate commit messages, analyze code, and more using AI models.`,
	}

	root.AddCommand(newCommitCmd())
	root.AddCommand(newAskCmd())

	return root
}

func newCommitCmd() *cobra.Command {
	var model string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "commit",
		Short: "Generate AI commit message",
		Long: `Generate a commit message based on staged changes.

This command runs 'git diff --cached' and sends the diff to an AI model
to generate a meaningful commit message.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			// Get staged diff
			diffCmd := exec.CommandContext(ctx, "git", "diff", "--cached")
			diffOutput, err := diffCmd.Output()
			if err != nil {
				return fmt.Errorf("git diff failed: %w", err)
			}

			if len(diffOutput) == 0 {
				return fmt.Errorf("no staged changes")
			}

			diff := string(diffOutput)
			if len(diff) > 10000 {
				diff = diff[:10000] + "\n... (truncated)"
			}

			// Generate commit message using AI
			prompt := fmt.Sprintf(`Generate a concise git commit message for the following diff.
Use conventional commit format (feat:, fix:, docs:, refactor:, etc.).
Keep the message under 72 characters for the subject line.
Include a brief body if needed.

Diff:
%s

Respond with ONLY the commit message, no explanations.`, diff)

			fmt.Println("Generating commit message...")

			message, err := askAI(ctx, prompt, model)
			if err != nil {
				return fmt.Errorf("AI request failed: %w", err)
			}

			message = strings.TrimSpace(message)

			if dryRun {
				fmt.Printf("\nSuggested commit message:\n%s\n", message)
				return nil
			}

			// Confirm with user
			fmt.Printf("\nSuggested commit message:\n%s\n\n", message)
			fmt.Print("Use this message? [Y/n]: ")

			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))

			if response != "" && response != "y" && response != "yes" {
				fmt.Println("Commit cancelled.")
				return nil
			}

			// Create the commit
			commitCmd := exec.CommandContext(ctx, "git", "commit", "-m", message)
			commitCmd.Stdout = os.Stdout
			commitCmd.Stderr = os.Stderr

			if err := commitCmd.Run(); err != nil {
				return fmt.Errorf("git commit failed: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&model, "model", "", "AI model to use")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show message without committing")

	return cmd
}

func newAskCmd() *cobra.Command {
	var model string
	var out output.OutputOptions

	cmd := &cobra.Command{
		Use:   "ask <question>",
		Short: "Ask AI a question",
		Long: `Ask an AI model a question and get a response.

The question can be provided as arguments or piped via stdin.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := out.Resolve(); err != nil {
				return err
			}

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			var question string
			if len(args) > 0 {
				question = strings.Join(args, " ")
			} else {
				// Read from stdin
				scanner := bufio.NewScanner(os.Stdin)
				var lines []string
				for scanner.Scan() {
					lines = append(lines, scanner.Text())
				}
				question = strings.Join(lines, "\n")
			}

			if strings.TrimSpace(question) == "" {
				return fmt.Errorf("no question provided")
			}

			response, err := askAI(ctx, question, model)
			if err != nil {
				return err
			}

			if out.Is(output.OutputJSON) {
				return output.JSON(map[string]string{
					"question": question,
					"response": response,
				})
			}

			fmt.Println(response)
			return nil
		},
	}

	cmd.Flags().StringVar(&model, "model", "", "AI model to use")
	out.AddOutputFlags(cmd, output.OutputTable)

	return cmd
}

// askAI sends a prompt to the AI and returns the response.
// It tries multiple providers in order of preference.
func askAI(ctx context.Context, prompt, model string) (string, error) {
	// Try Claude CLI first
	if _, err := exec.LookPath("claude"); err == nil {
		return askClaude(ctx, prompt, model)
	}

	// Try codex CLI
	if _, err := exec.LookPath("codex"); err == nil {
		return askCodex(ctx, prompt, model)
	}

	return "", fmt.Errorf("no AI provider available (install claude or codex CLI)")
}

func askClaude(ctx context.Context, prompt, model string) (string, error) {
	args := []string{"--print"}
	if model != "" {
		args = append(args, "--model", model)
	}
	args = append(args, prompt)

	cmd := exec.CommandContext(ctx, "claude", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("claude failed: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

func askCodex(ctx context.Context, prompt, model string) (string, error) {
	args := []string{"ask"}
	if model != "" {
		args = append(args, "--model", model)
	}
	args = append(args, prompt)

	cmd := exec.CommandContext(ctx, "codex", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("codex failed: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}
