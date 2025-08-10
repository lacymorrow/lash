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
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
)

// PTYRunner manages a persistent interactive shell in a PTY and executes commands within it.
// It is designed for sequential command execution. Calls to Exec are serialized via a mutex.
type PTYRunner struct {
	mu   sync.Mutex
	cmd  *exec.Cmd
	f    *os.File
	cwd  string
	env  []string
	once sync.Once
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
	// Prefer interactive shell so profiles/rc are loaded
	r.cmd = exec.Command(shellPath, "-i")
	if r.cwd != "" {
		r.cmd.Dir = r.cwd
	}
	r.cmd.Env = append([]string{}, r.env...)
	r.cmd.Env = append(r.cmd.Env, "LASH=1")

	f, err := pty.Start(r.cmd)
	if err != nil {
		r.cmd = nil
		return err
	}
	r.f = f
	return nil
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
