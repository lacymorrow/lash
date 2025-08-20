package permissions

import (
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/lacymorrow/lash/internal/tui/components/core"
)

type KeyMap struct {
	Left,
	Right,
	Tab,
	Select,
	Allow,
	AllowSession,
	Deny,
	ToggleDiffMode,
	ScrollDown,
	ScrollUp key.Binding
	ScrollLeft,
	ScrollRight key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Left: key.NewBinding(
			key.WithKeys(core.KeyLeft, core.KeyH),
			key.WithHelp("←", "previous"),
		),
		Right: key.NewBinding(
			key.WithKeys(core.KeyRight, core.KeyL),
			key.WithHelp("→", "next"),
		),
		Tab: key.NewBinding(
			key.WithKeys(core.KeyTab),
			key.WithHelp("tab", "switch"),
		),
		Allow: key.NewBinding(
			key.WithKeys(core.KeyA, core.KeyCapitalA, "ctrl+a"),
			key.WithHelp("a", "allow"),
		),
		AllowSession: key.NewBinding(
			key.WithKeys(core.KeyS, core.KeyCapitalS, core.KeyCtrlS),
			key.WithHelp("s", "allow session"),
		),
		Deny: key.NewBinding(
			key.WithKeys(core.KeyD, "D", core.KeyEsc),
			key.WithHelp("d", "deny"),
		),
		Select: key.NewBinding(
			key.WithKeys(core.KeyEnter, core.KeyCtrlY),
			key.WithHelp("enter", "confirm"),
		),
		ToggleDiffMode: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "toggle diff mode"),
		),
		ScrollDown: key.NewBinding(
			key.WithKeys(core.KeyShiftDown, core.KeyCapitalJ),
			key.WithHelp("shift+↓", "scroll down"),
		),
		ScrollUp: key.NewBinding(
			key.WithKeys(core.KeyShiftUp, core.KeyCapitalK),
			key.WithHelp("shift+↑", "scroll up"),
		),
		ScrollLeft: key.NewBinding(
			key.WithKeys(core.KeyShiftLeft, core.KeyCapitalH),
			key.WithHelp("shift+←", "scroll left"),
		),
		ScrollRight: key.NewBinding(
			key.WithKeys(core.KeyShiftRight, core.KeyCapitalL),
			key.WithHelp("shift+→", "scroll right"),
		),
	}
}

// KeyBindings implements layout.KeyMapProvider
func (k KeyMap) KeyBindings() []key.Binding {
	return []key.Binding{
		k.Left,
		k.Right,
		k.Tab,
		k.Select,
		k.Allow,
		k.AllowSession,
		k.Deny,
		k.ToggleDiffMode,
		k.ScrollDown,
		k.ScrollUp,
		k.ScrollLeft,
		k.ScrollRight,
	}
}

// FullHelp implements help.KeyMap.
func (k KeyMap) FullHelp() [][]key.Binding {
	m := [][]key.Binding{}
	slice := k.KeyBindings()
	for i := 0; i < len(slice); i += 4 {
		end := min(i+4, len(slice))
		m = append(m, slice[i:end])
	}
	return m
}

// ShortHelp implements help.KeyMap.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.ToggleDiffMode,
		key.NewBinding(
			key.WithKeys(core.KeyShiftLeft, core.KeyShiftDown, core.KeyShiftUp, core.KeyShiftRight),
			key.WithHelp("shift+←↓↑→", "scroll"),
		),
	}
}
