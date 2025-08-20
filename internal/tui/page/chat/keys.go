package chat

import (
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/lacymorrow/lash/internal/tui/components/core"
)

type KeyMap struct {
	NewSession    key.Binding
	AddAttachment key.Binding
	Cancel        key.Binding
	Tab           key.Binding
	Details       key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		NewSession: key.NewBinding(
			key.WithKeys(core.KeyCtrlN),
			key.WithHelp(core.KeyCtrlN, "new session"),
		),
		AddAttachment: key.NewBinding(
			key.WithKeys(core.KeyCtrlF),
			key.WithHelp(core.KeyCtrlF, "add attachment"),
		),
		Cancel: key.NewBinding(
			key.WithKeys(core.KeyEsc),
			key.WithHelp(core.KeyEsc, "cancel"),
		),
		Tab: key.NewBinding(
			key.WithKeys(core.KeyTab),
			key.WithHelp(core.KeyTab, "change focus"),
		),
		Details: key.NewBinding(
			key.WithKeys(core.KeyCtrlE),
			key.WithHelp(core.KeyCtrlE, "toggle details"),
		),
	}
}
