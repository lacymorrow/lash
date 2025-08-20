package editor

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/textarea"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/lacymorrow/lash/internal/app"
	"github.com/lacymorrow/lash/internal/fsext"
	"github.com/lacymorrow/lash/internal/message"
	"github.com/lacymorrow/lash/internal/session"
	"github.com/lacymorrow/lash/internal/shell"
	"github.com/lacymorrow/lash/internal/tui/components/chat"
	"github.com/lacymorrow/lash/internal/tui/components/completions"
	"github.com/lacymorrow/lash/internal/tui/components/core/layout"
	"github.com/lacymorrow/lash/internal/tui/components/dialogs"
	"github.com/lacymorrow/lash/internal/tui/components/dialogs/commands"
	"github.com/lacymorrow/lash/internal/tui/components/dialogs/filepicker"
	"github.com/lacymorrow/lash/internal/tui/components/dialogs/quit"
	"github.com/lacymorrow/lash/internal/tui/styles"
	"github.com/lacymorrow/lash/internal/tui/util"
)

type Editor interface {
	util.Model
	layout.Sizeable
	layout.Focusable
	layout.Help
	layout.Positional

	SetSession(session session.Session) tea.Cmd
	IsCompletionsOpen() bool
	HasAttachments() bool
	Cursor() *tea.Cursor
}

type FileCompletionItem struct {
	Path string // The file path
}

type editorCmp struct {
	width              int
	height             int
	x, y               int
	app                *app.App
	session            session.Session
	textarea           *textarea.Model
	attachments        []message.Attachment
	deleteMode         bool
	readyPlaceholder   string
	workingPlaceholder string

	keyMap EditorKeyMap

	// File path completions
	currentQuery          string
	completionsStartIndex int
	isCompletionsOpen     bool

	// Per-session input history
	inputHistory map[string][]string // sessionID -> history entries (most recent at end)
	historyIndex int                 // current index into history for active session, -1 means not in history selection
	historyTemp  string              // the current unsent input before entering history navigation
	inHistoryNav bool                // whether we are actively navigating history

	// When the first submission happens before a session exists, we temporarily
	// store that entry and attach it to the session's history once the session
	// is created and SetSession is called.
	pendingFirstHistoryEntry string
}

var DeleteKeyMaps = DeleteAttachmentKeyMaps{
	AttachmentDeleteMode: key.NewBinding(
		key.WithKeys("ctrl+r"),
		key.WithHelp("ctrl+r+{i}", "delete attachment at index i"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel delete mode"),
	),
	DeleteAllAttachments: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("ctrl+r+r", "delete all attachments"),
	),
}

const (
	maxAttachments      = 5
	maxSlashCompletions = 500
)

type OpenEditorMsg struct {
	Text string
}

func (m *editorCmp) openEditor(value string) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		// Use platform-appropriate default editor
		if runtime.GOOS == "windows" {
			editor = "notepad"
		} else {
			editor = "nvim"
		}
	}

	tmpfile, err := os.CreateTemp("", "msg_*.md")
	if err != nil {
		return util.ReportError(err)
	}
	defer tmpfile.Close() //nolint:errcheck
	if _, err := tmpfile.WriteString(value); err != nil {
		return util.ReportError(err)
	}
	c := exec.CommandContext(context.TODO(), editor, tmpfile.Name())
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return tea.ExecProcess(c, func(err error) tea.Msg {
		if err != nil {
			return util.ReportError(err)
		}
		content, err := os.ReadFile(tmpfile.Name())
		if err != nil {
			return util.ReportError(err)
		}
		if len(content) == 0 {
			return util.ReportWarn("Message is empty")
		}
		os.Remove(tmpfile.Name())
		return OpenEditorMsg{
			Text: strings.TrimSpace(string(content)),
		}
	})
}

func (m *editorCmp) Init() tea.Cmd {
	return nil
}

func (m *editorCmp) send() tea.Cmd {
	if m.app.CoderAgent == nil {
		return util.ReportError(fmt.Errorf("coder agent is not initialized"))
	}
	if m.app.CoderAgent.IsSessionBusy(m.session.ID) {
		return util.ReportWarn("Agent is working, please wait...")
	}

	value := m.textarea.Value()
	value = strings.TrimSpace(value)

	switch value {
	case "exit", "quit":
		m.textarea.Reset()
		return util.CmdHandler(dialogs.OpenDialogMsg{Model: quit.NewQuitDialog()})
	}

	// Append to global input history for cross-session navigation.
	// Also keep existing per-session stashing behavior for backward compatibility.
	if value != "" {
		// Global history (app-wide, persisted)
		_ = m.app.AppendInputHistory(value)

		// Existing per-session history behavior
		if m.session.ID == "" {
			// Defer attaching to a specific session ID until it's created.
			m.pendingFirstHistoryEntry = value
		} else {
			if m.inputHistory == nil {
				m.inputHistory = make(map[string][]string)
			}
			h := m.inputHistory[m.session.ID]
			if len(h) == 0 || h[len(h)-1] != value {
				m.inputHistory[m.session.ID] = append(h, value)
			}
		}
		// Reset history navigation state after sending
		m.inHistoryNav = false
		m.historyIndex = -1
		m.historyTemp = ""
	}

	m.textarea.Reset()
	attachments := m.attachments

	m.attachments = nil
	if value == "" {
		return nil
	}

	// Change the placeholder when sending a new message.
	m.randomizePlaceholders()

	return tea.Batch(
		util.CmdHandler(chat.SendMsg{
			Text:        value,
			Attachments: attachments,
		}),
	)
}

func (m *editorCmp) repositionCompletions() tea.Msg {
	x, y := m.completionsPosition()
	return completions.RepositionCompletionsMsg{X: x, Y: y}
}

func (m *editorCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m, m.repositionCompletions
	case filepicker.FilePickedMsg:
		if len(m.attachments) >= maxAttachments {
			return m, util.ReportError(fmt.Errorf("cannot add more than %d images", maxAttachments))
		}
		m.attachments = append(m.attachments, msg.Attachment)
		return m, nil
	case completions.CompletionsOpenedMsg:
		m.isCompletionsOpen = true
	case completions.CompletionsClosedMsg:
		m.isCompletionsOpen = false
		m.currentQuery = ""
		m.completionsStartIndex = 0
	case completions.SelectCompletionMsg:
		if !m.isCompletionsOpen {
			return m, nil
		}
		if item, ok := msg.Value.(FileCompletionItem); ok {
			word := m.textarea.Word()
			// If the selected item is a file, insert its path into the textarea
			value := m.textarea.Value()
			value = value[:m.completionsStartIndex] + // Remove the current query
				item.Path + // Insert the file path
				value[m.completionsStartIndex+len(word):] // Append the rest of the value
			// XXX: This will always move the cursor to the end of the textarea.
			m.textarea.SetValue(value)
			m.textarea.MoveToEnd()
			if !msg.Insert {
				m.isCompletionsOpen = false
				m.currentQuery = ""
				m.completionsStartIndex = 0
			}
		}

	case commands.OpenExternalEditorMsg:
		if m.app.CoderAgent.IsSessionBusy(m.session.ID) {
			return m, util.ReportWarn("Agent is working, please wait...")
		}
		return m, m.openEditor(m.textarea.Value())
	case OpenEditorMsg:
		m.textarea.SetValue(msg.Text)
		m.textarea.MoveToEnd()
	case tea.PasteMsg:
		path := strings.ReplaceAll(string(msg), "\\ ", " ")
		// try to get an image
		path, err := filepath.Abs(strings.TrimSpace(path))
		if err != nil {
			m.textarea, cmd = m.textarea.Update(msg)
			return m, cmd
		}
		isAllowedType := false
		for _, ext := range filepicker.AllowedTypes {
			if strings.HasSuffix(path, ext) {
				isAllowedType = true
				break
			}
		}
		if !isAllowedType {
			m.textarea, cmd = m.textarea.Update(msg)
			return m, cmd
		}
		tooBig, _ := filepicker.IsFileTooBig(path, filepicker.MaxAttachmentSize)
		if tooBig {
			m.textarea, cmd = m.textarea.Update(msg)
			return m, cmd
		}

		content, err := os.ReadFile(path)
		if err != nil {
			m.textarea, cmd = m.textarea.Update(msg)
			return m, cmd
		}
		mimeBufferSize := min(512, len(content))
		mimeType := http.DetectContentType(content[:mimeBufferSize])
		fileName := filepath.Base(path)
		attachment := message.Attachment{FilePath: path, FileName: fileName, MimeType: mimeType, Content: content}
		return m, util.CmdHandler(filepicker.FilePickedMsg{
			Attachment: attachment,
		})

	case commands.ToggleYoloModeMsg:
		m.setEditorPrompt()
		return m, nil
	case tea.KeyPressMsg:
		cur := m.textarea.Cursor()
		curIdx := m.textarea.Width()*cur.Y + cur.X
		switch {
		// Completions
		case msg.String() == "/" && !m.isCompletionsOpen &&
			// only show if beginning of prompt, or if previous char is a space or newline:
			(len(m.textarea.Value()) == 0 || unicode.IsSpace(rune(m.textarea.Value()[len(m.textarea.Value())-1]))):
			m.isCompletionsOpen = true
			m.currentQuery = ""
			m.completionsStartIndex = curIdx
			cmds = append(cmds, m.startCompletions)
		case m.isCompletionsOpen && curIdx <= m.completionsStartIndex:
			cmds = append(cmds, util.CmdHandler(completions.CloseCompletionsMsg{}))
		}

		// Clear input: ctrl+c
		if key.Matches(msg, m.keyMap.ClearInput) && m.textarea.Focused() {
			if strings.TrimSpace(m.textarea.Value()) != "" {
				m.textarea.Reset()
				// consume event so app doesn't open quit dialog by returning a non-nil no-op command
				return m, func() tea.Msg { return nil }
			}
			// empty input: open quit dialog (handled here to avoid global help flicker)
			return m, util.CmdHandler(dialogs.OpenDialogMsg{Model: quit.NewQuitDialog()})
		}

		// Platform-aware deletion shortcuts
		if m.textarea.Focused() {
			s := msg.String()
			// macOS: cmd+backspace deletes to start of line; option+backspace deletes previous word
			if runtime.GOOS == "darwin" && (s == "cmd+backspace" || s == "alt+backspace") {
				val := m.textarea.Value()
				cur := m.textarea.Cursor()
				if cur != nil {
					// Approximate linear index from visual cursor position
					idx := m.textarea.Width()*cur.Y + cur.X
					if idx > len(val) {
						idx = len(val)
					} else if idx < 0 {
						idx = 0
					}
					if s == "cmd+backspace" {
						// Delete from start of current line to cursor
						start := strings.LastIndex(val[:idx], "\n") + 1
						if start < 0 {
							start = 0
						}
						newVal := val[:start] + val[idx:]
						if newVal != val {
							m.textarea.SetValue(newVal)
							return m, nil
						}
					} else {
						// alt+backspace: delete previous word
						i := idx
						for i > 0 && unicode.IsSpace(rune(val[i-1])) {
							i--
						}
						for i > 0 && !unicode.IsSpace(rune(val[i-1])) {
							i--
						}
						if i < idx {
							newVal := val[:i] + val[idx:]
							m.textarea.SetValue(newVal)
							return m, nil
						}
					}
				}
			}
			// Windows/Linux: ctrl+backspace deletes to start of line
			if runtime.GOOS != "darwin" && s == "ctrl+backspace" {
				val := m.textarea.Value()
				cur := m.textarea.Cursor()
				if cur != nil {
					idx := m.textarea.Width()*cur.Y + cur.X
					if idx > len(val) {
						idx = len(val)
					} else if idx < 0 {
						idx = 0
					}
					start := strings.LastIndex(val[:idx], "\n") + 1
					if start < 0 {
						start = 0
					}
					newVal := val[:start] + val[idx:]
					if newVal != val {
						m.textarea.SetValue(newVal)
						return m, nil
					}
				}
			}
		}

		// History navigation: Up/Down (global, across sessions)
		if msg.String() == "up" || msg.String() == "down" {
			// Only handle when focused and not deleting attachments and not showing completions
			if m.textarea.Focused() && !m.deleteMode && !m.isCompletionsOpen {
				history := m.app.InputHistory
				if len(history) > 0 {
					if msg.String() == "up" {
						// Enter history nav only if at the top line to avoid interfering with multi-line editing
						if !m.inHistoryNav && cur != nil && cur.Y == 0 {
							m.inHistoryNav = true
							m.historyTemp = m.textarea.Value()
							m.historyIndex = len(history)
						}
						if m.inHistoryNav {
							if m.historyIndex > 0 {
								m.historyIndex--
							}
							m.textarea.SetValue(history[m.historyIndex])
							m.textarea.MoveToEnd()
							return m, nil
						}
					} else if msg.String() == "down" {
						if m.inHistoryNav {
							if m.historyIndex < len(history)-1 {
								m.historyIndex++
								m.textarea.SetValue(history[m.historyIndex])
							} else {
								// Exit history navigation and restore the temp content
								m.inHistoryNav = false
								m.historyIndex = -1
								m.textarea.SetValue(m.historyTemp)
								m.historyTemp = ""
							}
							m.textarea.MoveToEnd()
							return m, nil
						}
					}
				}
			}
		}

		if key.Matches(msg, DeleteKeyMaps.AttachmentDeleteMode) {
			m.deleteMode = true
			return m, nil
		}
		if key.Matches(msg, DeleteKeyMaps.DeleteAllAttachments) && m.deleteMode {
			m.deleteMode = false
			m.attachments = nil
			return m, nil
		}
		rune := msg.Code
		if m.deleteMode && unicode.IsDigit(rune) {
			num := int(rune - '0')
			m.deleteMode = false
			if num < 10 && len(m.attachments) > num {
				if num == 0 {
					m.attachments = m.attachments[num+1:]
				} else {
					m.attachments = slices.Delete(m.attachments, num, num+1)
				}
				return m, nil
			}
		}
		if key.Matches(msg, m.keyMap.OpenEditor) {
			if m.app.CoderAgent.IsSessionBusy(m.session.ID) {
				return m, util.ReportWarn("Agent is working, please wait...")
			}
			return m, m.openEditor(m.textarea.Value())
		}
		if key.Matches(msg, DeleteKeyMaps.Escape) {
			m.deleteMode = false
			return m, nil
		}
		if key.Matches(msg, m.keyMap.Newline) {
			m.textarea.InsertRune('\n')
			cmds = append(cmds, util.CmdHandler(completions.CloseCompletionsMsg{}))
		}
		// Handle Enter key
		if m.textarea.Focused() && key.Matches(msg, m.keyMap.SendMessage) {
			trimmed := strings.TrimSpace(m.textarea.Value())
			// If no active session and input is empty, open recent sessions dialog
			if trimmed == "" && m.session.ID == "" {
				return m, util.CmdHandler(commands.SwitchSessionsMsg{})
			}
			value := m.textarea.Value()
			if len(value) > 0 && value[len(value)-1] == '\\' {
				// If the last character is a backslash, remove it and add a newline
				m.textarea.SetValue(value[:len(value)-1])
			} else {
				// Otherwise, send the message
				return m, m.send()
			}
		}
	}

	m.textarea, cmd = m.textarea.Update(msg)
	cmds = append(cmds, cmd)

	if m.textarea.Focused() {
		kp, ok := msg.(tea.KeyPressMsg)
		if ok {
			if kp.String() == "space" || m.textarea.Value() == "" {
				m.isCompletionsOpen = false
				m.currentQuery = ""
				m.completionsStartIndex = 0
				cmds = append(cmds, util.CmdHandler(completions.CloseCompletionsMsg{}))
			} else {
				word := m.textarea.Word()
				if strings.HasPrefix(word, "/") {
					// XXX: wont' work if editing in the middle of the field.
					m.completionsStartIndex = strings.LastIndex(m.textarea.Value(), word)
					m.currentQuery = word[1:]
					x, y := m.completionsPosition()
					x -= len(m.currentQuery)
					m.isCompletionsOpen = true
					cmds = append(cmds,
						util.CmdHandler(completions.FilterCompletionsMsg{
							Query:  m.currentQuery,
							Reopen: m.isCompletionsOpen,
							X:      x,
							Y:      y,
						}),
					)
				} else if m.isCompletionsOpen {
					m.isCompletionsOpen = false
					m.currentQuery = ""
					m.completionsStartIndex = 0
					cmds = append(cmds, util.CmdHandler(completions.CloseCompletionsMsg{}))
				}
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *editorCmp) setEditorPrompt() {
	if m.app.Permissions.SkipRequests() {
		m.textarea.SetPromptFunc(4, m.yoloPromptFunc)
		return
	}
	m.textarea.SetPromptFunc(4, m.normalPromptFunc)
}

func (m *editorCmp) completionsPosition() (int, int) {
	cur := m.textarea.Cursor()
	if cur == nil {
		return m.x, m.y + 1 // adjust for padding
	}
	x := cur.X + m.x
	y := cur.Y + m.y + 1 // adjust for padding
	return x, y
}

func (m *editorCmp) Cursor() *tea.Cursor {
	cursor := m.textarea.Cursor()
	if cursor != nil {
		cursor.X = cursor.X + m.x + 1
		cursor.Y = cursor.Y + m.y + 1 // adjust for padding
	}
	return cursor
}

var readyPlaceholders = [...]string{
	"Ready!",
	"Ready...",
	"Ready?",
	"Ready for instructions",
}

var workingPlaceholders = [...]string{
	"Working!",
	"Working...",
	"Brrrrr...",
	"Prrrrrrrr...",
	"Processing...",
	"Thinking...",
}

func (m *editorCmp) randomizePlaceholders() {
	m.workingPlaceholder = workingPlaceholders[rand.Intn(len(workingPlaceholders))]
	m.readyPlaceholder = readyPlaceholders[rand.Intn(len(readyPlaceholders))]
}

func (m *editorCmp) View() string {
	t := styles.CurrentTheme()
	// Update placeholder
	if m.app.CoderAgent != nil && m.app.CoderAgent.IsBusy() {
		m.textarea.Placeholder = m.workingPlaceholder
	} else {
		m.textarea.Placeholder = m.readyPlaceholder
	}
	if m.app.Permissions.SkipRequests() {
		m.textarea.Placeholder = "Yolo mode!"
	}
	// Shell-like PWD near the input
	liveCwd := shell.GetUserPersistentShell(m.app.Config().WorkingDir()).GetWorkingDir()
	if liveCwd == "" {
		liveCwd = m.app.Config().WorkingDir()
	}
	cwd := fsext.DirTrim(fsext.PrettyPath(liveCwd), 4)
	cwdView := t.S().Base.Foreground(t.Secondary).Render(cwd)
	// If no active session and input is empty, hint about sessions
	if m.session.ID == "" && strings.TrimSpace(m.textarea.Value()) == "" {
		m.textarea.Placeholder = " Press Enter for recent sessions"
	}
	if len(m.attachments) == 0 {
		content := t.S().Base.Padding(0, 1, 1, 1).Render(
			lipgloss.JoinVertical(lipgloss.Left,
				cwdView,
				m.textarea.View(),
			),
		)
		return content
	}
	content := t.S().Base.Padding(0, 1, 1, 1).Render(
		lipgloss.JoinVertical(lipgloss.Top,
			cwdView,
			m.attachmentsContent(),
			m.textarea.View(),
		),
	)
	return content
}

func (m *editorCmp) SetSize(width, height int) tea.Cmd {
	m.width = width
	m.height = height
	m.textarea.SetWidth(width - 2)   // adjust for padding
	m.textarea.SetHeight(height - 2) // adjust for padding
	return nil
}

func (m *editorCmp) GetSize() (int, int) {
	return m.textarea.Width(), m.textarea.Height()
}

func (m *editorCmp) attachmentsContent() string {
	var styledAttachments []string
	t := styles.CurrentTheme()
	attachmentStyles := t.S().Base.
		MarginLeft(1).
		Background(t.FgMuted).
		Foreground(t.FgBase)
	for i, attachment := range m.attachments {
		var filename string
		if len(attachment.FileName) > 10 {
			filename = fmt.Sprintf(" %s %s...", styles.DocumentIcon, attachment.FileName[0:7])
		} else {
			filename = fmt.Sprintf(" %s %s", styles.DocumentIcon, attachment.FileName)
		}
		if m.deleteMode {
			filename = fmt.Sprintf("%d%s", i, filename)
		}
		styledAttachments = append(styledAttachments, attachmentStyles.Render(filename))
	}
	content := lipgloss.JoinHorizontal(lipgloss.Left, styledAttachments...)
	return content
}

func (m *editorCmp) SetPosition(x, y int) tea.Cmd {
	m.x = x
	m.y = y
	return nil
}

func (m *editorCmp) startCompletions() tea.Msg {
	files, _, _ := fsext.ListDirectory(".", nil, maxSlashCompletions)
	completionItems := make([]completions.Completion, 0, len(files))
	for _, file := range files {
		file = strings.TrimPrefix(file, "./")
		completionItems = append(completionItems, completions.Completion{
			Title: file,
			Value: FileCompletionItem{
				Path: file,
			},
		})
	}

	x, y := m.completionsPosition()
	return completions.OpenCompletionsMsg{
		Completions: completionItems,
		X:           x,
		Y:           y,
	}
}

// Blur implements Container.
func (c *editorCmp) Blur() tea.Cmd {
	c.textarea.Blur()
	return nil
}

// Focus implements Container.
func (c *editorCmp) Focus() tea.Cmd {
	return c.textarea.Focus()
}

// IsFocused implements Container.
func (c *editorCmp) IsFocused() bool {
	return c.textarea.Focused()
}

// Bindings implements Container.
func (c *editorCmp) Bindings() []key.Binding {
	return c.keyMap.KeyBindings()
}

// TODO: most likely we do not need to have the session here
// we need to move some functionality to the page level
func (c *editorCmp) SetSession(session session.Session) tea.Cmd {
	c.session = session
	// Reset transient history navigation state on session switch
	c.inHistoryNav = false
	c.historyIndex = -1
	c.historyTemp = ""
	// If we have a first entry typed before the session existed, attach it now.
	if c.pendingFirstHistoryEntry != "" {
		if c.inputHistory == nil {
			c.inputHistory = make(map[string][]string)
		}
		history := c.inputHistory[c.session.ID]
		if len(history) == 0 || history[len(history)-1] != c.pendingFirstHistoryEntry {
			c.inputHistory[c.session.ID] = append(history, c.pendingFirstHistoryEntry)
		}
		c.pendingFirstHistoryEntry = ""
	}
	return nil
}

func (c *editorCmp) IsCompletionsOpen() bool {
	return c.isCompletionsOpen
}

func (c *editorCmp) HasAttachments() bool {
	return len(c.attachments) > 0
}

func (m *editorCmp) normalPromptFunc(info textarea.PromptInfo) string {
	t := styles.CurrentTheme()
	if info.LineNumber == 0 {
		return "  > "
	}

	// Only show continuation prompt if there are multiple lines
	lines := strings.Split(m.textarea.Value(), "\n")
	if len(lines) <= 1 {
		return ""
	}

	if info.Focused {
		return t.S().Base.Foreground(t.GreenDark).Render("::: ")
	}
	return t.S().Muted.Render("::: ")
}

func (m *editorCmp) yoloPromptFunc(info textarea.PromptInfo) string {
	t := styles.CurrentTheme()
	if info.LineNumber == 0 {
		if info.Focused {
			return fmt.Sprintf("%s ", t.YoloIconFocused)
		} else {
			return fmt.Sprintf("%s ", t.YoloIconBlurred)
		}
	}

	// Only show continuation prompt if there are multiple lines
	lines := strings.Split(m.textarea.Value(), "\n")
	if len(lines) <= 1 {
		return ""
	}

	if info.Focused {
		return fmt.Sprintf("%s ", t.YoloDotsFocused)
	}
	return fmt.Sprintf("%s ", t.YoloDotsBlurred)
}

func New(app *app.App) Editor {
	t := styles.CurrentTheme()
	ta := textarea.New()
	ta.SetStyles(t.S().TextArea)
	ta.ShowLineNumbers = false
	ta.CharLimit = -1
	ta.SetVirtualCursor(false)
	ta.Focus()
	e := &editorCmp{
		// TODO: remove the app instance from here
		app:      app,
		textarea: ta,
		keyMap:   DefaultEditorKeyMap(),
		// history defaults
		inputHistory: make(map[string][]string),
		historyIndex: -1,
	}
	e.setEditorPrompt()

	e.randomizePlaceholders()
	e.textarea.Placeholder = e.readyPlaceholder

	return e
}
