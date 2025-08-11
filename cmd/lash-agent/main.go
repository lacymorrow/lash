package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/charmbracelet/crush/internal/app"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/db"
)

func main() {
	var (
		debug bool
		cwd   string
	)
	flag.BoolVar(&debug, "debug", false, "Enable debug logs")
	flag.StringVar(&cwd, "cwd", "", "Working directory")
	flag.Parse()

	args := flag.Args()
	prompt := strings.TrimSpace(strings.Join(args, " "))
	if prompt == "" || prompt == "-" {
		// Read from stdin if no prompt provided or explicit '-'
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			b, _ := io.ReadAll(os.Stdin)
			prompt = strings.TrimSpace(string(b))
		}
	}
	if prompt == "" {
		fmt.Fprintln(os.Stderr, "usage: lash-agent [--cwd DIR] [--debug] <prompt | ->")
		os.Exit(2)
	}

	if cwd != "" {
		if err := os.Chdir(cwd); err != nil {
			fmt.Fprintln(os.Stderr, "cd:", err)
			os.Exit(1)
		}
	}

	// Load config and bootstrap minimal app for agent usage
	wd, _ := os.Getwd()
	cfg, err := config.Init(wd, debug)
	if err != nil {
		fmt.Printf("EXPLAIN Failed to load configuration: %v\n", err)
		return
	}

	// Connect ephemeral DB backing services
	ctx := context.Background()
	conn, err := db.Connect(ctx, cfg.Options.DataDirectory)
	if err != nil {
		fmt.Printf("EXPLAIN Agent unavailable (database error): %v\n", err)
		return
	}
	application, err := app.New(ctx, conn, cfg)
	if err != nil {
		fmt.Printf("EXPLAIN Agent failed to initialize: %v\n", err)
		return
	}
	defer application.Shutdown()

	if !cfg.IsConfigured() || application.CoderAgent == nil {
		fmt.Printf("EXPLAIN No AI provider configured. Add providers to crush.json and set API keys, then retry.\n")
		return
	}

	// Create a short-lived session
	sess, err := application.Sessions.Create(ctx, "Lash Agent CLI")
	if err != nil {
		fmt.Fprintln(os.Stderr, "session:", err)
		os.Exit(1)
	}

	// Auto-approve for CLI
	application.Permissions.AutoApproveSession(sess.ID)

	done, err := application.CoderAgent.Run(ctx, sess.ID, prompt)
	if err != nil {
		fmt.Printf("EXPLAIN Agent error: %v\n", err)
		return
	}

	for ev := range done {
		if ev.Error != nil && !errors.Is(ev.Error, context.Canceled) {
			fmt.Printf("EXPLAIN Agent error: %v\n", ev.Error)
			return
		}
		if ev.Done {
			text := ev.Message.Content().String()
			suggestion := parseSuggestedCommandFromText(text)
			if suggestion != "" {
				// Print suggestion first for simple shell parsing
				fmt.Printf("SUGGEST %s\n", suggestion)
				// Include a compact explanation line
				compact := compactOneLine(text)
				if compact != "" {
					fmt.Printf("EXPLAIN %s\n", compact)
				}
			} else {
				compact := compactOneLine(text)
				if compact != "" {
					fmt.Printf("EXPLAIN %s\n", compact)
				}
			}
			return
		}
	}
}

// parseSuggestedCommandFromText extracts a single-line shell command from agent text, preferring fenced code.
func parseSuggestedCommandFromText(s string) string {
	// ```bash\n...\n```
	re := regexp.MustCompile("(?s)```(?:sh|bash|zsh)?\\n(.*?)```")
	if m := re.FindStringSubmatch(s); len(m) >= 2 {
		cmd := strings.TrimSpace(m[1])
		if i := strings.IndexByte(cmd, '\n'); i >= 0 {
			cmd = strings.TrimSpace(cmd[:i])
		}
		cmd = strings.TrimPrefix(cmd, "$ ")
		return cmd
	}
	// inline: $ command
	re2 := regexp.MustCompile(`\$\s+([^\n]+)`)
	if m2 := re2.FindStringSubmatch(s); len(m2) >= 2 {
		return strings.TrimSpace(m2[1])
	}
	// fallback: first non-empty line
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			return line
		}
	}
	return ""
}

// compactOneLine returns a short, single-line summary from model text
func compactOneLine(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.TrimSpace(s)
	// Prefer the first sentence
	dot := strings.IndexByte(s, '.')
	if dot > 0 {
		s = s[:dot+1]
	} else if nl := strings.IndexByte(s, '\n'); nl > 0 {
		s = s[:nl]
	}
	s = strings.TrimSpace(s)
	// Collapse whitespace
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	// Clamp length
	if len(s) > 200 {
		s = s[:200] + "…"
	}
	return s
}
