package models

import (
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/lacymorrow/lash/internal/tui/components/core"
)

type KeyMap struct {
	Select,
	Next,
	Previous,
	Tab,
	Close key.Binding

	isAPIKeyHelp  bool
	isAPIKeyValid bool
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Select: key.NewBinding(
			key.WithKeys(core.KeyEnter, core.KeyCtrlY),
			key.WithHelp("enter", "confirm"),
		),
		Next: key.NewBinding(
			key.WithKeys(core.KeyDown, core.KeyCtrlN),
			key.WithHelp("↓", "next item"),
		),
		Previous: key.NewBinding(
			key.WithKeys(core.KeyUp, core.KeyCtrlP),
			key.WithHelp("↑", "previous item"),
		),
		Tab: key.NewBinding(
			key.WithKeys(core.KeyTab),
			key.WithHelp("tab", "toggle type"),
		),
		Close: key.NewBinding(
			key.WithKeys(core.KeyEsc),
			key.WithHelp(core.KeyEsc, "cancel"),
		),
	}
}

// KeyBindings implements layout.KeyMapProvider
func (k KeyMap) KeyBindings() []key.Binding {
	return []key.Binding{
		k.Select,
		k.Next,
		k.Previous,
		k.Tab,
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
	if k.isAPIKeyHelp && !k.isAPIKeyValid {
		return []key.Binding{
			k.Close,
		}
	} else if k.isAPIKeyValid {
		return []key.Binding{
			k.Select,
		}
	}
	return []key.Binding{
		key.NewBinding(
			key.WithKeys(core.KeyDown, core.KeyUp),
			key.WithHelp("↑↓", "choose"),
		),
		k.Tab,
		k.Select,
		k.Close,
	}
}
