## Lash: Tasks (Fork of Charmbracelet Crush)

### Milestone M0 — Fork & Scaffolding
- T0.1: Fork `charmbracelet/crush`; set module name and licensing/NOTICE updates.
  - Acceptance: `lash --version` prints fork info; license headers preserved.
- T0.2: Add `lash` config namespace to `crush.json` schema and parser.
  - Acceptance: `lash.default_mode` (defaults to `auto`), `lash.keymap`, `lash.real_shell` parsed with defaults; last selected policy is persisted and takes precedence on startup.
- T0.3: Integrate minimal statusline rendering and key hints (no chat UI dependency).
  - Acceptance: Statusline renders; help overlay toggles; PTY height reserved.

### Milestone M1 — Shell-first PTY Pass-through
- T1.1: Implement PTY session to spawn real shell (`$SHELL` or configured) with pass-through.
  - Acceptance: Full TTY features; arrow keys; full-screen apps; echo behaves like a native terminal.
- T1.2: Reserve last row for statusline; propagate resize.
  - Acceptance: PTY sized to rows-1; statusline persists during vim/less/fzf.
- T1.3: Non-interactive guard for no TTY or `-c`.
  - Acceptance: Remote commands over SSH unaffected when set as login shell.

### Milestone M2 — Shell Hook Injection (CNF)
- T2.1: zsh hooks via injected `ZDOTDIR` rc fragments (source user rc first).
  - Acceptance: Capture `preexec`, `precmd`, and `command_not_found_handler`; emit sentinels; no visible artifacts.
- T2.2: bash hooks via injected `--rcfile` (source user rc first).
  - Acceptance: Capture last command/status and `command_not_found_handle`; emit sentinels; no visible artifacts.
- T2.3: Sentinel parser in PTY stream and CNF detection pipeline.
  - Acceptance: Accurate detection with noisy outputs and full-screen apps.

### Milestone M3 — Agent Fallback + Confirm
- T3.1: On CNF, forward original line to agent; render compact response.
  - Acceptance: Explanation-only responses appear in statusline without disrupting the PTY.
- T3.2: Confirmation panel for suggested commands; inject into same PTY on confirm.
  - Acceptance: Ctrl-Enter confirms; Esc cancels; commands run with shell history and completion unaffected.
- T3.3: Explicit agent invocation for next line via hotkey/prefix.
  - Acceptance: Forces agent without shell attempt; terminal view unchanged.

### Milestone M4 — Uniform Routing Policies
- T4.1: Implement mode as routing policy only (Shell / Agent / Auto).
  - Acceptance: Visuals identical across modes; last-selected policy persisted; defaults to Auto.
- T4.2: Keybindings: Ctrl-1/2/3, Ctrl-Enter, Ctrl-/ (configurable via `lash.keymap`).
  - Acceptance: Overrides via `crush.json`.

### Milestone M5 — SSH Interop & Robustness
- T5.1: `lash ssh user@host` convenience delegating to system `ssh` in PTY.
  - Acceptance: Behaves like native terminal; resize propagated; statusline intact.
- T5.2: Logging/redaction: integrate redact patterns; `--debug` flag.
  - Acceptance: Logs rotate; secrets redacted; debug toggles verbosity.
- T5.3: Error handling: MCP failure degrades gracefully; PTY fallback.
  - Acceptance: Killing MCP keeps Shell usable; restart works.

### Milestone M6 — Packaging & Login Shell Docs
- T6.1: GoReleaser targets macOS/Linux (amd64/arm64) with static builds.
  - Acceptance: CI artifacts built; checksums; signed if configured.
- T6.2: Install docs for login shell (`/etc/shells`, `chsh`) and bypass.
  - Acceptance: User can set as login shell; `LASH_DISABLE=1` bypass works.

### Testing
- Unit: config parsing, keymap, router policies, sentinel parsing, redaction.
- Integration: PTY lifecycle, resize, login-shell non-interactive exec, confirmation guard, MCP I/O, CNF detection.
- E2E: expect-like scripts verifying history/completion unaffected; full-screen apps unaffected by statusline; CNF fallback; SSH interop.

### Backlog
- Fish shell hooks; SSH profiles UI; multi-pane/tab support; native MCP client (no subprocess); model/tool selection palette.

### References
- Base + MCP: [charmbracelet/crush](https://github.com/charmbracelet/crush)
- Related agents (no native MCP): [BuilderIO/ai-shell](https://github.com/BuilderIO/ai-shell), [google-gemini/gemini-cli](https://github.com/google-gemini/gemini-cli)


