## Lash: Design (Fork of Charmbracelet Crush)

### Overview
Lash is a login-shell-friendly fork of Charmbracelet Crush that behaves as a natural shell with optional AI assist. It always runs the user’s real shell in a PTY, preserving history, tab completion, and full TTY features. Natural language is handled by the agent only when a command would otherwise fail (command not found), or when explicitly invoked via a hotkey/prefix. The visual experience is uniform: one terminal with a minimal statusline and a tiny confirmation panel when needed.

Reference: [charmbracelet/crush](https://github.com/charmbracelet/crush)

### Goals
- Mandatory MCP: reuse Crush’s native MCP support (stdio/http/SSE) and configuration.
- Shell-first UX: always run the real interactive shell in a PTY; history and tab completion are provided by the user’s shell, unmodified.
- Headless + SSH-safe: no GUI takeover; operates inside SSH sessions.
- Uniform UI: Shell, Agent, and Auto “modes” use the same terminal view; only routing behavior changes.
- Natural language fallback: when the shell would fail to resolve a command, Lash routes the original input to the agent.
- Minimal statusline with tiny confirmation panel; no chat UI required for core flows.

### Non-goals
- Re-implementing line editing, history, or tab completion.
- Building a GUI/terminal emulator.
- Replacing `ssh`; we delegate to the system `ssh` as needed.

### High-level Architecture
- Always-on PTY: spawn the user’s real shell (e.g., `/bin/zsh`) in a PTY with full pass-through of input/output and resize.
- Statusline + Panel: reserve the last terminal row for a thin statusline; render a small inline confirmation panel above it only when needed. PTY height is set to terminal height minus one row so full-screen apps don’t clobber the statusline.
- Mode Router (routing policy only):
  - Shell-first execution. The shell executes everything users type.
  - On “command not found” (CNF), route the original line to the Agent. If the agent suggests commands, show a confirm panel; on confirm, inject commands into the same PTY.
  - Optional explicit agent invocation via hotkey or prefix; otherwise flows are identical in appearance.
- Shell Hooks (no compromises):
  - Inject tiny RC fragments to install preexec/precmd (zsh) or DEBUG/PROMPT_COMMAND (bash) and command-not-found handlers.
  - Hooks emit metadata and CNF signals as sentinel lines (not visible to the user) that Lash parses from the PTY stream.
  - User RCs are preserved: our injected RC sources the user’s originals first, then adds small hook functions.
- MCP Agent: reuse Crush’s MCP plumbing; agent runs headless and only surfaces a confirm panel when proposing command execution.

### Process Model
1) Startup (interactive TTY):
   - If `LASH_DISABLE=1`, exec the real shell immediately.
   - Load `crush.json` (compatible), read Lash extensions.
   - Initialize PTY pass-through with the user’s shell and inject RC fragments for hooks.
   - Set PTY size to `(rows-1, cols)` and render statusline on the bottom row.
   - Mode is a routing policy only; the UI does not change across modes.
2) Non-interactive or `-c` mode:
   - Exec the real shell with original args (no PTY/TUI), preserving scripts/remote commands.
3) SSH:
   - Works transparently with PTY pass-through and the statusline row.

### Configuration
Existing Crush config is preserved. Lash adds additive fields only.

Example (JSON):
```json
{
  "$schema": "https://charm.land/crush.json",
  "options": { "debug": false },
  "mcp": { "filesystem": { "type": "stdio", "command": "node", "args": ["/path/to/mcp.js"] } },
  "lash": {
    "default_mode": "auto",
    "real_shell": "/bin/zsh",
    "statusline_position": "bottom",
    "auto_mode_enabled": true,
    "safety": { "confirm_agent_exec": true },
    "keymap": {
      "shell_mode": "ctrl+1",
      "agent_mode": "ctrl+2",
      "auto_mode":  "ctrl+3",
      "confirm":    "ctrl+enter",
      "help":       "ctrl+/"
    }
  }
}
```

MCP in Crush supports stdio, http, and sse transports and environment variable expansion. See: [charmbracelet/crush](https://github.com/charmbracelet/crush)

### UI
- Statusline (one row):
  - Left: Mode [Shell|Agent|Auto] (routing policy indicator)
  - Middle: Context (cwd, optionally `user@host`)
  - Right: Key hints (Ctrl-1/2/3; Ctrl-Enter; Help)
- Confirmation panel:
  - Appears above the statusline only when the agent proposes commands.
  - Options: Confirm (Ctrl-Enter), Revise (prompt), Cancel (Esc).
- No separate chat UI required for core shell workflows; the terminal remains the primary surface.

### Keyboard Defaults
- Ctrl-1: Routing policy to Shell-first only
- Ctrl-2: Routing policy to Agent-first (force agent for next line)
- Ctrl-3: Routing policy to Auto (Shell, falling back to Agent on CNF) [default]
- Ctrl-Enter: Confirm execution of agent-suggested commands (into the same PTY)
- Ctrl-/: Help overlay (compact)

### Shell Hook Injection
- zsh:
  - Use `ZDOTDIR` to point to a small injected directory containing `.zshenv`/`.zshrc` fragments that first source the user’s originals, then define:
    - `preexec()`: emit sentinel with the exact command.
    - `precmd()`: emit last status `$?` as sentinel for robust CNF detection.
    - `command_not_found_handler()`: emit sentinel with the original line and suppress default CNF text.
- bash:
  - Launch with `--rcfile` to an injected `.bashrc` that sources the user’s original rc and defines:
    - `trap '...' DEBUG` or `PROMPT_COMMAND` to capture last command and status.
    - `command_not_found_handle()`: emit sentinel and suppress default CNF text.
- fish (optional later): add equivalent event hooks.
- Sentinels are unique, single-line messages that Lash consumes from the PTY stream; they are not printed to the user’s screen.

### Routing Behavior (Uniform Look/Feel)
- Shell executes the line. If success: nothing special happens.
- On CNF:
  - Lash captures the original line via hooks and forwards it to the Agent as a natural-language request.
  - Agent may respond with:
    - Explanation only → show briefly in the statusline; no disruption.
    - Suggested command(s) → show confirmation panel; on confirm, inject into the PTY as if typed by the user.
- Explicit Agent:
  - Hotkey or prefix forces the next line to the Agent (without attempting shell execution first), but the terminal view remains unchanged.

### Error Handling & Resilience
- PTY failure: render inline error and exec the real shell directly as fallback.
- MCP failure: shell remains fully usable; show a non-intrusive status message.
- Resize propagated to PTY; statusline reflows accordingly.
- If hooks fail to load, fall back to Auto heuristic: first token executable check; if not, route to Agent.

### Security
- Confirm-to-execute is required for agent-suggested shell commands.
- Honor SSH `known_hosts` via system `ssh`.
- Redact likely secrets in logs; no secret persistence.
- `LASH_DISABLE=1` bypasses Lash at startup.

### Implementation Notes
- Language: Go.
- PTY: `creack/pty` for pass-through and programmatic execution in the same session.
- Statusline: render outside the PTY (reserved last row); PTY sized to rows-1.
- Shell hooks: generate injected RC directories at runtime; source user RCs first to preserve user setup and completion/history; append minimal hook functions.
- Agent integration: reuse Crush MCP, permissions, and logging. Confirm before injection.
- Packaging: keep GoReleaser and upstream licenses.

### Alternatives Considered
- Wrapper around Crush (unforked) plus a separate PTY TUI. Rejected for UX cohesion and deeper MCP features already embedded in Crush.
- Other agents like `ai-shell` or `gemini-cli` (no native MCP). Not chosen because MCP is mandatory. References: [BuilderIO/ai-shell](https://github.com/BuilderIO/ai-shell), [google-gemini/gemini-cli](https://github.com/google-gemini/gemini-cli)


