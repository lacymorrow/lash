package editor

import (
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/lacymorrow/lash/internal/tui/components/core"
)

type EditorKeyMap struct {
	AddFile     key.Binding
	SendMessage key.Binding
	OpenEditor  key.Binding
	Newline     key.Binding
	ClearInput  key.Binding
}

func DefaultEditorKeyMap() EditorKeyMap {
	return EditorKeyMap{
		AddFile: key.NewBinding(
			key.WithKeys(core.KeySlash),
			key.WithHelp(core.KeySlash, "add file"),
		),
		SendMessage: key.NewBinding(
			key.WithKeys(core.KeyEnter),
			key.WithHelp(core.KeyEnter, "send"),
		),
		OpenEditor: key.NewBinding(
			key.WithKeys(core.KeyCtrlO),
			key.WithHelp(core.KeyCtrlO, "open editor"),
		),
		Newline: key.NewBinding(
			key.WithKeys(core.KeyShiftEnter, core.KeyCtrlJ),
			key.WithHelp("shift+enter", "newline"),
		),
		ClearInput: key.NewBinding(
			key.WithKeys(core.KeyCtrlC),
			key.WithHelp(core.KeyCtrlC, "clear input"),
		),
	}
}

// KeyBindings implements layout.KeyMapProvider
func (k EditorKeyMap) KeyBindings() []key.Binding {
	return []key.Binding{
		k.AddFile,
		k.SendMessage,
		k.OpenEditor,
		k.Newline,
		k.ClearInput,
		AttachmentsKeyMaps.AttachmentDeleteMode,
		AttachmentsKeyMaps.DeleteAllAttachments,
		AttachmentsKeyMaps.Escape,
	}
}

type DeleteAttachmentKeyMaps struct {
	AttachmentDeleteMode key.Binding
	Escape               key.Binding
	DeleteAllAttachments key.Binding
}

// TODO: update this to use the new keymap concepts
var AttachmentsKeyMaps = DeleteAttachmentKeyMaps{
	AttachmentDeleteMode: key.NewBinding(
		key.WithKeys(core.KeyCtrlR),
		key.WithHelp("ctrl+r+{i}", "delete attachment at index i"),
	),
	Escape: key.NewBinding(
		key.WithKeys(core.KeyEsc, core.KeyCtrlC),
		key.WithHelp("esc/ctrl+c", "cancel delete mode"),
	),
	DeleteAllAttachments: key.NewBinding(
		key.WithKeys(core.KeyR),
		key.WithHelp("ctrl+r+r", "delete all attachments"),
	),
}
