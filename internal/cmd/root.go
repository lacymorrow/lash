package cmd

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"syscall"

	"github.com/charmbracelet/crush/internal/app"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/db"
	"github.com/charmbracelet/crush/internal/shell"
	"github.com/charmbracelet/crush/internal/version"
	"github.com/charmbracelet/fang"
	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.PersistentFlags().StringP("cwd", "c", "", "Current working directory")
	rootCmd.PersistentFlags().BoolP("debug", "d", false, "Debug")

	rootCmd.Flags().BoolP("help", "h", false, "Help")
	rootCmd.Flags().BoolP("yolo", "y", false, "Automatically accept all permissions (dangerous mode)")

	rootCmd.AddCommand(runCmd)
}

var rootCmd = &cobra.Command{
	Use:   "crush",
	Short: "Terminal-based AI assistant for software development",
	Long: `Crush is a powerful terminal-based AI assistant that helps with software development tasks.
It provides an interactive chat interface with AI capabilities, code analysis, and LSP integration
to assist developers in writing, debugging, and understanding code directly from the terminal.`,
	Example: `
# Run in interactive mode
crush

# Run with debug logging
crush -d

# Run with debug logging in a specific directory
crush -d -c /path/to/project

# Print version
crush -v

# Run a single non-interactive prompt
crush run "Explain the use of context in Go"

# Run in dangerous mode (auto-accept all permissions)
crush -y
  `,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Non-interactive guard and bypass
		if shouldExecRealShell() {
			return execRealShell()
		}

		app, err := setupApp(cmd)
		if err != nil {
			return err
		}
		defer app.Shutdown()

		// For now, start shell pass-through as the primary UI. Statusline shows routing policy.
		runner := shell.GetUserPTY(app.Config().WorkingDir())
		var suggestionMu sync.Mutex
		var pendingSuggestion string
		var agentSessionID string
		statusFn := func() string {
			suggestionMu.Lock()
			s := pendingSuggestion
			suggestionMu.Unlock()
			if s != "" {
				return fmt.Sprintf("Mode: %s  confirm: %s  (Enter=confirm Esc=cancel) | cwd: %s", app.Mode, s, runner.GetWorkingDir())
			}
			return fmt.Sprintf("Mode: %s  cwd: %s", app.Mode, runner.GetWorkingDir())
		}
		suggestionActive := func() (bool, string) {
			suggestionMu.Lock()
			s := pendingSuggestion
			suggestionMu.Unlock()
			return s != "", s
		}
		onConfirm := func() (string, bool) {
			suggestionMu.Lock()
			defer suggestionMu.Unlock()
			if pendingSuggestion == "" {
				return "", false
			}
			cmd := pendingSuggestion
			pendingSuggestion = ""
			return cmd, true
		}
		onCancel := func() {
			suggestionMu.Lock()
			pendingSuggestion = ""
			suggestionMu.Unlock()
		}
		// For now, simulate CNF fallback by listening to app events or future agent wire-up. Placeholder:
		// TODO: hook shell CNF sentinel to agent request and set pendingSuggestion when agent proposes a command.
		// On CNF, call agent and set pendingSuggestion to proposed command (requires Enter to execute)
		handleAgentRequest := func(prompt string) {
			go func() {
				// Create a session once for interactive suggestions
				if agentSessionID == "" {
					if sess, err := app.Sessions.Create(context.Background(), "Lash Interactive"); err == nil {
						agentSessionID = sess.ID
					} else {
						slog.Error("failed to create interactive session", "error", err)
						return
					}
				}
				done, err := app.CoderAgent.Run(context.Background(), agentSessionID, prompt)
				if err != nil {
					slog.Error("agent run failed", "error", err)
					return
				}
				for ev := range done {
					if ev.Error != nil {
						slog.Error("agent error", "error", ev.Error)
						break
					}
					if ev.Done {
						cmdText := parseSuggestedCommandFromText(ev.Message.Content().String())
						if cmdText != "" {
							suggestionMu.Lock()
							if len(cmdText) > 140 {
								pendingSuggestion = cmdText[:140] + "…"
							} else {
								pendingSuggestion = cmdText
							}
							suggestionMu.Unlock()
						}
						break
					}
				}
			}()
		}
		onPreexec := func(cmdline string) (bool, string) {
			// Mode switching: Ctrl-1/2/3 simulated via helper commands: :mode shell|agent|auto
			trimmed := strings.TrimSpace(cmdline)
			if strings.HasPrefix(trimmed, ":mode ") {
				mode := strings.TrimSpace(strings.TrimPrefix(trimmed, ":mode "))
				switch strings.ToLower(mode) {
				case "shell":
					app.Mode = "Shell"
					return true, ""
				case "agent":
					app.Mode = "Agent"
					return true, ""
				case "auto":
					app.Mode = "Auto"
					return true, ""
				}
			}
			// If in Agent mode, divert raw line to agent instead of shell
			if app.Mode == "Agent" {
				return true, cmdline
			}
			// In Auto: shell-first, but CNF will trigger agent via handleAgentRequest
			return false, ""
		}
		if err := runner.RunPassThrough(cmd.Context(), statusFn, suggestionActive, onConfirm, onCancel, handleAgentRequest, onPreexec); err != nil {
			slog.Error("Shell pass-through error", "error", err)
			return err
		}
		return nil
	},
}

func Execute() {
	if err := fang.Execute(
		context.Background(),
		rootCmd,
		fang.WithVersion(version.Version),
		fang.WithNotifySignal(os.Interrupt),
	); err != nil {
		os.Exit(1)
	}
}

// setupApp handles the common setup logic for both interactive and non-interactive modes.
// It returns the app instance, config, cleanup function, and any error.
func setupApp(cmd *cobra.Command) (*app.App, error) {
	debug, _ := cmd.Flags().GetBool("debug")
	yolo, _ := cmd.Flags().GetBool("yolo")
	ctx := cmd.Context()

	cwd, err := ResolveCwd(cmd)
	if err != nil {
		return nil, err
	}

	cfg, err := config.Init(cwd, debug)
	if err != nil {
		return nil, err
	}

	if cfg.Permissions == nil {
		cfg.Permissions = &config.Permissions{}
	}
	cfg.Permissions.SkipRequests = yolo

	// Connect to DB; this will also run migrations.
	conn, err := db.Connect(ctx, cfg.Options.DataDirectory)
	if err != nil {
		return nil, err
	}

	appInstance, err := app.New(ctx, conn, cfg)
	if err != nil {
		slog.Error("Failed to create app instance", "error", err)
		return nil, err
	}

	return appInstance, nil
}

func MaybePrependStdin(prompt string) (string, error) {
	if term.IsTerminal(os.Stdin.Fd()) {
		return prompt, nil
	}
	fi, err := os.Stdin.Stat()
	if err != nil {
		return prompt, err
	}
	if fi.Mode()&os.ModeNamedPipe == 0 {
		return prompt, nil
	}
	bts, err := io.ReadAll(os.Stdin)
	if err != nil {
		return prompt, err
	}
	return string(bts) + "\n\n" + prompt, nil
}

func ResolveCwd(cmd *cobra.Command) (string, error) {
	cwd, _ := cmd.Flags().GetString("cwd")
	if cwd != "" {
		err := os.Chdir(cwd)
		if err != nil {
			return "", fmt.Errorf("failed to change directory: %v", err)
		}
		return cwd, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %v", err)
	}
	return cwd, nil
}

// shouldExecRealShell decides whether to bypass Lash and exec the user's real shell.
func shouldExecRealShell() bool {
	if os.Getenv("LASH_DISABLE") == "1" {
		return true
	}
	// If stdin or stdout is not a terminal, avoid interactive TUI and exec the shell
	if !term.IsTerminal(os.Stdin.Fd()) || !term.IsTerminal(os.Stdout.Fd()) {
		return true
	}
	// Respect SSH_ORIGINAL_COMMAND (non-interactive ssh command execution)
	if os.Getenv("SSH_ORIGINAL_COMMAND") != "" {
		return true
	}
	return false
}

// execRealShell replaces the current process with the user's shell, passing along original args when applicable.
func execRealShell() error {
	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		shellPath = "/bin/sh"
	}
	// If SSH_ORIGINAL_COMMAND is set, run it via -c
	if cmd := os.Getenv("SSH_ORIGINAL_COMMAND"); cmd != "" {
		return syscallExec(shellPath, []string{shellPath, "-c", cmd})
	}
	// Fallback: exec interactive shell
	return syscallExec(shellPath, []string{shellPath, "-i"})
}

func syscallExec(bin string, argv []string) error {
	// Ensure the binary exists
	if _, err := exec.LookPath(bin); err != nil {
		return err
	}
	// Replace the current process (preserve env)
	return syscallExecRaw(bin, argv, os.Environ())
}

// syscallExecRaw is a small indirection for testability
var syscallExecRaw = func(bin string, argv []string, env []string) error {
	return syscall.Exec(bin, argv, env)
}

// parseSuggestedCommandFromText extracts a shell command from agent text, preferring fenced code blocks.
func parseSuggestedCommandFromText(s string) string {
	// ```bash\n...\n```
	re := regexp.MustCompile("(?s)```(?:sh|bash|zsh)?\\n(.*?)```")
	m := re.FindStringSubmatch(s)
	if len(m) >= 2 {
		cmd := strings.TrimSpace(m[1])
		// take first line
		if i := strings.IndexByte(cmd, '\n'); i >= 0 {
			cmd = strings.TrimSpace(cmd[:i])
		}
		cmd = strings.TrimPrefix(cmd, "$ ")
		return cmd
	}
	// inline: $ command
	re2 := regexp.MustCompile(`\$\s+([^\n]+)`)
	m2 := re2.FindStringSubmatch(s)
	if len(m2) >= 2 {
		return strings.TrimSpace(m2[1])
	}
	// fallback: first line
	line := s
	if i := strings.IndexByte(line, '\n'); i >= 0 {
		line = line[:i]
	}
	return strings.TrimSpace(line)
}
