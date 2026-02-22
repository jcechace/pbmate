package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
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

// Model is the root BubbleTea model for PBMate.
type Model struct {
	client   *sdk.Client // nil until connectMsg arrives
	mongoURI string      // connection URI for background connect
	ctx      context.Context
	cancel   context.CancelFunc

	styles Styles

	activeTab    tab
	width        int
	height       int
	pollInterval time.Duration
	connecting   bool   // true while waiting for the initial connection
	flashErr     string // transient error message for the status bar

	// activeOverlay captures all input when non-nil. Overlays include
	// backup forms, file pickers, profile name forms, and confirm dialogs.
	activeOverlay formOverlay

	// Help overlay — when true, the ? help panel is shown.
	showHelp bool

	// Sub-models.
	overview overviewModel
	backups  backupsModel
	config   configModel

	keys globalKeyMap
}

// New creates a new root model with the given theme. The SDK connection
// is established asynchronously — the TUI renders immediately while
// connecting in the background.
func New(uri string, theme Theme) Model {
	s := NewStyles(theme)
	ctx, cancel := context.WithCancel(context.Background())
	return Model{
		mongoURI:     uri,
		ctx:          ctx,
		cancel:       cancel,
		styles:       s,
		activeTab:    tabOverview,
		pollInterval: idleInterval,
		connecting:   true,
		overview:     newOverviewModel(&s),
		backups:      newBackupsModel(&s),
		config:       newConfigModel(&s),
		keys:         globalKeys,
	}
}

// setFlash sets or clears the transient error message in the status bar.
// On success (err == nil) the message is cleared. On failure the prefix
// and error are combined into a flash message.
func (m *Model) setFlash(prefix string, err error) {
	if err != nil {
		m.flashErr = fmt.Sprintf("%s: %v", prefix, err)
	} else {
		m.flashErr = ""
	}
}

// Close cancels the root context and disconnects the SDK client.
// Safe to call when the client is nil (e.g. connection never succeeded).
func (m Model) Close() {
	m.cancel()
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
		m.overview.ctx = m.ctx
		m.overview.client = msg.client
		return m, tickCmd(0)

	case tickMsg:
		if m.client == nil {
			return m, nil
		}
		// Always fetch overview data (needed for status bar).
		// Additionally fetch tab-specific data.
		cmds := []tea.Cmd{fetchOverviewCmd(m.ctx, m.client, m.overview.isFollowing())}
		if m.activeTab == tabBackups {
			cmds = append(cmds, fetchBackupsCmd(m.ctx, m.client), fetchRestoresCmd(m.ctx, m.client))
		}
		if m.activeTab == tabConfig {
			cmds = append(cmds, fetchConfigCmd(m.ctx, m.client))
		}
		return m, tea.Batch(cmds...)

	case overviewDataMsg:
		m.overview.setData(msg.overviewData)
		m.setFlash("fetch", msg.err)
		// Adaptive polling: faster when operations are running.
		if len(m.overview.data.operations) > 0 {
			m.pollInterval = activeInterval
		} else {
			m.pollInterval = idleInterval
		}
		return m, tickCmd(m.pollInterval)

	case backupsDataMsg:
		m.backups.setBackupData(msg.backupsData)
		m.setFlash("fetch", msg.err)
		return m, nil

	case restoresDataMsg:
		m.backups.setRestoreData(msg.restoresData)
		m.setFlash("fetch", msg.err)
		return m, nil

	case backupActionMsg:
		m.setFlash(msg.action, msg.err)
		// Trigger immediate re-fetch to pick up the change.
		return m, tickCmd(0)

	case logFollowMsg:
		// Discard messages from a stale follow session.
		if msg.session != m.overview.logFollowSession {
			return m, nil
		}
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
		// Discard done messages from a stale follow session.
		if msg.session != m.overview.logFollowSession {
			return m, nil
		}
		m.overview.stopFollow()
		if msg.err != nil {
			m.flashErr = fmt.Sprintf("follow: %v", msg.err)
		}
		return m, nil

	case backupFormReadyMsg:
		overlay, cmd := newBackupFormOverlay(m.ctx, m.client, msg.kind, msg.profiles)
		m.activeOverlay = overlay
		return m, cmd

	case configDataMsg:
		m.config.setData(msg.configData)
		m.setFlash("fetch", msg.err)
		// Trigger lazy profile YAML fetch if the selected profile is uncached.
		if name := m.config.needsProfileYAML(); name != "" {
			return m, fetchProfileYAMLCmd(m.ctx, m.client, name)
		}
		return m, nil

	case profileYAMLMsg:
		m.setFlash("fetch", msg.err)
		if msg.err == nil {
			m.config.setProfileYAML(msg.name, msg.yaml)
		}
		return m, nil

	case fetchProfileYAMLRequest:
		if m.client != nil {
			return m, fetchProfileYAMLCmd(m.ctx, m.client, msg.name)
		}
		return m, nil

	case configApplyRequest:
		var title string
		if msg.profileName == "" {
			title = "Select YAML \u2500 Main"
		} else {
			title = "Select YAML \u2500 " + msg.profileName
		}
		overlay, cmd := newFilePickerOverlay(m.ctx, m.client, msg.profileName, false, title)
		m.activeOverlay = overlay
		return m, cmd

	case configNewProfileRequest:
		overlay, cmd := newProfileNameOverlay(m.ctx, m.client)
		m.activeOverlay = overlay
		return m, cmd

	case configActionMsg:
		m.setFlash(msg.action, msg.err)
		// Clear cached profile YAMLs so they are re-fetched.
		m.config.profileYAMLs = make(map[string][]byte)
		return m, tickCmd(0)

	case deleteConfirmMsg:
		overlay, cmd := newConfirmOverlay(msg.title, msg.description, "Delete", "Cancel",
			deleteBackupCmd(m.ctx, m.client, msg.baseName))
		m.activeOverlay = overlay
		return m, cmd
	}

	// Route to the active overlay if one is open.
	if m.activeOverlay != nil {
		next, cmd := m.activeOverlay.Update(msg, m.keys.Back, m.keys.Quit)
		m.activeOverlay = next
		return m, cmd
	}

	// Key messages without an overlay go to the normal key handler.
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		return m.updateKeys(keyMsg)
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
			overlay, cmd := newConfirmOverlay(
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
		case tabConfig:
			if cmd := m.config.update(msg, m.keys); cmd != nil {
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
		return renderHelpOverlay(&m.styles, m.width, height)
	}
	if m.activeOverlay != nil {
		return m.activeOverlay.View(&m.styles, m.width, height)
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
	m.config.resize(m.width, contentH)
}

// openBackupForm fetches storage profiles then creates the form overlay.
// The form is created asynchronously when backupFormReadyMsg arrives.
func (m *Model) openBackupForm(kind backupFormKind) tea.Cmd {
	return fetchProfilesCmd(m.ctx, m.client, kind)
}
