## Lash: Requirements (Fork of Charmbracelet Crush)

### Must-haves
- MCP is mandatory. Use Crush’s built-in MCP mechanisms and configuration; do not remove or regress them. Reference: [charmbracelet/crush](https://github.com/charmbracelet/crush)
- Headless operation in any Unix terminal, including over SSH.
- Shell-first, uniform UX by default:
  - Always run the user’s real shell (`$SHELL` or configured) in a PTY with full pass-through; preserve native history and tab completion.
  - Modes are routing policies only; the terminal view is identical across Shell/Agent/Auto.
  - Natural-language fallback on command-not-found (CNF). Explicit agent invocation available via hotkey/prefix.
- Non-interactive or `-c`: immediately exec the real shell with original args (preserve scripts/remote commands).
- Minimal UI with a reserved one-line statusline and a tiny confirmation panel when needed.
- Agent-suggested commands never auto-execute; require explicit confirmation (default Ctrl-Enter).
- Configuration via `crush.json` with a `lash` extension block; environment and flags override.
- Logging compatible with Crush’s logging; add redaction for likely secrets.

### Functional Requirements
1) Shell-first PTY
   - Spawn real shell in a PTY; pass-through input/output.
   - Reserve last terminal row for statusline by sizing PTY to `(rows-1, cols)`; propagate resize.
   - Support full-screen TTY programs (vim, less, fzf, tmux) without interference.
   - Execute confirmed agent suggestions inside the same PTY session.

2) Command-not-found Fallback
   - Install minimal shell hooks to detect CNF and capture the original line:
     - zsh: `preexec`, `precmd`, `command_not_found_handler` via injected RC fragments using `ZDOTDIR`.
     - bash: `DEBUG`/`PROMPT_COMMAND`, `command_not_found_handle` via `--rcfile` to injected rc.
   - Emit unique, non-visible sentinels that Lash consumes from the PTY stream.
   - On CNF, forward the original line to the agent; show a compact confirmation when the agent proposes shell commands.

3) Agent Integration
   - Preserve Crush’s agent flow and MCP configuration (stdio/http/sse; env expansion).
   - Confirm-to-execute before injecting suggested commands into the PTY.
   - Optional explicit agent invocation for the next line via hotkey/prefix.

4) Routing Policies (Modes)
   - Shell: execute only in shell (no CNF fallback unless explicitly invoked).
   - Agent: force agent for next line (no shell attempt); terminal remains unchanged.
   - Auto (default): shell-first with CNF fallback to agent.
   - Persist last-selected policy; defaults to Auto on first run.

5) Login Shell Safety
   - Works as the user’s login shell (`chsh`) and within SSH sessions.
   - Non-interactive behavior falls through to real shell.
   - Bypass via `LASH_DISABLE=1` env var to exec real shell immediately.

6) SSH Interop
   - Running under SSH should behave like a normal terminal, including statusline reservation and resize propagation.
   - Optional convenience: `lash ssh user@host` delegates to system `ssh` in a PTY.

7) Configuration
   - Preserve `crush.json` schema; add `lash` namespaced keys for defaults, keymap, safety, shell path. Persist last-selected policy.
   - Hot-reload not required; re-read on startup is sufficient.

8) Logging/Observability
   - Reuse Crush’s logging; log to project-local and/or state directory per Crush behavior.
   - Redact likely secrets via regex patterns; toggle debug via config/flag.

### Non-functional Requirements
- Startup latency for interactive sessions: < 200ms on typical dev hardware.
- Memory: Shell-only baseline < 60MB; with Agent active < 140MB.
- CPU: idle usage negligible; streaming under 1 core in common cases.
- Binaries for macOS and Linux (amd64, arm64); no GUI dependencies.

### Security
- Confirm-to-execute is on by default and cannot be disabled without an explicit config option.
- Honor SSH `known_hosts`; do not silently accept new host keys.
- No secrets stored; redact in logs and transcripts.

### Compatibility & Limits
- Compatible with POSIX shells; does not reimplement line editing/completion.
- Does not interfere with `scp`/`sftp` or non-interactive SSH commands when set as login shell.

### External References
- Crush (base, MCP, config, logging): [charmbracelet/crush](https://github.com/charmbracelet/crush)
- Optional inspirations (no native MCP):
  - [BuilderIO/ai-shell](https://github.com/BuilderIO/ai-shell)
  - [google-gemini/gemini-cli](https://github.com/google-gemini/gemini-cli)


