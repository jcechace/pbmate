package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

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

// Layout constants.
const (
	leftPanelPct  = 30  // left panel width as percentage of terminal width
	minLeftPanelW = 28  // minimum left panel width in characters
	topPanelPct   = 60  // top row height as percentage of content area
	maxLogEntries = 200 // max log entries kept in the follow buffer

	panelBorderH  = 2 // horizontal border: left + right
	panelPaddingH = 2 // horizontal padding: left + right (from Padding(0,1))
	panelBorderV  = 2 // vertical border: top + bottom
)

// confirmAction represents a pending y/n confirmation in the bottom bar.
// Model is the root BubbleTea model for PBMate.
type Model struct {
	client   *sdk.Client // nil until connectMsg arrives
	mongoURI string      // connection URI for background connect

	styles Styles

	activeTab    tab
	width        int
	height       int
	pollInterval time.Duration
	connecting   bool   // true while waiting for the initial connection
	flashErr     string // transient error message for the status bar

	// Confirm form overlay — when non-nil, a confirmation dialog is shown
	// centered over the content area and all key input is routed to it.
	confirmForm       *huh.Form
	confirmFormResult *confirmFormResult
	confirmFormTitle  string
	confirmFormCmd    tea.Cmd // executed if the user confirms

	// Help overlay — when true, the ? help panel is shown.
	showHelp bool

	// Backup form — when non-nil, a huh form overlay is active.
	backupForm       *huh.Form
	backupFormResult *backupFormResult
	backupFormKind   backupFormKind

	// Sub-models.
	overview overviewModel
	backups  backupsModel

	keys globalKeyMap
}

// New creates a new root model with the given theme. The SDK connection
// is established asynchronously — the TUI renders immediately while
// connecting in the background.
func New(uri string, theme Theme) Model {
	s := NewStyles(theme)
	return Model{
		mongoURI:     uri,
		styles:       s,
		activeTab:    tabOverview,
		pollInterval: idleInterval,
		connecting:   true,
		overview:     newOverviewModel(&s),
		backups:      newBackupsModel(&s),
		keys:         globalKeys,
	}
}

// Close disconnects the SDK client if connected. Safe to call when the
// client is nil (e.g. connection never succeeded).
func (m Model) Close() {
	if m.client != nil {
		_ = m.client.Close(context.Background())
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(tea.WindowSize(), connectCmd(m.mongoURI))
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Data and system messages are handled first regardless of overlay state,
	// so polling and status bar updates continue while forms are open.
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateViewportDims()
		return m, nil

	case connectMsg:
		m.connecting = false
		if msg.err != nil {
			m.flashErr = fmt.Sprintf("connect: %v", msg.err)
			return m, nil
		}
		m.client = msg.client
		m.overview.client = msg.client
		return m, tickCmd(0)

	case tickMsg:
		if m.client == nil {
			return m, nil
		}
		// Always fetch overview data (needed for status bar).
		// Additionally fetch tab-specific data.
		cmds := []tea.Cmd{fetchOverviewCmd(m.client, m.overview.isFollowing())}
		if m.activeTab == tabBackups {
			cmds = append(cmds, fetchBackupsCmd(m.client), fetchRestoresCmd(m.client))
		}
		return m, tea.Batch(cmds...)

	case overviewDataMsg:
		m.overview.setData(msg.overviewData)
		if msg.err != nil {
			m.flashErr = fmt.Sprintf("fetch: %v", msg.err)
		} else {
			m.flashErr = ""
		}
		// Adaptive polling: faster when operations are running.
		if len(m.overview.data.operations) > 0 {
			m.pollInterval = activeInterval
		} else {
			m.pollInterval = idleInterval
		}
		return m, tickCmd(m.pollInterval)

	case backupsDataMsg:
		m.backups.setBackupData(msg.backupsData)
		if msg.err != nil {
			m.flashErr = fmt.Sprintf("fetch: %v", msg.err)
		}
		return m, nil

	case restoresDataMsg:
		m.backups.setRestoreData(msg.restoresData)
		if msg.err != nil {
			m.flashErr = fmt.Sprintf("fetch: %v", msg.err)
		}
		return m, nil

	case backupActionMsg:
		if msg.err != nil {
			m.flashErr = fmt.Sprintf("%s failed: %v", msg.action, msg.err)
		} else {
			m.flashErr = ""
		}
		// Trigger immediate re-fetch to pick up the change.
		return m, tickCmd(0)

	case logFollowMsg:
		if msg.err != nil {
			// Follow channel errored; stop following.
			m.overview.stopFollow()
			m.flashErr = fmt.Sprintf("follow: %v", msg.err)
			return m, nil
		}
		m.overview.appendLogEntries(msg.entries)
		// Wait for the next batch from the follow channel.
		return m, m.overview.nextLogCmd()

	case logFollowDoneMsg:
		m.overview.stopFollow()
		if msg.err != nil {
			m.flashErr = fmt.Sprintf("follow: %v", msg.err)
		}
		return m, nil

	case backupFormReadyMsg:
		var form *huh.Form
		var result *backupFormResult
		switch msg.kind {
		case backupFormQuick:
			form, result = newQuickBackupForm()
		case backupFormFull:
			form, result = newFullBackupForm(msg.profiles, nil)
		}
		result.profiles = msg.profiles
		m.backupForm = form
		m.backupFormResult = result
		m.backupFormKind = msg.kind
		return m, m.backupForm.Init()

	case deleteConfirmMsg:
		form, result := newConfirmForm(msg.description, "Delete", "Cancel")
		m.confirmForm = form
		m.confirmFormResult = result
		m.confirmFormTitle = msg.title
		m.confirmFormCmd = deleteBackupCmd(m.client, msg.baseName)
		return m, m.confirmForm.Init()
	}

	// Key messages: route to the active form overlay if one is open,
	// otherwise to the normal key handler.
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if m.backupForm != nil {
			return m.updateBackupForm(keyMsg)
		}
		if m.confirmForm != nil {
			return m.updateConfirmForm(keyMsg)
		}
		return m.updateKeys(keyMsg)
	}

	// Forward non-key messages to the active form (e.g. huh internals).
	if m.backupForm != nil {
		return m.updateBackupForm(msg)
	}
	if m.confirmForm != nil {
		return m.updateConfirmForm(msg)
	}

	return m, nil
}

// updateKeys handles key messages when no form overlay is active.
func (m Model) updateKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// If the help overlay is open, dismiss on ?/esc and ignore everything else.
	if m.showHelp {
		if key.Matches(msg, m.keys.Help) || key.Matches(msg, m.keys.Back) {
			m.showHelp = false
		}
		return m, nil
	}

	var newTab tab = -1
	switch {
	case key.Matches(msg, m.keys.Quit):
		m.overview.stopFollow()
		return m, tea.Quit
	case key.Matches(msg, m.keys.Help):
		m.showHelp = true
		return m, nil
	case key.Matches(msg, m.keys.Tab1):
		newTab = tabOverview
	case key.Matches(msg, m.keys.Tab2):
		newTab = tabBackups
	case key.Matches(msg, m.keys.Tab3):
		newTab = tabConfig
	case key.Matches(msg, backupKeys.Start) && m.client != nil:
		return m, m.openBackupForm(backupFormQuick)
	case key.Matches(msg, backupKeys.StartCustom) && m.client != nil:
		return m, m.openBackupForm(backupFormFull)
	case key.Matches(msg, backupKeys.Cancel) && m.client != nil:
		if len(m.overview.data.operations) > 0 {
			form, result := newConfirmForm(
				"Cancel the currently running backup?",
				"Cancel Backup", "Keep Running",
			)
			m.confirmForm = form
			m.confirmFormResult = result
			m.confirmFormTitle = "Cancel Backup"
			m.confirmFormCmd = cancelBackupCmd(m.client)
			return m, m.confirmForm.Init()
		}
		return m, nil
	default:
		// Forward to active tab sub-model.
		switch m.activeTab {
		case tabOverview:
			if cmd := m.overview.update(msg, m.keys); cmd != nil {
				return m, cmd
			}
		case tabBackups:
			if cmd := m.backups.update(msg, m.keys); cmd != nil {
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
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
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

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		content,
		bottomBar,
	)
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
	row := lipgloss.JoinHorizontal(lipgloss.Bottom,
		title,
		strings.Join(tabs, ""),
	)

	return m.styles.Header.Width(m.width).Render(row)
}

// contentView renders the active tab's content. Panels use viewports that
// produce their allocated height; MaxHeight is a safety net against overflow.
// When a form overlay is active, it renders on top of the current tab content.
func (m Model) contentView(height int) string {
	if m.showHelp {
		return renderHelpOverlay(m.styles, m.width, height)
	}
	if m.backupForm != nil {
		title := "Start Backup"
		if m.backupFormKind == backupFormFull {
			title = "Configure Backup"
		}
		return renderFormOverlay(m.backupForm, title, m.styles, m.width, height)
	}
	if m.confirmForm != nil {
		return renderFormOverlay(m.confirmForm, m.confirmFormTitle, m.styles, m.width, height)
	}

	switch m.activeTab {
	case tabOverview:
		return m.overview.view(m.width, height)
	case tabBackups:
		return m.backups.view(m.width, height)
	case tabConfig:
		return m.placeholderContent("Config - PBM configuration and profiles", height)
	default:
		return ""
	}
}

// placeholderContent renders a simple placeholder for unimplemented tabs.
func (m Model) placeholderContent(text string, height int) string {
	return lipgloss.NewStyle().
		Width(m.width).
		Height(height).
		Render(text)
}

// bottomBarView renders the single merged bottom bar with status HUD on the
// left and context-sensitive keybinding hints on the right.
func (m Model) bottomBarView() string {
	// Left zone: operational status HUD.
	var statusParts []string
	switch {
	case m.flashErr != "":
		statusParts = append(statusParts, m.styles.StatusError.Render(m.flashErr))
	case m.connecting:
		statusParts = append(statusParts, m.styles.StatusWarning.Render("Connecting..."))
	default:
		statusParts = append(statusParts, m.clusterTimeText())
		statusParts = append(statusParts, m.pitrStatusText())
		statusParts = append(statusParts, m.runningOpText())
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

// pitrStatusText returns a short PITR status string for the status bar.
func (m Model) pitrStatusText() string {
	if m.overview.data.pitr == nil {
		return "PITR:--"
	}
	if !m.overview.data.pitr.Enabled {
		return "PITR:off"
	}
	if m.overview.data.pitr.Running {
		return "PITR:on"
	}
	return "PITR:paused"
}

// runningOpText returns a short running operation string for the status bar.
func (m Model) runningOpText() string {
	if len(m.overview.data.operations) == 0 {
		return "Op:none"
	}
	op := m.overview.data.operations[0]
	text := fmt.Sprintf("Op:%s", op.Type)
	if len(m.overview.data.operations) > 1 {
		text += fmt.Sprintf("(+%d)", len(m.overview.data.operations)-1)
	}
	return text
}

// contextBindings returns the keybinding hints for the bottom bar.
// Only essential navigation and help/quit are shown; all other bindings
// are accessible through the ? help overlay.
func (m Model) contextBindings() []key.Binding {
	bindings := []key.Binding{
		m.keys.NextPanel, m.keys.PrevPanel,
		m.keys.Up, m.keys.Down,
	}
	if m.activeTab == tabBackups {
		bindings = append(bindings, backupKeys.Toggle)
	}
	bindings = append(bindings, m.keys.Help, m.keys.Quit)
	return bindings
}

// clusterTimeText returns the cluster time for the status bar.
func (m Model) clusterTimeText() string {
	if m.overview.data.clusterTime.IsZero() {
		return "--:--"
	}
	return m.overview.data.clusterTime.Time().UTC().Format("15:04")
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
}

// --- Backup form management ---

// openBackupForm fetches storage profiles then creates the form overlay.
// The form is created asynchronously when backupFormReadyMsg arrives.
func (m *Model) openBackupForm(kind backupFormKind) tea.Cmd {
	return fetchProfilesCmd(m.client, kind)
}

// updateBackupForm forwards a message to the active backup form and handles
// completion/abort. Data messages are already handled by Update before this
// is called, so only key messages and huh-internal messages arrive here.
func (m Model) updateBackupForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		// Esc or quit dismisses the form.
		if key.Matches(msg, m.keys.Back) || key.Matches(msg, m.keys.Quit) {
			m.backupForm = nil
			m.backupFormResult = nil
			return m, nil
		}
		// 'c' on the quick form transitions to the full wizard.
		if m.backupFormKind == backupFormQuick && msg.String() == "c" {
			return m, m.transitionToFullForm()
		}
	}

	// Forward everything else to the huh form.
	formModel, cmd := m.backupForm.Update(msg)
	if f, ok := formModel.(*huh.Form); ok {
		m.backupForm = f
	}

	// Check if the form completed.
	if m.backupForm.State == huh.StateCompleted {
		result := m.backupFormResult
		// Quick form: "Customize" was selected (confirmed == false).
		if m.backupFormKind == backupFormQuick && !result.confirmed {
			return m, m.transitionToFullForm()
		}
		m.backupForm = nil
		m.backupFormResult = nil
		// Full form: user declined on the final confirm.
		if !result.confirmed {
			return m, nil
		}
		return m, startBackupWithOptsCmd(m.client, result.toOptions())
	}

	// Check if the form was aborted.
	if m.backupForm.State == huh.StateAborted {
		m.backupForm = nil
		m.backupFormResult = nil
		return m, nil
	}

	return m, cmd
}

// transitionToFullForm switches from the quick confirm to the full wizard,
// carrying over the current result values and cached profiles.
func (m *Model) transitionToFullForm() tea.Cmd {
	prev := m.backupFormResult
	form, result := newFullBackupForm(prev.profiles, prev)
	m.backupForm = form
	m.backupFormResult = result
	m.backupFormKind = backupFormFull
	return m.backupForm.Init()
}

// --- Confirm form management ---

// updateConfirmForm forwards a message to the active confirm form and handles
// completion/abort.
func (m Model) updateConfirmForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		// Esc or quit dismisses the confirmation.
		if key.Matches(msg, m.keys.Back) || key.Matches(msg, m.keys.Quit) {
			m.confirmForm = nil
			m.confirmFormResult = nil
			m.confirmFormCmd = nil
			return m, nil
		}
	}

	// Forward to the huh form.
	formModel, cmd := m.confirmForm.Update(msg)
	if f, ok := formModel.(*huh.Form); ok {
		m.confirmForm = f
	}

	// Completed — execute the stored command if confirmed.
	if m.confirmForm.State == huh.StateCompleted {
		confirmed := m.confirmFormResult.confirmed
		actionCmd := m.confirmFormCmd
		m.confirmForm = nil
		m.confirmFormResult = nil
		m.confirmFormCmd = nil
		if confirmed {
			return m, actionCmd
		}
		return m, nil
	}

	// Aborted — dismiss.
	if m.confirmForm.State == huh.StateAborted {
		m.confirmForm = nil
		m.confirmFormResult = nil
		m.confirmFormCmd = nil
		return m, nil
	}

	return m, cmd
}
