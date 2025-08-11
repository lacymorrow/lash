# Lash zsh plugin: agent fallback and mode control
# Source this file from .zshrc or install via `install.sh`.
if [ -z "${ZSH_VERSION:-}" ]; then
  echo "lash.plugin.zsh: must be sourced in zsh" 1>&2
  return 0
fi

typeset -g LASH_MODE="auto"  # shell | agent | auto

# Print a tiny mode suffix in RPROMPT
_lash_set_prompt() {
  typeset m
  m="$LASH_MODE"
  RPROMPT="%F{242}${m}%f"
}
autoload -Uz add-zsh-hook
add-zsh-hook precmd _lash_set_prompt

# Force agent on current buffer (regardless of CNF)
lash-force-agent() {
  emulate -L zsh
  typeset prompt
  prompt="$BUFFER"
  if [[ -z "$prompt" ]]; then
    zle -M "(empty)"
    return 0
  fi
  typeset out
  out=$(lash-agent --cwd "$PWD" -- $prompt 2>/dev/null)
  typeset suggestion explanation
  suggestion=$(print -r -- "$out" | grep '^SUGGEST ' | sed 's/^SUGGEST //')
  explanation=$(print -r -- "$out" | grep '^EXPLAIN ' | sed 's/^EXPLAIN //')
  if [[ -n "$suggestion" ]]; then
    BUFFER="$suggestion"
    CURSOR=${#BUFFER}
    if [[ -n "$explanation" ]]; then
      zle -M "$explanation"
    fi
  else
    # In agent mode, show explanation; in shell mode, leave default CNF
    if [[ "$LASH_MODE" == "agent" || "$LASH_MODE" == "auto" ]]; then
      if [[ -n "$explanation" ]]; then
        zle -M "$explanation"
      else
        zle -M "(no suggestion)"
      fi
    fi
  fi
}
zle -N lash-force-agent

# Mode switching widgets
lash-mode-shell() { LASH_MODE="shell"; zle -M "mode: shell"; }
lash-mode-agent() { LASH_MODE="agent"; zle -M "mode: agent"; }
lash-mode-auto()  { LASH_MODE="auto";  zle -M "mode: auto"; }
zle -N lash-mode-shell
zle -N lash-mode-agent
zle -N lash-mode-auto

# Command-not-found fallback in auto mode
command_not_found_handler() {
  emulate -L zsh
  if [[ "$LASH_MODE" != "auto" ]]; then
    return 127
  fi
  typeset original out suggestion explanation
  original="$1"
  out=$(lash-agent --cwd "$PWD" -- "$original" 2>/dev/null)
  suggestion=$(print -r -- "$out" | grep '^SUGGEST ' | sed 's/^SUGGEST //')
  explanation=$(print -r -- "$out" | grep '^EXPLAIN ' | sed 's/^EXPLAIN //')
  if [[ -n "$suggestion" ]]; then
    BUFFER="$suggestion"
    CURSOR=${#BUFFER}
    if [[ -n "$explanation" ]]; then
      zle -M "$explanation"
    fi
    return 0
  fi
  # If no suggestion, still surface an explanation line when present
  if [[ -n "$explanation" ]]; then
    print -r -- "$explanation"
  fi
  return 127
}

# Bindings: Control-based (works reliably across terminals)
# Emacs keymap
bindkey -M emacs '^M' .accept-line                     # Enter
bindkey -M emacs '^X^A' lash-force-agent               # Ctrl-x Ctrl-a -> force agent
bindkey -M emacs '^Xa'  lash-force-agent               # Ctrl-x a     -> force agent
bindkey -M emacs '^X1'  lash-mode-shell                # Ctrl-x 1     -> Shell mode
bindkey -M emacs '^X2'  lash-mode-agent                # Ctrl-x 2     -> Agent mode
bindkey -M emacs '^X3'  lash-mode-auto                 # Ctrl-x 3     -> Auto mode

# Vi insert keymap (if using vi-mode)
bindkey -M viins '^M' .accept-line
bindkey -M viins '^X^A' lash-force-agent
bindkey -M viins '^Xa'  lash-force-agent
bindkey -M viins '^X1'  lash-mode-shell
bindkey -M viins '^X2'  lash-mode-agent
bindkey -M viins '^X3'  lash-mode-auto


