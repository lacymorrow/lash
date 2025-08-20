package completions

import (
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/lacymorrow/lash/internal/tui/components/core"
)

type KeyMap struct {
	Down,
	Up,
	Select,
	Cancel,
	Next,
	Previous,
	UpInsert,
	DownInsert key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Down: key.NewBinding(
			key.WithKeys(core.KeyDown),
			key.WithHelp("↓", "down"),
		),
		Up: key.NewBinding(
			key.WithKeys(core.KeyUp),
			key.WithHelp("↑", "up"),
		),
		Select: key.NewBinding(
			key.WithKeys(core.KeyEnter, core.KeyTab, core.KeyCtrlY),
			key.WithHelp("enter/tab", "select"),
		),
		Cancel: key.NewBinding(
			key.WithKeys(core.KeyEsc),
			key.WithHelp(core.KeyEsc, "cancel"),
		),
		Next: key.NewBinding(
			key.WithKeys(core.KeyCtrlN),
		),
		Previous: key.NewBinding(
			key.WithKeys(core.KeyCtrlP),
		),
		UpInsert: key.NewBinding(
			key.WithKeys(core.KeyShiftUp),
		),
		DownInsert: key.NewBinding(
			key.WithKeys(core.KeyShiftDown),
		),
	}
}

// KeyBindings implements layout.KeyMapProvider
func (k KeyMap) KeyBindings() []key.Binding {
	return []key.Binding{
		k.Down,
		k.Up,
		k.Select,
		k.Cancel,
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
		k.Up,
		k.Down,
	}
}
