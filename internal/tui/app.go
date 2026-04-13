package tui

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// tab identifies which tab is active.
type tab int

const (
	tabOverview tab = iota
	tabBackups
	tabConfig
	tabCount // sentinel for cycling
)

var tabNames = [tabCount]string{
	"Overview",
	"Backups",
	"Config",
}

// Options configures the TUI at startup. Fields are resolved from CLI flags,
// config file, and connection context before the TUI is created.
type Options struct {
	URI         string // MongoDB connection URI (required)
	ThemeName   string // Selected theme name (e.g. "default", "mocha")
	ContextName string // Named context (empty for direct --uri connections)
	Readonly    bool   // Disable all mutation actions
	Editor      string // External editor command (e.g. "vim", "code -w")
}

// Model is the root BubbleTea model for PBMate.
type Model struct {
	client      *sdk.Client // nil until connectMsg arrives
	mongoURI    string      // connection URI for background connect
	contextName string      // named context, empty for direct URI
	readonly    bool        // disable all mutation actions
	editor      string      // resolved editor command (e.g. "vim", "code -w")
	themeName   string
	theme       Theme
	ctx         context.Context
	cancel      context.CancelFunc

	styles *Styles

	activeTab       tab
	width           int
	height          int
	connecting      bool   // true while initial connection is in progress (including retries)
	connectAttempt  int    // number of connection attempts (0 = first try, 1+ = retries)
	flashErr        string // transient error message for the status bar
	flashFromAction bool   // true when flashErr was set by a user action (sticky across polls)

	// activeOverlay captures all input when non-nil. Overlays include
	// backup forms, file pickers, profile name forms, and confirm dialogs.
	activeOverlay formOverlay

	// exitMessage is printed to stdout after the TUI exits. Used by
	// physical/incremental restore to inform the user after dispatch.
	exitMessage string

	// Help overlay — when true, the ? help panel is shown.
	showHelp bool

	// quitPending is true after the first q press. A second q within
	// quitTimeout actually quits. The timer auto-clears the state.
	quitPending bool

	// Spinner for connecting and running operation indicators.
	spinner spinner.Model

	// Sub-models.
	overview overviewModel
	backups  backupsModel
	config   configModel

	keys globalKeyMap
}

// New creates a new root model from the given options. The SDK connection
// is established asynchronously — the TUI renders immediately while
// connecting in the background.
func New(opts Options) Model {
	ctx, cancel := context.WithCancel(context.Background())
	sp := spinner.New()
	sp.Spinner = spinner.MiniDot
	styles := &Styles{}
	m := Model{
		mongoURI:    opts.URI,
		contextName: opts.ContextName,
		readonly:    opts.Readonly,
		editor:      opts.Editor,
		themeName:   opts.ThemeName,
		ctx:         ctx,
		cancel:      cancel,
		styles:      styles,
		activeTab:   tabOverview,
		connecting:  true,
		spinner:     sp,
		overview:    newOverviewModel(styles),
		backups:     newBackupsModel(styles),
		config:      newConfigModel(styles),
		keys:        globalKeys,
	}
	m.applyTheme(LookupTheme(opts.ThemeName, true))
	return m
}

func (m Model) formTheme() huh.Theme {
	return m.theme.HuhTheme()
}

func (m *Model) applyTheme(theme Theme) {
	m.theme = theme
	*m.styles = theme.Styles()
}

// Close cancels the root context and disconnects the SDK client.
// Safe to call when the client is nil (e.g. connection never succeeded).
func (m Model) Close() {
	m.cancel()
	if m.client != nil {
		_ = m.client.Close(context.Background())
	}
}

// ExitMessage returns a message to print to stdout after the TUI exits.
// Empty string means no message. Used by physical/incremental restores
// to inform the user after dispatch triggers a clean exit.
func (m Model) ExitMessage() string {
	return m.exitMessage
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.RequestWindowSize,
		tea.RequestBackgroundColor,
		connectCmd(m.mongoURI),
		m.spinner.Tick,
	)
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if bgMsg, ok := msg.(tea.BackgroundColorMsg); ok {
		m.applyTheme(LookupTheme(m.themeName, bgMsg.IsDark()))
	}

	// Data and system messages are handled first regardless of overlay state,
	// so polling and status bar updates continue while forms are open.
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateViewportDims()
		return m, nil
	case quitTimeoutMsg:
		m.quitPending = false
		return m, nil
	case clearActionFlashMsg:
		if m.flashFromAction {
			m.flashErr = ""
			m.flashFromAction = false
		}
		return m, nil

	// Connection lifecycle.
	case connectMsg:
		return m.handleConnect(msg)
	case reconnectMsg:
		return m.handleReconnect(msg)
	case spinner.TickMsg:
		return m.handleSpinnerTick(msg)

	// Polling and data arrival.
	case tickMsg:
		return m.handleTick(msg)
	case overviewDataMsg:
		return m.handleOverviewData(msg)
	case backupsDataMsg:
		return m.handleBackupsData(msg)
	case restoresDataMsg:
		return m.handleRestoresData(msg)
	case configDataMsg:
		return m.handleConfigData(msg)
	case profileYAMLMsg:
		return m.handleProfileYAML(msg)
	case fetchProfileYAMLRequest:
		return m.handleFetchProfileYAMLRequest(msg)

	// Action results.
	case actionResultMsg:
		return m.handleActionResult(msg)
	case editorDoneMsg:
		return m.handleEditorDone(msg)
	case physicalRestoreResultMsg:
		return m.handlePhysicalRestoreResult(msg)

	// Log follow and filter.
	case logFollowMsg:
		return m.handleLogFollow(msg)
	case logFollowDoneMsg:
		return m.handleLogFollowDone(msg)
	case logFilterRequest:
		return m.handleLogFilterRequest(msg)
	case logFilterResultMsg:
		return m.handleLogFilterResult(msg)

	// Overlay and form creation.
	case bulkDeleteRequest:
		return m.handleBulkDeleteRequest(msg)
	case bulkDeleteFormReadyMsg:
		return m.handleBulkDeleteFormReady(msg)
	case backupFormReadyMsg:
		return m.handleBackupFormReady(msg)
	case resyncFormRequest:
		return m.handleResyncFormRequest(msg)
	case setConfigRequest:
		return m.handleSetConfigRequest(msg)
	case removeProfileRequest:
		return m.handleRemoveProfileRequest(msg)
	case editConfigRequest:
		return m.handleEditConfigRequest(msg)
	case editConfigReadyMsg:
		return m.handleEditConfigReady(msg)
	case deleteCheckRequest:
		return m.handleDeleteCheckRequest(msg)
	case canDeleteMsg:
		return m.handleCanDelete(msg)
	case restoreTargetRequest:
		return m.handleRestoreTargetRequest(msg)
	case restoreRequest:
		return m.handleRestoreRequest(msg)
	case physicalRestoreConfirmRequest:
		return m.handlePhysicalRestoreConfirmRequest(msg)
	}

	// Route to the active overlay if one is open.
	if m.activeOverlay != nil {
		next, cmd := m.activeOverlay.Update(msg, m.keys.Back, m.keys.Quit)
		m.activeOverlay = next
		return m, cmd
	}

	// Key messages without an overlay go to the normal key handler.
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		return m.updateKeys(keyMsg)
	}

	return m, nil
}

// updateKeys handles key messages when no form overlay is active.
func (m Model) updateKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// If the help overlay is open, dismiss on ?/esc and ignore everything else.
	if m.showHelp {
		if key.Matches(msg, m.keys.Help) || key.Matches(msg, m.keys.Back) {
			m.showHelp = false
		}
		return m, nil
	}

	var newTab tab = -1
	switch {
	case key.Matches(msg, m.keys.ForceQuit):
		m.overview.stopFollow()
		return m, tea.Quit
	case key.Matches(msg, m.keys.Quit):
		if m.quitPending {
			m.overview.stopFollow()
			return m, tea.Quit
		}
		m.quitPending = true
		return m, quitTimeoutCmd()
	case key.Matches(msg, m.keys.Help):
		m.showHelp = true
		return m, nil
	case key.Matches(msg, m.keys.Tab1):
		newTab = tabOverview
	case key.Matches(msg, m.keys.Tab2):
		newTab = tabBackups
	case key.Matches(msg, m.keys.Tab3):
		newTab = tabConfig
	case key.Matches(msg, backupKeys.Start) && m.client != nil && !m.readonly:
		return m, m.openBackupForm(backupFormQuick)
	case key.Matches(msg, backupKeys.StartCustom) && m.client != nil && !m.readonly:
		return m, m.openBackupForm(backupFormFull)
	case key.Matches(msg, m.keys.PITRToggle) && m.client != nil && !m.readonly:
		pitr := m.overview.data.pitr
		if pitr == nil {
			return m, nil // no status yet
		}
		if pitr.Enabled {
			overlay, cmd := newConfirmOverlay(m.formTheme(),
				"Disable PITR",
				"Stop oplog slicing on all nodes?\nExisting oplog chunks are preserved.",
				"Disable", "Cancel",
				disablePITRCmd(m.ctx, m.client))
			m.activeOverlay = overlay
			return m, cmd
		}
		overlay, cmd := newConfirmOverlay(m.formTheme(),
			"Enable PITR",
			"Start oplog slicing on all nodes?",
			"Enable", "Cancel",
			enablePITRCmd(m.ctx, m.client))
		m.activeOverlay = overlay
		return m, cmd
	case key.Matches(msg, backupKeys.Cancel) && m.client != nil && !m.readonly:
		if m.overview.HasRunningOps() {
			overlay, cmd := newConfirmOverlay(m.formTheme(),
				"Cancel Backup",
				"Cancel the currently running backup?",
				"Cancel Backup", "Keep Running",
				cancelBackupCmd(m.ctx, m.client),
			)
			m.activeOverlay = overlay
			return m, cmd
		}
		return m, nil
	default:
		// No input to sub-models before connection — there's no data to
		// interact with, and sub-models may rely on client/ctx being set.
		if m.client == nil {
			return m, nil
		}
		// Forward to active tab sub-model.
		switch m.activeTab {
		case tabOverview:
			if cmd := m.overview.update(msg, m.keys); cmd != nil {
				return m, cmd
			}
		case tabBackups:
			if cmd := m.backups.update(msg, m.keys, m.readonly); cmd != nil {
				return m, cmd
			}
		case tabConfig:
			if cmd := m.config.update(msg, m.keys, m.readonly); cmd != nil {
				return m, cmd
			}
		}
	}
	// Handle tab switch with immediate data fetch.
	if newTab >= 0 && newTab != m.activeTab {
		m.activeTab = newTab
		return m, tickCmd(0)
	}

	return m, nil
}

// View implements tea.Model.
func (m Model) View() tea.View {
	if m.width == 0 {
		v := tea.NewView("Loading...")
		v.AltScreen = true
		return v
	}

	header := m.headerView()
	bottomBar := m.bottomBarView()

	// Calculate remaining height for content.
	chromeHeight := lipgloss.Height(header) + lipgloss.Height(bottomBar)
	contentHeight := m.height - chromeHeight
	if contentHeight < 0 {
		contentHeight = 0
	}

	content := lipgloss.NewStyle().
		MaxHeight(contentHeight).
		Render(m.contentView(contentHeight))

	v := tea.NewView(lipgloss.JoinVertical(lipgloss.Left,
		header,
		content,
		bottomBar,
	))
	v.AltScreen = true
	return v
}

// headerView renders the tab bar.
func (m Model) headerView() string {
	var tabs []string
	for i := 0; i < int(tabCount); i++ {
		label := fmt.Sprintf("%d:%s", i+1, tabNames[i])
		if tab(i) == m.activeTab {
			tabs = append(tabs, m.styles.ActiveTab.Render(label))
		} else {
			tabs = append(tabs, m.styles.InactiveTab.Render(label))
		}
	}

	title := lipgloss.NewStyle().Bold(true).Padding(0, 1).Render("PBMate")

	var headerParts []string
	headerParts = append(headerParts, title)
	if m.contextName != "" {
		ctxLabel := m.styles.StatusMuted.Render(m.contextName)
		headerParts = append(headerParts, ctxLabel, " ")
	}
	headerParts = append(headerParts, strings.Join(tabs, ""))

	row := lipgloss.JoinHorizontal(lipgloss.Bottom, headerParts...)

	return m.styles.Header.Width(m.width).Render(row)
}

// contentView renders the active tab's content. Panels use viewports that
// produce their allocated height; MaxHeight is a safety net against overflow.
// When a form overlay is active, it renders on top of the current tab content.
func (m Model) contentView(height int) string {
	if m.showHelp {
		return renderHelpOverlay(m.styles, m.width, height, m.readonly)
	}
	if m.activeOverlay != nil {
		return m.activeOverlay.View(m.styles, m.width, height)
	}

	switch m.activeTab {
	case tabOverview:
		return m.overview.view(m.width, height)
	case tabBackups:
		return m.backups.view(m.width, height)
	case tabConfig:
		return m.config.view(m.width, height)
	default:
		return ""
	}
}

// bottomBarView renders the single merged bottom bar with status HUD on the
// left and context-sensitive keybinding hints on the right.
func (m Model) bottomBarView() string {
	// Left zone: operational status HUD.
	var statusParts []string
	if m.readonly {
		statusParts = append(statusParts, m.styles.StatusWarning.Bold(true).Render("READONLY"))
	}
	switch {
	case m.quitPending:
		statusParts = append(statusParts, m.styles.StatusWarning.Render("Press q again to quit"))
	case m.flashErr != "":
		statusParts = append(statusParts, m.styles.StatusError.Render(m.flashErr))
	case m.connecting && m.connectAttempt > 0:
		label := fmt.Sprintf("Connecting (attempt %d) %s", m.connectAttempt+1, m.spinner.View())
		statusParts = append(statusParts, m.styles.StatusWarning.Render(label))
	case m.connecting:
		statusParts = append(statusParts, m.styles.StatusWarning.Render("Connecting "+m.spinner.View()))
	default:
		statusParts = append(statusParts, m.overview.ClusterTimeText())
		statusParts = append(statusParts, m.overview.PITRStatusText())
		statusParts = append(statusParts, m.overview.RunningOpText(m.spinner.View()))
	}
	leftZone := " " + strings.Join(statusParts, "  ")

	// Right zone: context-sensitive keybinding hints, truncated to fit.
	bindings := m.contextBindings()
	const hintPadding = 2 // 1 char padding on each side
	availWidth := m.width - lipgloss.Width(leftZone) - hintPadding
	rightZone := m.renderHints(bindings, availWidth) + " "

	// Compose: left-aligned status, gap, right-aligned hints.
	gap := m.width - lipgloss.Width(leftZone) - lipgloss.Width(rightZone)
	if gap < 0 {
		gap = 0
	}
	bar := leftZone + strings.Repeat(" ", gap) + rightZone

	return m.styles.BottomBar.Width(m.width).Render(bar)
}

// renderHints formats keybinding hints for the bottom bar using
// foreground-only styles. Bindings that exceed maxWidth are dropped.
func (m Model) renderHints(bindings []key.Binding, maxWidth int) string {
	const hintSep = "  "
	var parts []string
	totalWidth := 0

	for _, b := range bindings {
		if !b.Enabled() {
			continue
		}
		keys := b.Help().Key
		desc := b.Help().Desc
		if keys == "" || desc == "" {
			continue
		}
		hint := m.styles.HintKey.Render(keys) + " " + m.styles.HintDesc.Render(desc)
		hintWidth := lipgloss.Width(hint)

		// Account for separator before this hint (if not the first).
		sepWidth := 0
		if len(parts) > 0 {
			sepWidth = lipgloss.Width(hintSep)
		}
		if totalWidth+sepWidth+hintWidth > maxWidth {
			break
		}
		totalWidth += sepWidth + hintWidth
		parts = append(parts, hint)
	}
	return strings.Join(parts, hintSep)
}

// contextBindings returns the keybinding hints for the bottom bar.
// Only essential navigation and help/quit are shown; all other bindings
// are accessible through the ? help overlay.
func (m Model) contextBindings() []key.Binding {
	bindings := []key.Binding{
		m.keys.NextPanel, m.keys.PrevPanel,
		m.keys.Up, m.keys.Down,
	}
	switch m.activeTab {
	case tabBackups:
		bindings = append(bindings, backupKeys.Toggle)
	case tabConfig:
		bindings = append(bindings, configKeys.Toggle)
	}
	bindings = append(bindings, m.keys.Help, m.keys.Quit)
	return bindings
}

// updateViewportDims precomputes all viewport dimensions from the current
// terminal size. This allows Update-time operations (scrolling, GotoBottom)
// to use correct bounds, since View-time dimension setting operates on a
// value copy and doesn't persist.
func (m *Model) updateViewportDims() {
	if m.width == 0 || m.height == 0 {
		return
	}
	chromeH := lipgloss.Height(m.headerView()) + lipgloss.Height(m.bottomBarView())
	contentH := max(m.height-chromeH, 0)

	m.overview.resize(m.width, contentH)
	m.backups.resize(m.width, contentH)
	m.config.resize(m.width, contentH)
}

// openBackupForm fetches storage profiles then creates the form overlay.
// The form is created asynchronously when backupFormReadyMsg arrives.
// The already-fetched backup list is passed through for chain detection.
func (m *Model) openBackupForm(kind backupFormKind) tea.Cmd {
	return fetchBackupFormDataCmd(m.ctx, m.client, kind, m.backups.allBackups())
}
