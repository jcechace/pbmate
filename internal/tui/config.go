package tui

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// configModel is the sub-model for the Config tab.
type configModel struct {
	// Data from the SDK.
	config     *sdk.Config
	configYAML []byte
	profiles   []sdk.StorageProfile

	// Cached profile YAMLs, keyed by profile name.
	profileYAMLs map[string][]byte

	// UI state.
	cursor int   // 0 = main config, 1..N = profiles[cursor-1]
	focus  panel // panelLeft or panelRight
	styles *Styles

	// Panel viewports.
	listVP   viewport.Model
	detailVP viewport.Model
}

// newConfigModel creates a new config sub-model.
func newConfigModel(styles *Styles) configModel {
	return configModel{
		styles:       styles,
		focus:        panelLeft,
		profileYAMLs: make(map[string][]byte),
		listVP:       newPanelViewport(),
		detailVP:     newPanelViewport(),
	}
}

// itemCount returns the total number of selectable items (main + profiles).
func (m *configModel) itemCount() int {
	n := 0
	if m.config != nil {
		n = 1 // main config
	}
	return n + len(m.profiles)
}

// selectedProfileName returns the name of the selected profile, or "" if the
// main config is selected or no items exist.
func (m *configModel) selectedProfileName() string {
	if m.config != nil {
		// Main config exists: cursor 0 = main, 1..N = profiles.
		if m.cursor == 0 {
			return ""
		}
		idx := m.cursor - 1
		if idx >= 0 && idx < len(m.profiles) {
			return m.profiles[idx].Name.String()
		}
		return ""
	}
	// No main config: cursor maps directly to profiles.
	if m.cursor >= 0 && m.cursor < len(m.profiles) {
		return m.profiles[m.cursor].Name.String()
	}
	return ""
}

// setData receives fresh config data from a poll cycle.
func (m *configModel) setData(d configData) {
	m.config = d.config
	m.configYAML = d.yaml
	m.profiles = d.profiles

	// Sort profiles alphabetically by name.
	sort.Slice(m.profiles, func(i, j int) bool {
		return m.profiles[i].Name.String() < m.profiles[j].Name.String()
	})

	// Clamp cursor if the profile list shrank.
	if m.cursor >= m.itemCount() {
		m.cursor = max(0, m.itemCount()-1)
	}

	m.rebuildListContent()
	m.rebuildDetailContent()
}

// setProfileYAML caches the YAML for a profile and rebuilds the detail
// view if that profile is currently selected.
func (m *configModel) setProfileYAML(name string, yaml []byte) {
	m.profileYAMLs[name] = yaml
	if m.selectedProfileName() == name {
		m.rebuildDetailContent()
	}
}

// needsProfileYAML returns the name of the currently selected profile if
// its YAML has not been cached yet. Returns "" if no fetch is needed.
func (m *configModel) needsProfileYAML() string {
	name := m.selectedProfileName()
	if name == "" {
		return ""
	}
	if _, ok := m.profileYAMLs[name]; ok {
		return ""
	}
	return name
}

// fetchProfileYAMLRequest is emitted by the config model when it needs
// a profile YAML fetched. The root model handles this by dispatching the
// actual SDK call (since the sub-model doesn't hold the client).
type fetchProfileYAMLRequest struct {
	name string
}

// setConfigRequest is emitted by the config tab when the user wants to
// set configuration. Carries the preset target and cached state so the
// overlay can be created without a round-trip fetch.
type setConfigRequest struct {
	initial    *setConfigFormResult
	profiles   []sdk.StorageProfile
	mainExists bool
}

// removeProfileRequest is emitted when the user presses 'x' to delete
// the selected storage profile.
type removeProfileRequest struct {
	name string
}

// resyncFormRequest is emitted by the config tab when the user wants to
// resync storage. Carries the preset target and cached profiles so the
// overlay can be created without a round-trip fetch.
type resyncFormRequest struct {
	initial  *resyncFormResult
	profiles []sdk.StorageProfile
}

// --- Update ---

// update handles key messages for the Config tab.
func (m *configModel) update(msg tea.KeyMsg, keys globalKeyMap) tea.Cmd {
	switch {
	case key.Matches(msg, configKeys.SetConfig):
		return m.emitSetConfigRequest(nil)
	case key.Matches(msg, configKeys.SetConfigSelected):
		return m.emitSetConfigRequest(m.setConfigPresetFromSelection())
	case key.Matches(msg, keys.Delete):
		if name := m.selectedProfileName(); name != "" {
			return func() tea.Msg {
				return removeProfileRequest{name: name}
			}
		}
	case key.Matches(msg, configKeys.Resync):
		return m.emitResyncRequest(nil)
	case key.Matches(msg, configKeys.ResyncSelected):
		return m.emitResyncRequest(m.resyncPresetFromSelection())
	case key.Matches(msg, keys.NextPanel):
		m.cyclePanel(1)
	case key.Matches(msg, keys.PrevPanel):
		m.cyclePanel(-1)
	case key.Matches(msg, keys.Down):
		m.handleVertical(1)
	case key.Matches(msg, keys.Up):
		m.handleVertical(-1)
	}
	return m.requestProfileYAMLIfNeeded()
}

// requestProfileYAMLIfNeeded returns a tea.Cmd that emits a
// fetchProfileYAMLRequest if the selected profile's YAML is not cached.
func (m *configModel) requestProfileYAMLIfNeeded() tea.Cmd {
	name := m.needsProfileYAML()
	if name == "" {
		return nil
	}
	return func() tea.Msg {
		return fetchProfileYAMLRequest{name: name}
	}
}

// cyclePanel moves focus to the next or previous panel.
func (m *configModel) cyclePanel(delta int) {
	old := m.focus
	m.focus = panel((int(m.focus) + delta + int(panelCount)) % int(panelCount))
	if m.focus != old {
		m.rebuildListContent()
	}
}

// handleVertical dispatches Up/Down to the focused panel.
func (m *configModel) handleVertical(delta int) {
	switch m.focus {
	case panelLeft:
		m.moveCursor(delta)
	case panelRight:
		if delta > 0 {
			m.detailVP.ScrollDown(delta)
		} else {
			m.detailVP.ScrollUp(-delta)
		}
	}
}

// moveCursor moves the list cursor by delta. Rebuilds content after moving.
func (m *configModel) moveCursor(delta int) {
	n := m.itemCount()
	if n == 0 {
		return
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= n {
		m.cursor = n - 1
	}
	m.rebuildListContent()
	m.rebuildDetailContent()
}

// --- View ---

// borderColor returns the border color for the given panel.
func (m *configModel) borderColor(p panel) lipgloss.TerminalColor {
	return panelBorderColor(m.focus == p, m.styles)
}

// view renders the Config tab with left list + right detail panels.
func (m *configModel) view(totalW, totalH int) string {
	panelLeftW, panelRightW, contentLeftW, contentRightW := horizontalSplit(totalW)
	innerH := innerHeight(totalH)

	m.listVP.Width = contentLeftW
	m.listVP.Height = innerH
	m.detailVP.Width = contentRightW
	m.detailVP.Height = innerH

	border := m.styles.PanelBorder

	left := renderTitledPanel("Config", m.listVP.View(),
		m.styles.LeftPanel, panelLeftW, innerH, border, m.borderColor(panelLeft))
	right := renderTitledPanel("Detail", m.detailVP.View(),
		m.styles.RightPanel, panelRightW, innerH, border, m.borderColor(panelRight))

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

// resize precomputes viewport dimensions for Update-time scrolling.
func (m *configModel) resize(totalW, totalH int) {
	_, _, contentLeftW, contentRightW := horizontalSplit(totalW)
	innerH := innerHeight(totalH)

	m.listVP.Width = contentLeftW
	m.listVP.Height = innerH
	m.detailVP.Width = contentRightW
	m.detailVP.Height = innerH
}

// --- List content ---

func (m *configModel) rebuildListContent() {
	m.listVP.SetContent(m.listContent())
}

func (m *configModel) listContent() string {
	if m.config == nil && len(m.profiles) == 0 {
		return m.styles.StatusMuted.Render("No configuration")
	}

	// Collect names and types for two-column alignment.
	type entry struct {
		name     string
		typeName string
	}
	var entries []entry

	if m.config != nil {
		entries = append(entries, entry{name: "Main", typeName: m.config.Storage.Type.String()})
	}
	for _, p := range m.profiles {
		entries = append(entries, entry{name: p.Name.String(), typeName: p.Storage.Type.String()})
	}

	// Compute column width from the longest name.
	nameW := 0
	for _, e := range entries {
		if len(e.name) > nameW {
			nameW = len(e.name)
		}
	}

	// Build lines for renderCursorList. The Main entry includes a trailing
	// divider so it visually separates from profiles without being a
	// separate selectable item.
	hasMain := m.config != nil
	hasProfiles := len(m.profiles) > 0
	var lines []string
	for i, e := range entries {
		line := fmt.Sprintf("%-*s  %s", nameW, e.name, m.styles.StatusMuted.Render(e.typeName))
		if hasMain && hasProfiles && i == 0 {
			label := m.styles.StatusMuted.Render("── Profiles ──")
			line += fmt.Sprintf("\n\n  %s\n", label)
		}
		lines = append(lines, line)
	}

	return renderCursorList(lines, m.cursor, m.focus == panelLeft, m.styles)
}

// --- Detail content ---

func (m *configModel) rebuildDetailContent() {
	m.detailVP.SetContent(m.detailContent())
	m.detailVP.GotoTop()
}

func (m *configModel) detailContent() string {
	if m.config == nil && len(m.profiles) == 0 {
		return m.styles.StatusMuted.Render("No selection")
	}

	if m.config != nil && m.cursor == 0 {
		return m.mainConfigDetail()
	}

	// Profile index: offset by 1 when main config exists (cursor 0 = main).
	idx := m.cursor
	if m.config != nil {
		idx = m.cursor - 1
	}
	if idx >= 0 && idx < len(m.profiles) {
		return m.profileDetail(&m.profiles[idx])
	}

	return m.styles.StatusMuted.Render("No selection")
}

// mainConfigDetail renders the main config detail view.
func (m *configModel) mainConfigDetail() string {
	var b strings.Builder

	m.renderStorageSection(&b, &m.config.Storage)
	m.renderPITRSection(&b, m.config.PITR)
	m.renderBackupSection(&b, m.config.Backup)
	m.renderRestoreSection(&b, m.config.Restore)

	if len(m.configYAML) > 0 {
		b.WriteByte('\n')
		m.renderYAMLSection(&b, m.configYAML)
	}

	return b.String()
}

// profileDetail renders a storage profile detail view.
func (m *configModel) profileDetail(p *sdk.StorageProfile) string {
	var b strings.Builder

	b.WriteString(m.styles.SectionHeader.Render("Profile"))
	b.WriteByte('\n')
	fmt.Fprintf(&b, "  Name:    %s\n", p.Name)
	b.WriteByte('\n')

	m.renderStorageSection(&b, &p.Storage)

	yaml, ok := m.profileYAMLs[p.Name.String()]
	if ok && len(yaml) > 0 {
		b.WriteByte('\n')
		m.renderYAMLSection(&b, yaml)
	} else if !ok {
		b.WriteByte('\n')
		b.WriteString(m.styles.StatusMuted.Render("Loading YAML..."))
	}

	return b.String()
}

// --- Detail section renderers ---

// yamlDividerWidth is the character width of the divider line above the YAML section.
const yamlDividerWidth = 40

func (m *configModel) renderYAMLSection(b *strings.Builder, yaml []byte) {
	divider := m.styles.StatusMuted.Render(strings.Repeat("─", yamlDividerWidth))
	b.WriteString(divider)
	b.WriteByte('\n')
	b.WriteString(highlightYAML(yaml, m.styles.ChromaStyle))
}

func (m *configModel) renderStorageSection(b *strings.Builder, s *sdk.StorageConfig) {
	b.WriteString(m.styles.SectionHeader.Render("Storage"))
	b.WriteByte('\n')

	fmt.Fprintf(b, "  Type:    %s\n", valueOrMuted(s.Type.String(), m.styles))
	fmt.Fprintf(b, "  Path:    %s\n", valueOrMuted(s.Path, m.styles))
	fmt.Fprintf(b, "  Region:  %s\n", valueOrMuted(s.Region, m.styles))
}

func (m *configModel) renderPITRSection(b *strings.Builder, pitr *sdk.PITRConfig) {
	b.WriteByte('\n')
	b.WriteString(m.styles.SectionHeader.Render("PITR"))
	b.WriteByte('\n')

	if pitr == nil {
		fmt.Fprintf(b, "  %s\n", m.styles.StatusMuted.Render("Not configured"))
		return
	}

	fmt.Fprintf(b, "  Enabled:      %v\n", pitr.Enabled)
	fmt.Fprintf(b, "  Oplog Only:   %v\n", pitr.OplogOnly)
	if pitr.OplogSpanMin > 0 {
		fmt.Fprintf(b, "  Oplog Span:   %.1f min\n", pitr.OplogSpanMin)
	}
	if !pitr.Compression.IsZero() {
		fmt.Fprintf(b, "  Compression:  %s\n", pitr.Compression)
	}
	if pitr.CompressionLevel != nil {
		fmt.Fprintf(b, "  Compr Level:  %d\n", *pitr.CompressionLevel)
	}
	if len(pitr.Priority) > 0 {
		fmt.Fprintf(b, "  Priority:     %s\n", formatPriorityMap(pitr.Priority))
	}
}

func (m *configModel) renderBackupSection(b *strings.Builder, backup *sdk.BackupConfig) {
	b.WriteByte('\n')
	b.WriteString(m.styles.SectionHeader.Render("Backup"))
	b.WriteByte('\n')

	if backup == nil {
		fmt.Fprintf(b, "  %s\n", m.styles.StatusMuted.Render("Not configured"))
		return
	}

	fmt.Fprintf(b, "  Compression:  %s\n", valueOrMuted(backup.Compression.String(), m.styles))
	if backup.CompressionLevel != nil {
		fmt.Fprintf(b, "  Compr Level:  %d\n", *backup.CompressionLevel)
	}
	if backup.NumParallelCollections > 0 {
		fmt.Fprintf(b, "  Parallel:     %d collections\n", backup.NumParallelCollections)
	}
	if backup.OplogSpanMin > 0 {
		fmt.Fprintf(b, "  Oplog Span:   %.1f min\n", backup.OplogSpanMin)
	}
	if len(backup.Priority) > 0 {
		fmt.Fprintf(b, "  Priority:     %s\n", formatPriorityMap(backup.Priority))
	}
	if backup.Timeouts != nil && backup.Timeouts.StartingStatus != nil {
		fmt.Fprintf(b, "  Start Timeout: %ds\n", *backup.Timeouts.StartingStatus)
	}
}

func (m *configModel) renderRestoreSection(b *strings.Builder, restore *sdk.RestoreConfig) {
	b.WriteByte('\n')
	b.WriteString(m.styles.SectionHeader.Render("Restore"))
	b.WriteByte('\n')

	if restore == nil {
		fmt.Fprintf(b, "  %s\n", m.styles.StatusMuted.Render("Not configured"))
		return
	}

	if restore.NumParallelCollections > 0 {
		fmt.Fprintf(b, "  Parallel:     %d collections\n", restore.NumParallelCollections)
	}
	if restore.NumInsertionWorkers > 0 {
		fmt.Fprintf(b, "  Workers:      %d per collection\n", restore.NumInsertionWorkers)
	}
	if restore.BatchSize > 0 {
		fmt.Fprintf(b, "  Batch Size:   %d\n", restore.BatchSize)
	}
	if restore.NumDownloadWorkers > 0 {
		fmt.Fprintf(b, "  DL Workers:   %d\n", restore.NumDownloadWorkers)
	}
	if restore.MaxDownloadBufferMb > 0 {
		fmt.Fprintf(b, "  DL Buffer:    %dMB\n", restore.MaxDownloadBufferMb)
	}
	if restore.DownloadChunkMb > 0 {
		fmt.Fprintf(b, "  DL Chunk:     %dMB\n", restore.DownloadChunkMb)
	}
	if restore.MongodLocation != "" {
		fmt.Fprintf(b, "  Mongod:       %s\n", restore.MongodLocation)
	}
	if restore.FallbackEnabled != nil {
		fmt.Fprintf(b, "  Fallback:     %v\n", *restore.FallbackEnabled)
	}
	if restore.AllowPartlyDone != nil {
		fmt.Fprintf(b, "  Partly Done:  %v\n", *restore.AllowPartlyDone)
	}
}

// valueOrMuted returns the value if non-empty, or a muted "--" placeholder.
func valueOrMuted(v string, s *Styles) string {
	if v == "" {
		return s.StatusMuted.Render("--")
	}
	return v
}

// formatPriorityMap renders a node priority map as "node:weight, ..." for display.
func formatPriorityMap(m map[string]float64) string {
	if len(m) == 0 {
		return ""
	}
	parts := make([]string, 0, len(m))
	for k, v := range m {
		parts = append(parts, fmt.Sprintf("%s:%.0f", k, v))
	}
	sort.Strings(parts)
	return strings.Join(parts, ", ")
}

// --- Set config helpers ---

// setConfigPresetFromSelection returns a setConfigFormResult preset based on
// the currently selected config item. Returns nil if no items exist.
func (m *configModel) setConfigPresetFromSelection() *setConfigFormResult {
	if m.itemCount() == 0 {
		return nil
	}
	name := m.selectedProfileName()
	if name == "" {
		return &setConfigFormResult{target: setConfigTargetMain}
	}
	return &setConfigFormResult{
		target:  setConfigTargetProfile,
		profile: name,
	}
}

// emitSetConfigRequest returns a command that emits a setConfigRequest with the
// given preset, cached profile list, and whether the main config exists.
func (m *configModel) emitSetConfigRequest(initial *setConfigFormResult) tea.Cmd {
	profiles := m.profiles
	mainExists := m.config != nil
	return func() tea.Msg {
		return setConfigRequest{initial: initial, profiles: profiles, mainExists: mainExists}
	}
}

// --- Resync helpers ---

// resyncPresetFromSelection returns a resyncFormResult preset based on the
// currently selected config item. Returns nil if no items exist.
func (m *configModel) resyncPresetFromSelection() *resyncFormResult {
	if m.itemCount() == 0 {
		return nil
	}
	name := m.selectedProfileName()
	if name == "" {
		// Main config selected.
		return &resyncFormResult{scope: resyncScopeMain}
	}
	return &resyncFormResult{
		scope:       resyncScopeProfile,
		profileName: name,
	}
}

// emitResyncRequest returns a command that emits a resyncFormRequest with the
// given preset and the cached profile list. If initial is nil, defaults are used.
func (m *configModel) emitResyncRequest(initial *resyncFormResult) tea.Cmd {
	profiles := m.profiles
	return func() tea.Msg {
		return resyncFormRequest{initial: initial, profiles: profiles}
	}
}

// --- YAML Syntax Highlighting ---

// highlightYAML applies syntax highlighting to YAML content using Chroma.
// The chromaStyle parameter selects the Chroma color scheme (e.g. "catppuccin-mocha").
// Falls back to plain text if highlighting fails.
func highlightYAML(yamlBytes []byte, chromaStyle string) string {
	lexer := lexers.Get("yaml")
	if lexer == nil {
		return string(yamlBytes)
	}
	lexer = chroma.Coalesce(lexer)

	style := styles.Get(chromaStyle)
	if style == nil {
		style = styles.Fallback
	}

	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	iterator, err := lexer.Tokenise(nil, string(yamlBytes))
	if err != nil {
		return string(yamlBytes)
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, iterator); err != nil {
		return string(yamlBytes)
	}

	// Trim trailing newline that Chroma may add.
	return strings.TrimRight(buf.String(), "\n")
}
