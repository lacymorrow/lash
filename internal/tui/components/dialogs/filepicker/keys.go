package filepicker

import (
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/lacymorrow/lash/internal/tui/components/core"
)

type KeyMap struct {
	Select,
	Down,
	Up,
	Forward,
	Backward,
	Close key.Binding

	Scroll key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Select: key.NewBinding(
			key.WithKeys(core.KeyEnter),
			key.WithHelp("enter", "select"),
		),
		Down: key.NewBinding(
			key.WithKeys(core.KeyDown, core.KeyJ),
			key.WithHelp("↓", "down"),
		),
		Up: key.NewBinding(
			key.WithKeys(core.KeyUp, core.KeyK),
			key.WithHelp("↑", "up"),
		),
		Forward: key.NewBinding(
			key.WithKeys(core.KeyRight, core.KeyL),
			key.WithHelp("→", "open dir"),
		),
		Backward: key.NewBinding(
			key.WithKeys(core.KeyLeft, core.KeyH),
			key.WithHelp("←", "go back"),
		),
		Close: key.NewBinding(
			key.WithKeys(core.KeyEsc),
			key.WithHelp(core.KeyEsc, "cancel"),
		),
		Scroll: key.NewBinding(
			key.WithKeys(core.KeyRight, core.KeyL, core.KeyLeft, core.KeyH, core.KeyUp, core.KeyK, core.KeyDown, core.KeyJ),
		),
	}
}

// KeyBindings implements layout.KeyMapProvider
func (k KeyMap) KeyBindings() []key.Binding {
	return []key.Binding{
		k.Select,
		k.Down,
		k.Up,
		k.Forward,
		k.Backward,
		k.Close,
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
		k.Forward,
		k.Backward,
		k.Up,
		k.Down,
		k.Select,
		k.Close,
	}
}
