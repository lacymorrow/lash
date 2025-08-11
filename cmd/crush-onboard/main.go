package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/crush/internal/app"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/db"
	"github.com/charmbracelet/crush/internal/tui"
)

func main() {
	debug := os.Getenv("CRUSH_DEBUG") == "1"
	cwd := os.Getenv("CRUSH_CWD")
	if cwd != "" {
		if err := os.Chdir(cwd); err != nil {
			fmt.Fprintln(os.Stderr, "failed to chdir:", err)
			os.Exit(1)
		}
	}

	wd, _ := os.Getwd()
	cfg, err := config.Init(wd, debug)
	if err != nil {
		fmt.Fprintln(os.Stderr, "config error:", err)
		os.Exit(1)
	}

	ctx := context.Background()
	conn, err := db.Connect(ctx, cfg.Options.DataDirectory)
	if err != nil {
		fmt.Fprintln(os.Stderr, "db error:", err)
		os.Exit(1)
	}

	a, err := app.New(ctx, conn, cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "app error:", err)
		os.Exit(1)
	}
	defer a.Shutdown()

	p := tea.NewProgram(
		tui.New(a),
		tea.WithAltScreen(),
		tea.WithContext(ctx),
	)

	if _, err := p.Run(); err != nil {
		slog.Error("onboarding TUI error", "error", err)
		os.Exit(1)
	}
}
