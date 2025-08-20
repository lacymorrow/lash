package list

import (
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/lacymorrow/lash/internal/tui/components/core"
)

type KeyMap struct {
	Down,
	Up,
	DownOneItem,
	UpOneItem,
	PageDown,
	PageUp,
	HalfPageDown,
	HalfPageUp,
	Home,
	End key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Down: key.NewBinding(
			key.WithKeys(core.KeyDown, core.KeyCtrlJ, core.KeyCtrlN, core.KeyJ),
			key.WithHelp("↓", "down"),
		),
		Up: key.NewBinding(
			key.WithKeys(core.KeyUp, "ctrl+k", core.KeyCtrlP, core.KeyK),
			key.WithHelp("↑", "up"),
		),
		UpOneItem: key.NewBinding(
			key.WithKeys(core.KeyShiftUp, core.KeyCapitalK),
			key.WithHelp("shift+↑", "up one item"),
		),
		DownOneItem: key.NewBinding(
			key.WithKeys(core.KeyShiftDown, core.KeyCapitalJ),
			key.WithHelp("shift+↓", "down one item"),
		),
		HalfPageDown: key.NewBinding(
			key.WithKeys(core.KeyD),
			key.WithHelp("d", "half page down"),
		),
		PageDown: key.NewBinding(
			key.WithKeys(core.KeyPageDown, core.KeySpace, core.KeyF),
			key.WithHelp("f/pgdn", "page down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys(core.KeyPageUp, core.KeyB),
			key.WithHelp("b/pgup", "page up"),
		),
		HalfPageUp: key.NewBinding(
			key.WithKeys(core.KeyU),
			key.WithHelp("u", "half page up"),
		),
		Home: key.NewBinding(
			key.WithKeys(core.KeyG, core.KeyHome),
			key.WithHelp("g", "home"),
		),
		End: key.NewBinding(
			key.WithKeys(core.KeyCapitalG, core.KeyEnd),
			key.WithHelp("G", "end"),
		),
	}
}

func (k KeyMap) KeyBindings() []key.Binding {
	return []key.Binding{
		k.Down,
		k.Up,
		k.DownOneItem,
		k.UpOneItem,
		k.HalfPageDown,
		k.HalfPageUp,
		k.Home,
		k.End,
	}
}
