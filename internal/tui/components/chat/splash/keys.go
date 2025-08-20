package splash

import (
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/lacymorrow/lash/internal/tui/components/core"
)

type KeyMap struct {
	Select,
	Next,
	Previous,
	Yes,
	No,
	Tab,
	LeftRight,
	Back,
	Close key.Binding
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
		Yes: key.NewBinding(
			key.WithKeys(core.KeyY, core.KeyCapitalY),
			key.WithHelp("y/Y", "yes"),
		),
		No: key.NewBinding(
			key.WithKeys(core.KeyN, core.KeyCapitalN),
			key.WithHelp("n/N", "no"),
		),
		Tab: key.NewBinding(
			key.WithKeys(core.KeyTab),
			key.WithHelp("tab", "toggle"),
		),
		LeftRight: key.NewBinding(
			key.WithKeys(core.KeyLeft, core.KeyRight),
			key.WithHelp("←/→", "switch"),
		),
		Back: key.NewBinding(
			key.WithKeys(core.KeyEsc),
			key.WithHelp(core.KeyEsc, "back"),
		),
		Close: key.NewBinding(
			key.WithKeys(core.KeyEsc),
			key.WithHelp(core.KeyEsc, "cancel"),
		),
	}
}
