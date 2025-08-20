package tui

import (
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/lacymorrow/lash/internal/tui/components/core"
)

type KeyMap struct {
	Quit     key.Binding
	QuitEOF  key.Binding
	Help     key.Binding
	Commands key.Binding
	Suspend  key.Binding
	Sessions key.Binding

	// Modes
	ToggleMode key.Binding

	// Safety
	ToggleAutoConfirm key.Binding

	// YOLO
	ToggleYolo key.Binding

	pageBindings []key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys(core.KeyCtrlC),
			key.WithHelp(core.KeyCtrlC, core.HelpQuit),
		),
		QuitEOF: key.NewBinding(
			key.WithKeys(core.KeyCtrlD),
			key.WithHelp(core.KeyCtrlD, core.HelpQuit),
		),
		Help: key.NewBinding(
			key.WithKeys(core.KeyCtrlG),
			key.WithHelp(core.KeyCtrlG, core.HelpMore),
		),
		Commands: key.NewBinding(
			key.WithKeys(core.KeyCtrlP),
			key.WithHelp(core.KeyCtrlP, core.HelpCommands),
		),
		Suspend: key.NewBinding(
			key.WithKeys(core.KeyCtrlZ),
			key.WithHelp(core.KeyCtrlZ, core.HelpSuspend),
		),
		Sessions: key.NewBinding(
			key.WithKeys(core.KeyCtrlS),
			key.WithHelp(core.KeyCtrlS, core.HelpSessions),
		),

		// Single toggle for modes
		ToggleMode: key.NewBinding(
			key.WithKeys(core.KeyCtrlSpace),
			key.WithHelp(core.KeyCtrlSpace, core.HelpMode),
		),

		// Toggle auto-confirm (Ctrl+Y)
		ToggleAutoConfirm: key.NewBinding(
			key.WithKeys(core.KeyCtrlY),
			key.WithHelp(core.KeyCtrlY, core.HelpAutoConfirm),
		),

		// Toggle YOLO (Shift-Tab)
		ToggleYolo: key.NewBinding(
			key.WithKeys(core.KeyShiftTab),
			key.WithHelp(core.KeyShiftTab, core.HelpYolo),
		),
	}
}
