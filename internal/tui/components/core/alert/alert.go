package alert

import (
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/lacymorrow/lash/internal/tui/styles"
	"github.com/lacymorrow/lash/internal/tui/util"
)

type AlertCmp interface {
	util.Model
	IsVisible() bool
}

type alertCmp struct {
	info       util.InfoMsg
	width      int
	messageTTL time.Duration
}

// clearMessageCmd is a command that clears alert messages after a timeout
func (m *alertCmp) clearMessageCmd(ttl time.Duration) tea.Cmd {
	return tea.Tick(ttl, func(time.Time) tea.Msg {
		return util.ClearStatusMsg{}
	})
}

func (m *alertCmp) Init() tea.Cmd {
	return nil
}

func (m *alertCmp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	// Handle alert info
	case util.InfoMsg:
		m.info = msg
		ttl := msg.TTL
		if ttl == 0 {
			ttl = m.messageTTL
		}
		return m, m.clearMessageCmd(ttl)
	case util.ClearStatusMsg:
		m.info = util.InfoMsg{}
	}
	return m, nil
}

func (m *alertCmp) View() string {
	if m.info.Msg == "" {
		return ""
	}
	return m.infoMsg()
}

func (m *alertCmp) IsVisible() bool {
	return m.info.Msg != ""
}

func (m *alertCmp) infoMsg() string {
	t := styles.CurrentTheme()
	message := ""
	infoType := ""
	switch m.info.Type {
	case util.InfoTypeError:
		infoType = t.S().Base.Background(t.Red).Padding(0, 1).Render("ERROR")
		widthLeft := m.width - (lipgloss.Width(infoType) + 2)
		info := ansi.Truncate(m.info.Msg, widthLeft, "…")
		message = t.S().Base.Background(t.Error).Width(widthLeft+2).Foreground(t.White).Padding(0, 1).Render(info)
	case util.InfoTypeWarn:
		infoType = t.S().Base.Foreground(t.BgOverlay).Background(t.Yellow).Padding(0, 1).Render("WARNING")
		widthLeft := m.width - (lipgloss.Width(infoType) + 2)
		info := ansi.Truncate(m.info.Msg, widthLeft, "…")
		message = t.S().Base.Foreground(t.BgOverlay).Width(widthLeft+2).Background(t.Warning).Padding(0, 1).Render(info)
	default:
		infoType = t.S().Base.Foreground(t.BgOverlay).Background(t.Green).Padding(0, 1).Render("OKAY!")
		widthLeft := m.width - (lipgloss.Width(infoType) + 2)
		info := ansi.Truncate(m.info.Msg, widthLeft, "…")
		message = t.S().Base.Background(t.Success).Width(widthLeft+2).Foreground(t.White).Padding(0, 1).Render(info)
	}
	return ansi.Truncate(infoType+message, m.width, "…")
}

func NewAlertCmp() AlertCmp {
	return &alertCmp{
		messageTTL: 5 * time.Second,
	}
}
