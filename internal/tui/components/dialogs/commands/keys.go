package commands

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

	Confirm     key.Binding
	NextSection key.Binding
	PrevSection key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Select: key.NewBinding(
			key.WithKeys(core.KeyEnter, core.KeyCtrlY),
			key.WithHelp("enter", "confirm"),
		),
		Next: key.NewBinding(
			key.WithKeys(core.KeyDown, core.KeyCtrlN),
			key.WithHelp("↓", "next"),
		),
		Previous: key.NewBinding(
			key.WithKeys(core.KeyUp, core.KeyCtrlP),
			key.WithHelp("↑", "previous"),
		),
		Tab: key.NewBinding(
			key.WithKeys(core.KeyTab),
			key.WithHelp("tab", "move"),
		),
		Close: key.NewBinding(
			key.WithKeys(core.KeyEsc),
			key.WithHelp(core.KeyEsc, "cancel"),
		),
		Confirm: key.NewBinding(
			key.WithKeys(core.KeyEnter),
		),
		NextSection: key.NewBinding(
			key.WithKeys(core.KeyTab, core.KeyDown),
		),
		PrevSection: key.NewBinding(
			key.WithKeys(core.KeyShiftTab, core.KeyUp),
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
	return []key.Binding{
		k.Tab,
		key.NewBinding(
			key.WithKeys(core.KeyDown, core.KeyUp),
			key.WithHelp("↑↓", "choose"),
		),
		k.Select,
		k.Close,
	}
}

type ArgumentsDialogKeyMap struct {
	Confirm  key.Binding
	Next     key.Binding
	Previous key.Binding
}

func DefaultArgumentsDialogKeyMap() ArgumentsDialogKeyMap {
	return ArgumentsDialogKeyMap{
		Confirm: key.NewBinding(
			key.WithKeys(core.KeyEnter),
			key.WithHelp("enter", "confirm"),
		),

		Next: key.NewBinding(
			key.WithKeys(core.KeyTab, core.KeyDown),
			key.WithHelp("tab/↓", "next"),
		),
		Previous: key.NewBinding(
			key.WithKeys(core.KeyShiftTab, core.KeyUp),
			key.WithHelp("shift+tab/↑", "previous"),
		),
	}
}

// KeyBindings implements layout.KeyMapProvider
func (k ArgumentsDialogKeyMap) KeyBindings() []key.Binding {
	return []key.Binding{
		k.Confirm,
		k.Next,
		k.Previous,
	}
}

// FullHelp implements help.KeyMap.
func (k ArgumentsDialogKeyMap) FullHelp() [][]key.Binding {
	m := [][]key.Binding{}
	slice := k.KeyBindings()
	for i := 0; i < len(slice); i += 4 {
		end := min(i+4, len(slice))
		m = append(m, slice[i:end])
	}
	return m
}

// ShortHelp implements help.KeyMap.
func (k ArgumentsDialogKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Confirm,
		k.Next,
		k.Previous,
	}
}
