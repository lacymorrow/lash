package shell

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	xterm "github.com/charmbracelet/x/term"
	"github.com/creack/pty"
)

// PTYRunner manages a persistent interactive shell in a PTY and executes commands within it.
// It is designed for sequential command execution. Calls to Exec are serialized via a mutex.
type PTYRunner struct {
	mu    sync.Mutex
	cmd   *exec.Cmd
	f     *os.File
	cwd   string
	env   []string
	once  sync.Once
	rcDir string
}

var (
	userPTYOnce sync.Once
	userPTY     *PTYRunner
)

// GetUserPTY returns a process-wide persistent PTY running the user's shell.
func GetUserPTY(cwd string) *PTYRunner {
	userPTYOnce.Do(func() {
		r := &PTYRunner{cwd: cwd, env: os.Environ()}
		if err := r.start(); err != nil {
			// Fallback: if start fails, attempt again with HOME as cwd
			home, _ := os.UserHomeDir()
			r.cwd = home
			_ = r.start()
		}
		userPTY = r
	})
	return userPTY
}

func (r *PTYRunner) start() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cmd != nil {
		return nil
	}

	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		shellPath = "/bin/sh"
	}
	// Prefer interactive shell so profiles/rc are loaded, with injected hooks when possible
	base := filepath.Base(shellPath)
	switch base {
	case "zsh":
		dir, _ := r.prepareZshRC()
		r.rcDir = dir
		r.cmd = exec.Command(shellPath, "-i")
		if dir != "" {
			r.cmd.Env = append([]string{}, r.env...)
			r.cmd.Env = append(r.cmd.Env, "ZDOTDIR="+dir)
		}
	case "bash":
		rc, _ := r.prepareBashRC()
		r.rcDir = filepath.Dir(rc)
		r.cmd = exec.Command(shellPath, "-i", "--rcfile", rc)
	default:
		r.cmd = exec.Command(shellPath, "-i")
	}
	if r.cwd != "" {
		r.cmd.Dir = r.cwd
	}
	if len(r.cmd.Env) == 0 {
		r.cmd.Env = append([]string{}, r.env...)
	}
	r.cmd.Env = append(r.cmd.Env, "LASH=1")

	f, err := pty.Start(r.cmd)
	if err != nil {
		r.cmd = nil
		return err
	}
	r.f = f
	return nil
}

// prepareZshRC creates a temporary ZDOTDIR with minimal rc fragments that source the user's originals
// and install preexec/precmd/command_not_found_handler hooks that emit internal sentinels.
func (r *PTYRunner) prepareZshRC() (string, error) {
	dir, err := os.MkdirTemp("", "lash-zshrc-")
	if err != nil {
		return "", err
	}
	home, _ := os.UserHomeDir()
	// .zshenv: source user's .zshenv if present
	zshenv := filepath.Join(dir, ".zshenv")
	_ = os.WriteFile(zshenv, []byte(fmt.Sprintf(`
if [ -f %q ]; then
  . %q
fi
`, filepath.Join(home, ".zshenv"), filepath.Join(home, ".zshenv"))), 0o600)

	// .zshrc: source user's, then hooks
	zshrc := filepath.Join(dir, ".zshrc")
	content := &strings.Builder{}
	fmt.Fprintf(content, "if [ -f %q ]; then\n  . %q\nfi\n", filepath.Join(home, ".zshrc"), filepath.Join(home, ".zshrc"))
	hooks := `
preexec() {
  printf "__LASH_PREEXEC:%s\n" "$1"
}
precmd() {
  printf "__LASH_PRECMD:STATUS=%d\n" $?
}
command_not_found_handler() {
  printf "__LASH_CNF:%s\n" "$1"
  return 127
}
`
	content.WriteString(hooks)
	if err := os.WriteFile(zshrc, []byte(content.String()), 0o600); err != nil {
		return "", err
	}
	return dir, nil
}

// prepareBashRC creates a temporary rc file for bash that sources user's rc and installs DEBUG/PROMPT_COMMAND/CNF hooks
func (r *PTYRunner) prepareBashRC() (string, error) {
	dir, err := os.MkdirTemp("", "lash-bashrc-")
	if err != nil {
		return "", err
	}
	home, _ := os.UserHomeDir()
	rc := filepath.Join(dir, "lash.bashrc")
	b := &strings.Builder{}
	fmt.Fprintf(b, "if [ -f %q ]; then\n  . %q\nfi\n", filepath.Join(home, ".bashrc"), filepath.Join(home, ".bashrc"))
	b.WriteString(`
lash_last_status() {
  printf "__LASH_PRECMD:STATUS=%d\n" $?
}
PROMPT_COMMAND="lash_last_status; $PROMPT_COMMAND"
trap 'printf "__LASH_PREEXEC:%s\n" "$BASH_COMMAND"' DEBUG
command_not_found_handle() {
  printf "__LASH_CNF:%s\n" "$1"
  return 127
}
`)
	if err := os.WriteFile(rc, []byte(b.String()), 0o600); err != nil {
		return "", err
	}
	return rc, nil
}

// Exec runs a command inside the PTY shell and captures output until a unique sentinel is observed.
// Returns combined output and an error if the command exit status is non-zero or the context expires.
func (r *PTYRunner) Exec(ctx context.Context, command string) (string, string, error) {
	if err := r.start(); err != nil {
		return "", "", err
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	reader := bufio.NewReader(r.f)

	runWithSentinel := func(ctx context.Context, cmd string) (out string, exit int, err error) {
		// Unique markers
		token := make([]byte, 8)
		_, _ = rand.Read(token)
		sid := hex.EncodeToString(token)
		begin := "__LASH_BEGIN__" + sid
		end := "__LASH_END__" + sid
		// Disable input echo to avoid the typed command appearing in output
		// Then print BEGIN, run the command with stdin closed (</dev/null) capturing stderr, re-enable echo, and print END:status
		// Compose the control sequence in one string (status is printed in the second printf using $rc)
		line := fmt.Sprintf("stty -echo; printf '%s\\n'; { %s; } </dev/null 2>&1; rc=$?; stty echo; printf '\n%s:'; printf '%%d\\n' $rc\n", begin, cmd, end)
		// Note: we avoid echoing the typed command and delimit output clearly between begin and end markers
		if _, err := io.WriteString(r.f, line); err != nil {
			return "", 0, err
		}
		var b strings.Builder
		const maxBytes = 128 * 1024
		deadline := time.Time{}
		if dl, ok := ctx.Deadline(); ok {
			deadline = dl
		}
		started := false
		sentInterrupt := false
		interruptDeadline := time.Time{}
		for {
			now := time.Now()
			if !deadline.IsZero() && now.After(deadline) {
				if !sentInterrupt {
					_, _ = r.f.Write([]byte{0x03})
					sentInterrupt = true
					interruptDeadline = now.Add(1200 * time.Millisecond)
				} else if !interruptDeadline.IsZero() && now.After(interruptDeadline) {
					return strings.TrimRight(b.String(), "\n"), 0, context.DeadlineExceeded
				}
			}
			ln, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					time.Sleep(10 * time.Millisecond)
					continue
				}
				return b.String(), 0, err
			}
			ln = strings.ReplaceAll(ln, "\r", "")
			t := strings.TrimSpace(ln)
			if !started {
				if t == begin {
					started = true
				}
				continue
			}
			if strings.HasPrefix(t, end+":") {
				parts := strings.SplitN(t, ":", 2)
				if len(parts) == 2 {
					fmt.Sscanf(parts[1], "%d", &exit)
				}
				break
			}
			b.WriteString(ln)
			if b.Len() >= maxBytes && !sentInterrupt {
				_, _ = r.f.Write([]byte{0x03})
				sentInterrupt = true
				interruptDeadline = time.Now().Add(1200 * time.Millisecond)
			}
		}
		outStr := strings.TrimRight(b.String(), "\n")
		if len(outStr) >= maxBytes {
			outStr += "\n[truncated]"
		}
		return outStr, exit, nil
	}

	// Run the user command
	out, status, err := runWithSentinel(ctx, command)

	// Best-effort: update CWD if the command may have changed directories
	if strings.HasPrefix(strings.TrimSpace(command), "cd") || strings.Contains(command, "; cd ") {
		if pwdOut, _, e := runWithSentinel(context.Background(), "pwd"); e == nil {
			r.cwd = strings.TrimSpace(pwdOut)
		}
	}

	if err == nil && status != 0 {
		err = fmt.Errorf("exit status %d", status)
	}
	return out, "", err
}

// GetWorkingDir returns the last known working directory. It may lag until after a cd command updates it.
func (r *PTYRunner) GetWorkingDir() string {
	if r.cwd == "" {
		return r.cmd.Dir
	}
	return r.cwd
}

// SetWorkingDir sets desired starting cwd; only effective before start.
func (r *PTYRunner) SetWorkingDir(dir string) error {
	if dir == "" {
		return nil
	}
	if !filepath.IsAbs(dir) {
		abs, err := filepath.Abs(dir)
		if err == nil {
			dir = abs
		}
	}
	r.cwd = dir
	return nil
}

// RunPassThrough starts or reuses the PTY session and forwards stdin/stdout with a reserved statusline.
// The PTY is sized to rows-1 to reserve the bottom row for status. The provided statusFn is called to
// render the statusline text on each resize tick.
// RunPassThrough wires the PTY to the user's terminal. It reserves the last row for a status line.
// suggestionActive should return whether a confirmation is pending and a short preview string.
// onConfirm should return the command to inject into the PTY and true if it was handled.
// onCancel will be called when the user cancels the confirmation.
func (r *PTYRunner) RunPassThrough(
	ctx context.Context,
	statusFn func() string,
	suggestionActive func() (bool, string),
	onConfirm func() (string, bool),
	onCancel func(),
	onAgentRequest func(prompt string),
	onPreexec func(cmd string) (cancel bool, agentPrompt string),
) error {
	if err := r.start(); err != nil {
		return err
	}

	// Put the user's terminal in raw mode
	oldState, err := xterm.MakeRaw(uintptr(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	defer func() {
		_ = xterm.Restore(uintptr(os.Stdin.Fd()), oldState)
	}()

	// Initial size and draw
	resize := func() {
		width, height, _ := xterm.GetSize(uintptr(os.Stdout.Fd()))
		if height <= 0 {
			height = 1
		}
		// Reserve last row for our statusline
		_ = pty.Setsize(r.f, &pty.Winsize{Cols: uint16(width), Rows: uint16(max(height-1, 1))})
		// Draw statusline on the real terminal bottom row
		drawStatusLine(width, height, statusFn)
	}
	resize()

	// Handle SIGWINCH
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGWINCH)
	defer signal.Stop(sigs)
	go func() {
		for range sigs {
			resize()
		}
	}()

	// Forward input to PTY and output back to stdout, filtering internal sentinels
	errCh := make(chan error, 2)

	// stdin handler: forward to PTY unless a suggestion is active, in which case capture Enter/Esc
	go func() {
		buf := make([]byte, 1)
		for {
			n, e := os.Stdin.Read(buf)
			if n > 0 {
				if active, _ := suggestionActive(); active {
					b := buf[0]
					if b == 0x1b { // ESC cancels
						if onCancel != nil {
							onCancel()
						}
						resize()
						continue
					}
					if b == '\r' || b == '\n' { // Enter confirms
						if onConfirm != nil {
							if cmd, ok := onConfirm(); ok && cmd != "" {
								io.WriteString(r.f, cmd+"\n")
							}
						}
						resize()
						continue
					}
					// ignore other keys while confirming
					continue
				}
				// normal mode: forward to PTY
				if _, ew := r.f.Write(buf[:n]); ew != nil {
					errCh <- ew
					return
				}
			}
			if e != nil {
				errCh <- e
				return
			}
		}
	}()

	// pty -> stdout with sentinel handling; preserve normal echo
	go func() {
		reader := bufio.NewReader(r.f)
		for {
			b, e := reader.ReadBytes('\n')
			if len(b) > 0 {
				lineRaw := strings.ReplaceAll(string(b), "\r", "")
				line := strings.TrimRight(lineRaw, "\n")
				if strings.HasPrefix(line, "__LASH_") {
					// Handle sentinels
					switch {
					case strings.HasPrefix(line, "__LASH_CNF:"):
						if onAgentRequest != nil {
							original := strings.TrimPrefix(line, "__LASH_CNF:")
							go onAgentRequest(original)
						}
						// swallow
						continue
					case strings.HasPrefix(line, "__LASH_PREEXEC:"):
						if onPreexec != nil {
							cmd := strings.TrimPrefix(line, "__LASH_PREEXEC:")
							cancel, agentPrompt := onPreexec(cmd)
							if cancel {
								// Interrupt the just-starting command
								_, _ = r.f.Write([]byte{0x03})
							}
							if agentPrompt != "" && onAgentRequest != nil {
								go onAgentRequest(agentPrompt)
							}
						}
						// swallow
						continue
					case strings.HasPrefix(line, "__LASH_PRECMD:"):
						// status update; swallow
						continue
					}
				}
				os.Stdout.Write(b)
			}
			if e != nil {
				if e == io.EOF {
					time.Sleep(10 * time.Millisecond)
					continue
				}
				errCh <- e
				return
			}
		}
	}()

	// Wait for context or IO error
	select {
	case <-ctx.Done():
		return ctx.Err()
	case e := <-errCh:
		return e
	}
}

func drawStatusLine(width, height int, statusFn func() string) {
	if width <= 0 || height <= 0 {
		return
	}
	// Save cursor, draw on bottom row, restore cursor
	// ESC sequences: 7/8 save/restore (DEC), CSI {row};{col}H to position, 2K to clear line
	text := statusFn()
	if len(text) > width-1 {
		text = text[:width-1]
	}
	fmt.Fprintf(os.Stdout, "\x1b7\x1b[%d;1H\x1b[2K%s\x1b8", height, text)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
