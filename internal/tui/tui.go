package tui

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/v2/key"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/lacymorrow/lash/internal/app"
	"github.com/lacymorrow/lash/internal/config"
	"github.com/lacymorrow/lash/internal/llm/agent"
	"github.com/lacymorrow/lash/internal/permission"
	"github.com/lacymorrow/lash/internal/pubsub"
	cmpChat "github.com/lacymorrow/lash/internal/tui/components/chat"
	"github.com/lacymorrow/lash/internal/tui/components/chat/splash"
	"github.com/lacymorrow/lash/internal/tui/components/completions"
	"github.com/lacymorrow/lash/internal/tui/components/core"
	"github.com/lacymorrow/lash/internal/tui/components/core/layout"
	"github.com/lacymorrow/lash/internal/tui/components/core/status"
	"github.com/lacymorrow/lash/internal/tui/components/dialogs"
	"github.com/lacymorrow/lash/internal/tui/components/dialogs/commands"
	"github.com/lacymorrow/lash/internal/tui/components/dialogs/compact"
	"github.com/lacymorrow/lash/internal/tui/components/dialogs/filepicker"
	"github.com/lacymorrow/lash/internal/tui/components/dialogs/models"
	"github.com/lacymorrow/lash/internal/tui/components/dialogs/permissions"
	"github.com/lacymorrow/lash/internal/tui/components/dialogs/quit"
	"github.com/lacymorrow/lash/internal/tui/components/dialogs/sessions"
	"github.com/lacymorrow/lash/internal/tui/page"
	"github.com/lacymorrow/lash/internal/tui/page/chat"
	"github.com/lacymorrow/lash/internal/tui/styles"
	"github.com/lacymorrow/lash/internal/tui/util"
)

var lastMouseEvent time.Time

func MouseEventFilter(m tea.Model, msg tea.Msg) tea.Msg {
	switch msg.(type) {
	case tea.MouseWheelMsg, tea.MouseMotionMsg:
		now := time.Now()
		// trackpad is sending too many requests
		if now.Sub(lastMouseEvent) < 15*time.Millisecond {
			return nil
		}
		lastMouseEvent = now
	}
	return msg
}

// appModel represents the main application model that manages pages, dialogs, and UI state.
type appModel struct {
	wWidth, wHeight int // Window dimensions
	width, height   int
	keyMap          KeyMap

	currentPage  page.PageID
	previousPage page.PageID
	pages        map[page.PageID]util.Model
	loadedPages  map[page.PageID]bool

	// Status
	status          status.StatusCmp
	showingFullHelp bool

	app *app.App

	dialog       dialogs.DialogCmp
	completions  completions.Completions
	isConfigured bool

	// Chat Page Specific
	selectedSessionID string // The ID of the currently selected session

	// Active mode state for status display
	activeMode string
}

// routeToActive routes a message to the active dialog if present, otherwise to the current page.
func (a *appModel) routeToActive(msg tea.Msg) tea.Cmd {
	if a.dialog.HasDialogs() {
		u, dialogCmd := a.dialog.Update(msg)
		a.dialog = u.(dialogs.DialogCmp)
		return dialogCmd
	}
	updated, pageCmd := a.pages[a.currentPage].Update(msg)
	a.pages[a.currentPage] = updated.(util.Model)
	return pageCmd
}

// ActiveMode returns the current mode string (Shell/Agent/Auto)
func (a *appModel) ActiveMode() string { return a.activeMode }

// renderModeBadge returns a colored/icon badge for the given mode without a prefix.
func (a *appModel) renderModeBadge(mode string) string {
	t := styles.CurrentTheme()
	base := t.S().Base.Bold(true)
	switch mode {
	case "Shell":
		icon := t.S().Base.Foreground(t.Secondary).Render("▌")
		label := base.Foreground(t.Secondary).Render("Shell")
		return lipgloss.JoinHorizontal(lipgloss.Left, icon, " ", label)
	case "Agent":
		icon := t.S().Base.Foreground(t.Accent).Render("▌")
		label := base.Foreground(t.Accent).Render("Agent")
		return lipgloss.JoinHorizontal(lipgloss.Left, icon, " ", label)
	default: // Auto
		icon := t.S().Base.Foreground(t.Primary).Render("▌")
		label := base.Foreground(t.Primary).Render("Auto ")
		return lipgloss.JoinHorizontal(lipgloss.Left, icon, " ", label)
	}
}

// renderLeftPrefix composes the left status prefix: Mode badge and optional YOLO badge.
func (a *appModel) renderLeftPrefix() string {
	left := a.renderModeBadge(a.activeMode)
	if a.app != nil && a.app.Permissions != nil && a.app.Permissions.SkipRequests() {
		t := styles.CurrentTheme()
		base := t.S().Base.Bold(true)
		icon := t.S().Base.Foreground(t.Red).Render("▌")
		label := base.Foreground(t.Red).Render("YOLO")
		yolo := lipgloss.JoinHorizontal(lipgloss.Left, icon, " ", label)
		left = lipgloss.JoinHorizontal(lipgloss.Left, left, "   ", yolo)
	}
	return left
}

// Init initializes the application model and returns initial commands.
func (a *appModel) Init() tea.Cmd {
	var cmds []tea.Cmd
	cmd := a.pages[a.currentPage].Init()
	cmds = append(cmds, cmd)
	a.loadedPages[a.currentPage] = true

	cmd = a.status.Init()
	cmds = append(cmds, cmd)

	cmds = append(cmds, tea.EnableMouseAllMotion)

	// Initialize mode from config helper
	a.activeMode = config.Get().ActiveMode()
	a.app.Mode = a.activeMode
	// Always show initial info on launch
	a.status.SetLeft(a.renderLeftPrefix())

	return tea.Batch(cmds...)
}

// Update handles incoming messages and updates the application state.
func (a *appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd
	a.isConfigured = config.HasInitialDataConfig()

	switch msg := msg.(type) {
	case tea.KeyboardEnhancementsMsg:
		for id, page := range a.pages {
			m, pageCmd := page.Update(msg)
			a.pages[id] = m.(util.Model)
			if pageCmd != nil {
				cmds = append(cmds, pageCmd)
			}
		}
		return a, tea.Batch(cmds...)
	case tea.WindowSizeMsg:
		a.wWidth, a.wHeight = msg.Width, msg.Height
		u, completionCmd := a.completions.Update(msg)
		a.completions = u.(completions.Completions)
		return a, tea.Batch(completionCmd, a.handleWindowResize(msg.Width, msg.Height))

	// Terminal resumed (e.g., after suspend or losing window focus). Re-apply layout
	// and give the active page a chance to restore focus.
	case tea.ResumeMsg:
		var cmds []tea.Cmd
		cmds = append(cmds, a.handleWindowResize(a.wWidth, a.wHeight))
		updated, pageCmd := a.pages[a.currentPage].Update(msg)
		a.pages[a.currentPage] = updated.(util.Model)
		cmds = append(cmds, pageCmd)
		return a, tea.Batch(cmds...)

	// Completions messages
	case completions.OpenCompletionsMsg, completions.FilterCompletionsMsg,
		completions.CloseCompletionsMsg, completions.RepositionCompletionsMsg:
		u, completionCmd := a.completions.Update(msg)
		a.completions = u.(completions.Completions)
		return a, completionCmd

	// Dialog messages
	case dialogs.OpenDialogMsg, dialogs.CloseDialogMsg:
		u, completionCmd := a.completions.Update(completions.CloseCompletionsMsg{})
		a.completions = u.(completions.Completions)
		u, dialogCmd := a.dialog.Update(msg)
		a.dialog = u.(dialogs.DialogCmp)
		return a, tea.Batch(completionCmd, dialogCmd)
	case commands.ShowArgumentsDialogMsg:
		return a, util.CmdHandler(
			dialogs.OpenDialogMsg{
				Model: commands.NewCommandArgumentsDialog(
					msg.CommandID,
					msg.Content,
					msg.ArgNames,
				),
			},
		)
	// Page change messages
	case page.PageChangeMsg:
		return a, a.moveToPage(msg.ID)

	// Status Messages
	case util.InfoMsg, util.ClearStatusMsg:
		s, statusCmd := a.status.Update(msg)
		a.status = s.(status.StatusCmp)
		cmds = append(cmds, statusCmd)
		return a, tea.Batch(cmds...)

	// Session
	case cmpChat.SessionSelectedMsg:
		a.selectedSessionID = msg.ID
	case cmpChat.SessionClearedMsg:
		a.selectedSessionID = ""
	// Commands
	case commands.SwitchSessionsMsg:
		return a, func() tea.Msg {
			allSessions, _ := a.app.Sessions.List(context.Background())
			return dialogs.OpenDialogMsg{
				Model: sessions.NewSessionDialogCmp(allSessions, a.selectedSessionID),
			}
		}

	case commands.SwitchModelMsg:
		return a, util.CmdHandler(
			dialogs.OpenDialogMsg{
				Model: models.NewModelDialogCmp(),
			},
		)
	// Compact
	case commands.CompactMsg:
		return a, util.CmdHandler(dialogs.OpenDialogMsg{
			Model: compact.NewCompactDialogCmp(a.app.CoderAgent, msg.SessionID, true),
		})
	case commands.QuitMsg:
		return a, util.CmdHandler(dialogs.OpenDialogMsg{
			Model: quit.NewQuitDialog(),
		})
	case commands.ToggleYoloModeMsg:
		// Toggle YOLO skip mode
		if a.app == nil || a.app.Permissions == nil {
			return a, util.ReportError(fmt.Errorf("permissions service unavailable"))
		}
		newSkip := !a.app.Permissions.SkipRequests()
		a.app.Permissions.SetSkipRequests(newSkip)
		a.status.SetLeft(a.renderLeftPrefix())

		// Persist YOLO flag and also toggle auto-confirm to mirror YOLO state
		cfg := config.Get()
		if cfg.Lash == nil {
			cfg.Lash = &config.LashConfig{}
		}
		_ = cfg.SetConfigField("lash.yolo", newSkip)
		currentAllowed := []string{"bash", "bash:execute"}
		if newSkip {
			// Enable auto-confirm when YOLO is on
			falseVal := false
			cfg.Lash.Safety.ConfirmAgentExec = &falseVal
			_ = cfg.SetConfigField("lash.safety.confirm_agent_exec", false)
			a.app.Permissions.SetAllowedTools(currentAllowed)
			return a, util.ReportInfo("YOLO enabled (auto-confirm on)")
		}
		// Disable auto-confirm when YOLO is off
		trueVal := true
		cfg.Lash.Safety.ConfirmAgentExec = &trueVal
		_ = cfg.SetConfigField("lash.safety.confirm_agent_exec", true)
		a.app.Permissions.SetAllowedTools([]string{})
		return a, util.ReportInfo("YOLO disabled (auto-confirm off)")
	case commands.ToggleHelpMsg:
		a.status.ToggleFullHelp()
		a.showingFullHelp = !a.showingFullHelp
		return a, a.handleWindowResize(a.wWidth, a.wHeight)
	// Model Switch
	case models.ModelSelectedMsg:
		// Ensure agent config exists (first-time setup after provider auth)
		wasConfigured := a.isConfigured
		cfg := config.Get()
		if cfg.Agents == nil || cfg.Agents["coder"].ID == "" {
			cfg.SetupAgents()
			// sync into app's config
			a.app.Config().Agents = cfg.Agents
		}
		// Persist the newly selected model before initializing/updating the agent
		// so provider resolution (e.g., synthetic 'anthropic-max') is available.
		cfg.UpdatePreferredModel(msg.ModelType, msg.Model)
		// Sanity-check connectivity for the selected provider to surface actionable errors early
		if pc := cfg.GetProviderForModel(msg.ModelType); pc != nil {
			if err := pc.TestConnection(cfg.Resolver()); err != nil {
				// For OAuth providers, the error might be due to expired token which will be refreshed on use
				if pc.ID == "anthropic-max" && strings.Contains(err.Error(), "401") {
					// Log warning but continue - token will be refreshed on actual use
					slog.Warn("OAuth token may be expired, will refresh on use", "provider", pc.ID)
				} else {
					return a, util.ReportError(fmt.Errorf("failed provider connectivity for %s: %w", pc.ID, err))
				}
			}
		}
		// Ensure the selected provider exists in config. If missing (e.g., added via OAuth), seed from known providers.
		selectedProviderID := msg.Model.Provider
		if _, ok := cfg.Providers.Get(selectedProviderID); !ok {
			if known, err := config.Providers(); err == nil {
				for _, kp := range known {
					if string(kp.ID) == selectedProviderID {
						pc := config.ProviderConfig{
							ID:           selectedProviderID,
							Name:         kp.Name,
							BaseURL:      kp.APIEndpoint,
							Type:         kp.Type,
							ExtraHeaders: map[string]string{},
							Models:       kp.Models,
						}
						cfg.Providers.Set(selectedProviderID, pc)
						_ = cfg.SetConfigField("providers."+selectedProviderID, pc)
						break
					}
				}
			}
		}
		// Ensure small model provider is valid; if missing/invalid, default to the selected provider
		small := cfg.Models[config.SelectedModelTypeSmall]
		if small.Provider == "" {
			cfg.Models[config.SelectedModelTypeSmall] = msg.Model
			_ = cfg.SetConfigField("models.small", msg.Model)
		} else {
			if _, ok := cfg.Providers.Get(small.Provider); !ok {
				cfg.Models[config.SelectedModelTypeSmall] = msg.Model
				_ = cfg.SetConfigField("models.small", msg.Model)
			}
		}
		// If agent isn't initialized yet (e.g., first-time config via OAuth), initialize it now
		if a.app.CoderAgent == nil {
			if err := a.app.InitCoderAgent(); err != nil {
				// Retry once after forcing agent setup
				cfg.SetupAgents()
				a.app.Config().Agents = cfg.Agents
				if err2 := a.app.InitCoderAgent(); err2 != nil {
					return a, util.ReportError(fmt.Errorf("failed to initialize agent after model selection: %v", err2))
				}
			}
		}
		if a.app.CoderAgent != nil && a.app.CoderAgent.IsBusy() {
			return a, util.ReportWarn("Agent is busy, please wait...")
		}
		// Update the agent with the new model/provider configuration
		if err := a.app.UpdateAgentModel(); err != nil {
			return a, util.ReportError(fmt.Errorf("model changed to %s but failed to update agent: %v", msg.Model.Model, err))
		}

		modelTypeName := "large"
		if msg.ModelType == config.SelectedModelTypeSmall {
			modelTypeName = "small"
		}
		infoCmd := util.ReportInfo(fmt.Sprintf("%s model changed to %s", modelTypeName, msg.Model.Model))
		// If this is the first configuration flow (onboarding), notify splash to complete
		if !wasConfigured {
			return a, tea.Batch(infoCmd, util.CmdHandler(splash.OnboardingCompleteMsg{}))
		}
		return a, infoCmd

	// File Picker
	case commands.OpenFilePickerMsg:
		if a.dialog.ActiveDialogID() == filepicker.FilePickerID {
			// If the commands dialog is already open, close it
			return a, util.CmdHandler(dialogs.CloseDialogMsg{})
		}
		return a, util.CmdHandler(dialogs.OpenDialogMsg{
			Model: filepicker.NewFilePickerCmp(a.app.Config().WorkingDir()),
		})
	// Open config file in $EDITOR (create parent dirs if missing)
	case commands.OpenConfigFileMsg:
		cfgPath := config.GlobalConfigData()
		if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
			return a, util.ReportError(fmt.Errorf("failed to ensure config directory: %w", err))
		}
		if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
			// create minimal file if missing
			if err := os.WriteFile(cfgPath, []byte("{}\n"), 0o600); err != nil {
				return a, util.ReportError(fmt.Errorf("failed to create config file: %w", err))
			}
		}
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "nvim"
		}
		cmd := exec.CommandContext(context.TODO(), editor, cfgPath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return a, tea.ExecProcess(cmd, func(err error) tea.Msg {
			if err != nil {
				return util.ReportError(err)
			}
			return util.ReportInfo("Closed config file")
		})
	// Permissions
	case pubsub.Event[permission.PermissionNotification]:
		// forward to page
		updated, cmd := a.pages[a.currentPage].Update(msg)
		a.pages[a.currentPage] = updated.(util.Model)
		return a, cmd
	case pubsub.Event[permission.PermissionRequest]:
		return a, util.CmdHandler(dialogs.OpenDialogMsg{
			Model: permissions.NewPermissionDialogCmp(msg.Payload, &permissions.Options{}),
		})
	case permissions.PermissionResponseMsg:
		if a.app != nil && a.app.Permissions != nil {
			switch msg.Action {
			case permissions.PermissionAllow:
				a.app.Permissions.Grant(msg.Permission)
			case permissions.PermissionAllowForSession:
				a.app.Permissions.GrantPersistent(msg.Permission)
			case permissions.PermissionDeny:
				a.app.Permissions.Deny(msg.Permission)
			}
		}
		return a, nil
	// Agent Events
	case pubsub.Event[agent.AgentEvent]:
		payload := msg.Payload

		// Forward agent events to dialogs
		if a.dialog.HasDialogs() && a.dialog.ActiveDialogID() == compact.CompactDialogID {
			u, dialogCmd := a.dialog.Update(payload)
			a.dialog = u.(dialogs.DialogCmp)
			cmds = append(cmds, dialogCmd)
		}

		// Handle auto-compact logic
		if payload.Done && payload.Type == agent.AgentEventTypeResponse && a.selectedSessionID != "" {
			// Get current session to check token usage
			session, err := a.app.Sessions.Get(context.Background(), a.selectedSessionID)
			if err == nil {
				model := a.app.CoderAgent.Model()
				contextWindow := model.ContextWindow
				tokens := session.CompletionTokens + session.PromptTokens
				cfg := config.Get()
				disableAuto := cfg.Options != nil && cfg.Options.DisableAutoSummarize
				if (tokens >= int64(float64(contextWindow)*0.95)) && !disableAuto { // Show compact confirmation dialog
					cmds = append(cmds, util.CmdHandler(dialogs.OpenDialogMsg{
						Model: compact.NewCompactDialogCmp(a.app.CoderAgent, a.selectedSessionID, false),
					}))
				}
			}
		}

		return a, tea.Batch(cmds...)
	case splash.OnboardingCompleteMsg:
		a.isConfigured = config.HasInitialDataConfig()
		updated, pageCmd := a.pages[a.currentPage].Update(msg)
		a.pages[a.currentPage] = updated.(util.Model)
		cmds = append(cmds, pageCmd)
		return a, tea.Batch(cmds...)
	// Key Press Messages
	case tea.KeyPressMsg:
		return a, a.handleKeyPressMsg(msg)

	case tea.MouseWheelMsg:
		return a, a.routeToActive(msg)
	case tea.PasteMsg:
		return a, a.routeToActive(msg)
	}
	s, _ := a.status.Update(msg)
	a.status = s.(status.StatusCmp)
	updated, cmd := a.pages[a.currentPage].Update(msg)
	a.pages[a.currentPage] = updated.(util.Model)
	if a.dialog.HasDialogs() {
		u, dialogCmd := a.dialog.Update(msg)
		a.dialog = u.(dialogs.DialogCmp)
		cmds = append(cmds, dialogCmd)
	}
	cmds = append(cmds, cmd)
	return a, tea.Batch(cmds...)
}

// handleWindowResize processes window resize events and updates all components.
func (a *appModel) handleWindowResize(width, height int) tea.Cmd {
	var cmds []tea.Cmd
	if a.showingFullHelp {
		height -= 5
	} else {
		height -= 2
	}
	a.width, a.height = width, height
	// Update status bar
	s, cmd := a.status.Update(tea.WindowSizeMsg{Width: width, Height: height})
	a.status = s.(status.StatusCmp)
	cmds = append(cmds, cmd)

	// Update the current page
	for p, page := range a.pages {
		updated, pageCmd := page.Update(tea.WindowSizeMsg{Width: width, Height: height})
		a.pages[p] = updated.(util.Model)
		cmds = append(cmds, pageCmd)
	}

	// Update the dialogs
	dialog, cmd := a.dialog.Update(tea.WindowSizeMsg{Width: width, Height: height})
	a.dialog = dialog.(dialogs.DialogCmp)
	cmds = append(cmds, cmd)

	return tea.Batch(cmds...)
}

// handleKeyPressMsg processes keyboard input and routes to appropriate handlers.
func (a *appModel) handleKeyPressMsg(msg tea.KeyPressMsg) tea.Cmd {
	if a.completions.Open() {
		// completions
		keyMap := a.completions.KeyMap()
		switch {
		case key.Matches(msg, keyMap.Up), key.Matches(msg, keyMap.Down),
			key.Matches(msg, keyMap.Select), key.Matches(msg, keyMap.Cancel),
			key.Matches(msg, keyMap.UpInsert), key.Matches(msg, keyMap.DownInsert):
			u, cmd := a.completions.Update(msg)
			a.completions = u.(completions.Completions)
			return cmd
		}
	}
	// Immediate quit on Ctrl+D (EOF)
	if key.Matches(msg, a.keyMap.QuitEOF) {
		return tea.Quit
	}
	// Treat Ctrl+C like Esc for open menus/dialogs; otherwise follow quit flow
	if key.Matches(msg, a.keyMap.Quit) {
		// If completions popup is open, close it (acts like Esc)
		if a.completions.Open() {
			u, cmd := a.completions.Update(completions.CloseCompletionsMsg{})
			a.completions = u.(completions.Completions)
			return cmd
		}
		if a.dialog.HasDialogs() {
			if a.dialog.ActiveDialogID() == quit.QuitDialogID {
				return tea.Quit
			}
			// Close the topmost dialog (acts like Esc)
			return util.CmdHandler(dialogs.CloseDialogMsg{})
		}
		updated, pageCmd := a.pages[a.currentPage].Update(msg)
		a.pages[a.currentPage] = updated.(util.Model)
		if pageCmd != nil {
			return pageCmd
		}
		return util.CmdHandler(dialogs.OpenDialogMsg{Model: quit.NewQuitDialog()})
	}
	switch {
	// help
	case key.Matches(msg, a.keyMap.Help):
		a.status.ToggleFullHelp()
		a.showingFullHelp = !a.showingFullHelp
		return a.handleWindowResize(a.wWidth, a.wHeight)
	// dialogs
	case key.Matches(msg, a.keyMap.Commands):
		// if the app is not configured show no commands
		if !a.isConfigured {
			return nil
		}
		if a.dialog.ActiveDialogID() == commands.CommandsDialogID {
			return util.CmdHandler(dialogs.CloseDialogMsg{})
		}
		if a.dialog.HasDialogs() {
			return nil
		}
		return util.CmdHandler(dialogs.OpenDialogMsg{
			Model: commands.NewCommandDialog(a.selectedSessionID),
		})
	case key.Matches(msg, a.keyMap.Sessions):
		// if the app is not configured show no sessions
		if !a.isConfigured {
			return nil
		}
		if a.dialog.ActiveDialogID() == sessions.SessionsDialogID {
			return util.CmdHandler(dialogs.CloseDialogMsg{})
		}
		if a.dialog.HasDialogs() && a.dialog.ActiveDialogID() != commands.CommandsDialogID {
			return nil
		}
		var cmds []tea.Cmd
		if a.dialog.ActiveDialogID() == commands.CommandsDialogID {
			// If the commands dialog is open, close it first
			cmds = append(cmds, util.CmdHandler(dialogs.CloseDialogMsg{}))
		}
		cmds = append(cmds,
			func() tea.Msg {
				allSessions, _ := a.app.Sessions.List(context.Background())
				return dialogs.OpenDialogMsg{
					Model: sessions.NewSessionDialogCmp(allSessions, a.selectedSessionID),
				}
			},
		)
		return tea.Sequence(cmds...)
	case key.Matches(msg, a.keyMap.ToggleMode):
		// Cycle through Shell -> Agent -> Auto -> Shell
		switch a.activeMode {
		case "Shell":
			a.activeMode = "Agent"
		case "Agent":
			a.activeMode = "Auto"
		default:
			a.activeMode = "Shell"
		}
		a.status.SetLeft(a.renderLeftPrefix())
		a.app.Mode = a.activeMode
		// Persist last selected mode using config helper
		if err := config.Get().SetActiveMode(a.activeMode); err != nil {
			return tea.Batch(util.ReportWarn("failed to persist mode: "+err.Error()), a.handleWindowResize(a.wWidth, a.wHeight))
		}
		return a.handleWindowResize(a.wWidth, a.wHeight)
	case key.Matches(msg, a.keyMap.ToggleYolo):
		// Route through the common handler so pages update their prompts/styles
		return util.CmdHandler(commands.ToggleYoloModeMsg{})
	case key.Matches(msg, a.keyMap.ToggleAutoConfirm):
		// Toggle Lash safety confirm flag at runtime and persist to config data file
		cfg := config.Get()
		if cfg.Lash == nil {
			cfg.Lash = &config.LashConfig{}
		}
		current := true
		if cfg.Lash.Safety.ConfirmAgentExec != nil {
			current = *cfg.Lash.Safety.ConfirmAgentExec
		}
		newVal := !current
		cfg.Lash.Safety.ConfirmAgentExec = &newVal
		persistErr := cfg.SetConfigField("lash.safety.confirm_agent_exec", newVal)
		// Update permission service allowlist live
		currentAllowed := []string{"bash", "bash:execute"}
		if !newVal {
			// enable auto-confirm: allow bash executions without prompts
			if a.app != nil && a.app.Permissions != nil {
				a.app.Permissions.SetAllowedTools(currentAllowed)
			}
			if persistErr != nil {
				return tea.Batch(util.ReportWarn("failed to persist auto-confirm: "+persistErr.Error()), util.ReportInfo("Auto-confirm enabled"))
			}
			return util.ReportInfo("Auto-confirm enabled")
		}
		// disable auto-confirm: remove from allowlist by setting empty list
		if a.app != nil && a.app.Permissions != nil {
			a.app.Permissions.SetAllowedTools([]string{})
		}
		if persistErr != nil {
			return tea.Batch(util.ReportWarn("failed to persist auto-confirm: "+persistErr.Error()), util.ReportInfo("Auto-confirm disabled"))
		}
		return util.ReportInfo("Auto-confirm disabled")
	case key.Matches(msg, a.keyMap.Suspend):
		if a.app.CoderAgent != nil && a.app.CoderAgent.IsBusy() {
			return util.ReportWarn("Agent is busy, please wait...")
		}
		return tea.Suspend
	default:
		if a.dialog.HasDialogs() {
			u, dialogCmd := a.dialog.Update(msg)
			a.dialog = u.(dialogs.DialogCmp)
			return dialogCmd
		} else {
			updated, cmd := a.pages[a.currentPage].Update(msg)
			a.pages[a.currentPage] = updated.(util.Model)
			return cmd
		}
	}
}

// moveToPage handles navigation between different pages in the application.
func (a *appModel) moveToPage(pageID page.PageID) tea.Cmd {
	if a.app.CoderAgent.IsBusy() {
		// TODO: maybe remove this :  For now we don't move to any page if the agent is busy
		return util.ReportWarn("Agent is busy, please wait...")
	}

	var cmds []tea.Cmd
	if _, ok := a.loadedPages[pageID]; !ok {
		cmd := a.pages[pageID].Init()
		cmds = append(cmds, cmd)
		a.loadedPages[pageID] = true
	}
	a.previousPage = a.currentPage
	a.currentPage = pageID
	if sizable, ok := a.pages[a.currentPage].(layout.Sizeable); ok {
		cmd := sizable.SetSize(a.width, a.height)
		cmds = append(cmds, cmd)
	}

	return tea.Batch(cmds...)
}

// View renders the complete application interface including pages, dialogs, and overlays.
func (a *appModel) View() tea.View {
	var view tea.View
	t := styles.CurrentTheme()
	view.BackgroundColor = t.BgBase
	if a.wWidth < 25 || a.wHeight < 15 {
		view.Layer = lipgloss.NewCanvas(
			lipgloss.NewLayer(
				t.S().Base.Width(a.wWidth).Height(a.wHeight).
					Align(lipgloss.Center, lipgloss.Center).
					Render(
						t.S().Base.
							Padding(1, 4).
							Foreground(t.White).
							BorderStyle(lipgloss.RoundedBorder()).
							BorderForeground(t.Primary).
							Render("Window too small!"),
					),
			),
		)
		return view
	}

	page := a.pages[a.currentPage]
	if withHelp, ok := page.(core.KeyMapHelp); ok {
		a.status.SetKeyMap(withHelp.Help())
	}
	pageView := page.View()
	components := []string{
		pageView,
	}
	components = append(components, a.status.View())

	appView := lipgloss.JoinVertical(lipgloss.Top, components...)
	layers := []*lipgloss.Layer{
		lipgloss.NewLayer(appView),
	}
	if a.dialog.HasDialogs() {
		layers = append(
			layers,
			a.dialog.GetLayers()...,
		)
	}

	var cursor *tea.Cursor
	if v, ok := page.(util.Cursor); ok {
		cursor = v.Cursor()
		// Hide the cursor if it's positioned outside the textarea
		statusHeight := a.height - strings.Count(pageView, "\n") + 1
		if cursor != nil && cursor.Y+statusHeight+chat.EditorHeight-2 <= a.height { // 2 for the top and bottom app padding
			cursor = nil
		}
	}
	activeView := a.dialog.ActiveModel()
	if activeView != nil {
		cursor = nil // Reset cursor if a dialog is active unless it implements util.Cursor
		if v, ok := activeView.(util.Cursor); ok {
			cursor = v.Cursor()
		}
	}

	if a.completions.Open() && cursor != nil {
		cmp := a.completions.View()
		x, y := a.completions.Position()
		layers = append(
			layers,
			lipgloss.NewLayer(cmp).X(x).Y(y),
		)
	}

	canvas := lipgloss.NewCanvas(
		layers...,
	)

	view.Layer = canvas
	view.Cursor = cursor
	return view
}

// New creates and initializes a new TUI application model.
func New(app *app.App) tea.Model {
	chatPage := chat.New(app)
	keyMap := DefaultKeyMap()
	keyMap.pageBindings = chatPage.Bindings()

	model := &appModel{
		currentPage: chat.ChatPageID,
		app:         app,
		status:      status.NewStatusCmp(),
		loadedPages: make(map[page.PageID]bool),
		keyMap:      keyMap,

		pages: map[page.PageID]util.Model{
			chat.ChatPageID: chatPage,
		},

		dialog:      dialogs.NewDialogCmp(),
		completions: completions.New(),

		activeMode: "Auto",
	}
	// Register a minimal accessor for other components
	util.RegisterAppModel(model)

	return model
}
