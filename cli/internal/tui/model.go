package tui

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/FareinheitsTemp/opd_panel/cli/internal/client"
	"github.com/FareinheitsTemp/opd_panel/cli/internal/ipc"
)

// ── styles ────────────────────────────────────────────────────────────────────

var (
	styleTitle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	styleTab       = lipgloss.NewStyle().Padding(0, 2).Foreground(lipgloss.Color("240"))
	styleTabActive = lipgloss.NewStyle().Padding(0, 2).Foreground(lipgloss.Color("86")).Bold(true).Underline(true)
	styleSelected  = lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("255")).PaddingLeft(1)
	styleNormal    = lipgloss.NewStyle().PaddingLeft(1)
	styleHelp      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	styleLog       = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	styleBorder    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("238")).Padding(0, 1)
	styleCursor    = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	styleErr       = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	styleOK        = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	styleLabel     = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Width(14)
	styleValue     = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))

	statusColors = map[string]string{
		"running":  "82",
		"stopped":  "240",
		"starting": "220",
		"stopping": "220",
		"crashed":  "196",
	}
)

func colorStatus(s string) string {
	if c, ok := statusColors[s]; ok {
		return lipgloss.NewStyle().Foreground(lipgloss.Color(c)).Render(s)
	}
	return s
}

func fmtUptime(secs uint64) string {
	if secs == 0 {
		return "—"
	}
	h := secs / 3600
	m := (secs % 3600) / 60
	s := secs % 60
	if h > 0 {
		return fmt.Sprintf("%dh%02dm%02ds", h, m, s)
	}
	return fmt.Sprintf("%dm%02ds", m, s)
}

func fmtRAM(b uint64) string {
	if b == 0 {
		return "0 MB"
	}
	return fmt.Sprintf("%d MB", b/1024/1024)
}

// ── messages ──────────────────────────────────────────────────────────────────

type tickMsg struct{}
type serversMsg []ipc.ServerInfo
type sysStatsMsg ipc.SysStats
type logLineMsg string
type actionDoneMsg struct{ err error }
type createDoneMsg struct {
	dir string
	err error
}
type settingsSavedMsg struct{ err error }

// ── tabs ──────────────────────────────────────────────────────────────────────

type tabID int

const (
	tabServers tabID = iota
	tabStats
	tabSystem
	tabCount
)

var tabNames = []string{" Servers ", " Stats ", " System "}

// ── views within Servers tab ──────────────────────────────────────────────────

type srvView int

const (
	srvList srvView = iota
	srvDetail
	srvLogs
	srvConsole
	srvAddForm
	srvSettings
	srvConfirmDelete
)

// ── form field ────────────────────────────────────────────────────────────────

type formField struct {
	Label string
	Value string
	Hint  string
}

// ── model ─────────────────────────────────────────────────────────────────────

type Model struct {
	client  *client.Client
	program *tea.Program

	width  int
	height int
	curTab tabID

	// servers tab
	servers      []ipc.ServerInfo
	cursor       int
	curView      srvView
	activeServer string
	logs         []string
	input        string
	notice       string
	noticeOK     bool

	// add form
	addFields  []formField
	addFocused int

	// settings form
	setFields  []formField
	setFocused int

	// system tab
	sysStats *ipc.SysStats

	// log streaming
	logMu     sync.Mutex
	logCancel context.CancelFunc
}

func NewModel(c *client.Client) *Model {
	return &Model{
		client:    c,
		addFields: defaultAddFields(),
	}
}

func defaultAddFields() []formField {
	return []formField{
		{Label: "ID", Hint: "e.g. survival"},
		{Label: "Name", Hint: "display name"},
		{Label: "Port", Value: "25565", Hint: "1-65535"},
		{Label: "RAM min MB", Value: "512", Hint: "min heap"},
		{Label: "RAM max MB", Value: "2048", Hint: "max heap"},
		{Label: "Jar", Value: "server.jar", Hint: "filename"},
	}
}

func (m *Model) SetProgram(p *tea.Program) { m.program = p }

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.fetchServers(), tickEvery(2*time.Second))
}

func tickEvery(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return tickMsg{} })
}

func (m *Model) fetchServers() tea.Cmd {
	return func() tea.Msg {
		servers, _ := m.client.List()
		return serversMsg(servers)
	}
}

func (m *Model) fetchSysStats() tea.Cmd {
	return func() tea.Msg {
		s, err := m.client.SysStats()
		if err != nil || s == nil {
			return nil
		}
		return sysStatsMsg(*s)
	}
}

func (m *Model) selectedServer() *ipc.ServerInfo {
	if len(m.servers) == 0 || m.cursor >= len(m.servers) {
		return nil
	}
	return &m.servers[m.cursor]
}

func (m *Model) selectedID() string {
	if s := m.selectedServer(); s != nil {
		return s.ID
	}
	return ""
}

func (m *Model) cancelLogStream() {
	m.logMu.Lock()
	defer m.logMu.Unlock()
	if m.logCancel != nil {
		m.logCancel()
		m.logCancel = nil
	}
}

func (m *Model) setNotice(msg string, ok bool) {
	m.notice = msg
	m.noticeOK = ok
}

// ── Update ────────────────────────────────────────────────────────────────────

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

	case tickMsg:
		cmds := []tea.Cmd{m.fetchServers(), tickEvery(2 * time.Second)}
		if m.curTab == tabSystem {
			cmds = append(cmds, m.fetchSysStats())
		}
		return m, tea.Batch(cmds...)

	case serversMsg:
		m.servers = []ipc.ServerInfo(msg)
		if m.cursor >= len(m.servers) && len(m.servers) > 0 {
			m.cursor = len(m.servers) - 1
		}
		if len(m.servers) == 0 {
			m.cursor = 0
		}

	case sysStatsMsg:
		s := ipc.SysStats(msg)
		m.sysStats = &s

	case logLineMsg:
		m.logs = append(m.logs, string(msg))
		const maxLogs = 500
		if len(m.logs) > maxLogs {
			m.logs = m.logs[len(m.logs)-maxLogs:]
		}
		return m, nil

	case actionDoneMsg:
		if msg.err != nil {
			m.setNotice("✗ "+msg.err.Error(), false)
		} else {
			m.setNotice("✔ done", true)
		}
		return m, m.fetchServers()

	case createDoneMsg:
		if msg.err != nil {
			m.setNotice("✗ "+msg.err.Error(), false)
		} else {
			m.setNotice("✔ created → "+msg.dir, true)
			m.curView = srvList
			m.addFields = defaultAddFields()
			m.addFocused = 0
		}
		return m, m.fetchServers()

	case settingsSavedMsg:
		if msg.err != nil {
			m.setNotice("✗ "+msg.err.Error(), false)
		} else {
			m.setNotice("✔ settings saved", true)
			m.curView = srvList
		}
		return m, m.fetchServers()

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.cancelLogStream()
			return m, tea.Quit
		case "tab":
			if m.curView == srvList || m.curView == srvDetail {
				m.curTab = (m.curTab + 1) % tabCount
				if m.curTab == tabSystem {
					return m, m.fetchSysStats()
				}
				return m, nil
			}
		case "1":
			if m.curView == srvList || m.curView == srvDetail {
				m.curTab = tabServers
				return m, nil
			}
		case "2":
			if m.curView == srvList || m.curView == srvDetail {
				m.curTab = tabStats
				return m, nil
			}
		case "3":
			if m.curView == srvList || m.curView == srvDetail {
				m.curTab = tabSystem
				return m, m.fetchSysStats()
			}
		}

		switch m.curTab {
		case tabServers:
			switch m.curView {
			case srvList:
				return m.updateList(msg)
			case srvDetail:
				return m.updateDetail(msg)
			case srvLogs:
				return m.updateLogs(msg)
			case srvConsole:
				return m.updateConsole(msg)
			case srvAddForm:
				return m.updateAddForm(msg)
			case srvSettings:
				return m.updateSettings(msg)
			case srvConfirmDelete:
				return m.updateConfirmDelete(msg)
			}
		}
	}
	return m, nil
}

// ── Servers tab key handlers ───────────────────────────────────────────────────

func (m *Model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.notice = ""
	switch msg.String() {
	case "q":
		m.cancelLogStream()
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if len(m.servers) > 0 && m.cursor < len(m.servers)-1 {
			m.cursor++
		}
	case "enter":
		if m.selectedServer() != nil {
			m.curView = srvDetail
		}
	case "s":
		if id := m.selectedID(); id != "" {
			return m, func() tea.Msg {
				_, err := m.client.Start(id)
				return actionDoneMsg{err}
			}
		}
	case "x":
		if id := m.selectedID(); id != "" {
			return m, func() tea.Msg {
				_, err := m.client.Stop(id)
				return actionDoneMsg{err}
			}
		}
	case "r":
		if id := m.selectedID(); id != "" {
			return m, func() tea.Msg {
				_, err := m.client.Restart(id)
				return actionDoneMsg{err}
			}
		}
	case "l":
		if id := m.selectedID(); id != "" {
			m.cancelLogStream()
			m.activeServer = id
			m.curView = srvLogs
			m.logs = nil
			return m, m.startLogStream(id)
		}
	case "c":
		if id := m.selectedID(); id != "" {
			m.cancelLogStream()
			m.activeServer = id
			m.curView = srvConsole
			m.input = ""
			m.logs = nil
			return m, m.startLogStream(id)
		}
	case "a":
		m.addFields = defaultAddFields()
		m.addFocused = 0
		m.curView = srvAddForm
	case "e":
		if s := m.selectedServer(); s != nil {
			m.setFields = []formField{
				{Label: "Name", Value: s.Name},
				{Label: "Port", Value: fmt.Sprintf("%d", s.Port)},
				{Label: "RAM max MB", Value: fmt.Sprintf("%d", s.RAMMax/1024/1024)},
			}
			m.setFocused = 0
			m.activeServer = s.ID
			m.curView = srvSettings
		}
	case "d":
		if m.selectedServer() != nil {
			m.curView = srvConfirmDelete
		}
	}
	return m, nil
}

func (m *Model) updateDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "backspace":
		m.curView = srvList
	case "l":
		if s := m.selectedServer(); s != nil {
			m.cancelLogStream()
			m.activeServer = s.ID
			m.curView = srvLogs
			m.logs = nil
			return m, m.startLogStream(s.ID)
		}
	case "c":
		if s := m.selectedServer(); s != nil {
			m.cancelLogStream()
			m.activeServer = s.ID
			m.curView = srvConsole
			m.input = ""
			m.logs = nil
			return m, m.startLogStream(s.ID)
		}
	case "s":
		if s := m.selectedServer(); s != nil {
			id := s.ID
			return m, func() tea.Msg {
				_, err := m.client.Start(id)
				return actionDoneMsg{err}
			}
		}
	case "x":
		if s := m.selectedServer(); s != nil {
			id := s.ID
			return m, func() tea.Msg {
				_, err := m.client.Stop(id)
				return actionDoneMsg{err}
			}
		}
	case "r":
		if s := m.selectedServer(); s != nil {
			id := s.ID
			return m, func() tea.Msg {
				_, err := m.client.Restart(id)
				return actionDoneMsg{err}
			}
		}
	}
	return m, nil
}

func (m *Model) updateLogs(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "q" || msg.String() == "esc" {
		m.cancelLogStream()
		m.curView = srvList
		m.logs = nil
		m.activeServer = ""
	}
	return m, nil
}

func (m *Model) updateConsole(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.cancelLogStream()
		m.curView = srvList
		m.input = ""
		m.activeServer = ""
	case "enter":
		if m.input != "" && m.activeServer != "" {
			id := m.activeServer
			cmd := m.input
			m.input = ""
			return m, func() tea.Msg {
				m.client.SendCommand(id, cmd)
				return nil
			}
		}
	case "backspace":
		if len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
		}
	default:
		if len(msg.Runes) > 0 {
			m.input += string(msg.Runes)
		}
	}
	return m, nil
}

func (m *Model) updateAddForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.curView = srvList
	case "tab", "down":
		if m.addFocused < len(m.addFields)-1 {
			m.addFocused++
		}
	case "shift+tab", "up":
		if m.addFocused > 0 {
			m.addFocused--
		}
	case "enter":
		if m.addFocused < len(m.addFields)-1 {
			m.addFocused++
		} else {
			return m, m.submitAddForm()
		}
	case "backspace":
		f := &m.addFields[m.addFocused]
		if len(f.Value) > 0 {
			f.Value = f.Value[:len(f.Value)-1]
		}
	default:
		if len(msg.Runes) > 0 {
			m.addFields[m.addFocused].Value += string(msg.Runes)
		}
	}
	return m, nil
}

func (m *Model) submitAddForm() tea.Cmd {
	f := m.addFields
	getInt := func(s string, def int) int {
		var v int
		if _, err := fmt.Sscanf(s, "%d", &v); err != nil || v == 0 {
			return def
		}
		return v
	}
	cr := ipc.CreateRequest{
		ID:       strings.TrimSpace(f[0].Value),
		Name:     strings.TrimSpace(f[1].Value),
		Port:     getInt(f[2].Value, 25565),
		RAMMinMB: getInt(f[3].Value, 512),
		RAMMaxMB: getInt(f[4].Value, 2048),
		Jar:      strings.TrimSpace(f[5].Value),
	}
	if cr.Jar == "" {
		cr.Jar = "server.jar"
	}
	return func() tea.Msg {
		dir, err := m.client.Create(cr)
		return createDoneMsg{dir, err}
	}
}

func (m *Model) updateSettings(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.curView = srvList
	case "tab", "down":
		if m.setFocused < len(m.setFields)-1 {
			m.setFocused++
		}
	case "shift+tab", "up":
		if m.setFocused > 0 {
			m.setFocused--
		}
	case "enter":
		if m.setFocused < len(m.setFields)-1 {
			m.setFocused++
		} else {
			return m, m.submitSettings()
		}
	case "backspace":
		f := &m.setFields[m.setFocused]
		if len(f.Value) > 0 {
			f.Value = f.Value[:len(f.Value)-1]
		}
	default:
		if len(msg.Runes) > 0 {
			m.setFields[m.setFocused].Value += string(msg.Runes)
		}
	}
	return m, nil
}

func (m *Model) submitSettings() tea.Cmd {
	f := m.setFields
	getInt := func(s string, def int) int {
		var v int
		if _, err := fmt.Sscanf(s, "%d", &v); err != nil || v == 0 {
			return def
		}
		return v
	}
	us := ipc.UpdateSettingsRequest{
		ServerID: m.activeServer,
		Name:     strings.TrimSpace(f[0].Value),
		Port:     getInt(f[1].Value, 25565),
		RAMMaxMB: getInt(f[2].Value, 2048),
	}
	return func() tea.Msg {
		err := m.client.UpdateSettings(us)
		return settingsSavedMsg{err}
	}
}

func (m *Model) updateConfirmDelete(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if id := m.selectedID(); id != "" {
			m.curView = srvList
			return m, func() tea.Msg {
				_, err := m.client.Remove(id)
				return actionDoneMsg{err}
			}
		}
	case "n", "N", "esc":
		m.curView = srvList
	}
	return m, nil
}

// startLogStream opens a log subscription and pumps lines via p.Send().
func (m *Model) startLogStream(id string) tea.Cmd {
	return func() tea.Msg {
		ch, err := m.client.StreamLogs(id)
		if err != nil {
			return nil
		}

		ctx, cancel := context.WithCancel(context.Background())
		m.logMu.Lock()
		m.logCancel = cancel
		m.logMu.Unlock()

		go func() {
			defer cancel()
			for {
				select {
				case line, ok := <-ch:
					if !ok {
						return
					}
					if m.program != nil {
						m.program.Send(logLineMsg(line))
					}
				case <-ctx.Done():
					return
				}
			}
		}()
		return nil
	}
}

// ── View ──────────────────────────────────────────────────────────────────────

func (m *Model) View() string {
	var sb strings.Builder
	sb.WriteString(m.renderTabBar() + "\n")

	switch m.curTab {
	case tabServers:
		sb.WriteString(m.viewServers())
	case tabStats:
		sb.WriteString(m.viewStats())
	case tabSystem:
		sb.WriteString(m.viewSystem())
	}

	if m.notice != "" {
		var ns lipgloss.Style
		if m.noticeOK {
			ns = styleOK
		} else {
			ns = styleErr
		}
		sb.WriteString("\n" + ns.Render(m.notice))
	}

	return sb.String()
}

func (m *Model) renderTabBar() string {
	var parts []string
	for i, name := range tabNames {
		if tabID(i) == m.curTab {
			parts = append(parts, styleTabActive.Render(name))
		} else {
			parts = append(parts, styleTab.Render(name))
		}
	}
	return styleTitle.Render(" ◈ OPD ") + "  " + strings.Join(parts, "│") + "  " + styleHelp.Render("Tab/1-3 switch")
}

// ── Servers tab views ─────────────────────────────────────────────────────────

func (m *Model) viewServers() string {
	switch m.curView {
	case srvDetail:
		return m.viewDetail()
	case srvLogs:
		return m.viewLogs()
	case srvConsole:
		return m.viewConsole()
	case srvAddForm:
		return m.viewAddForm()
	case srvSettings:
		return m.viewSettingsForm()
	case srvConfirmDelete:
		return m.viewConfirmDelete()
	default:
		return m.viewList()
	}
}

func (m *Model) viewList() string {
	var sb strings.Builder

	if len(m.servers) == 0 {
		sb.WriteString(styleNormal.Render("No servers found. Press 'a' to add one.") + "\n")
	} else {
		header := fmt.Sprintf("  %-16s %-10s %-20s %-10s %-16s %s",
			"ID", "STATUS", "NAME", "CPU", "RAM", "PORT")
		sb.WriteString(styleHelp.Render(header) + "\n")
		for i, s := range m.servers {
			ram := fmt.Sprintf("%s/%s", fmtRAM(s.RAMUsed), fmtRAM(s.RAMMax))
			cpu := fmt.Sprintf("%.1f%%", s.CPU)
			line := fmt.Sprintf("%-16s %-18s %-20s %-10s %-16s :%d",
				s.ID, colorStatus(s.Status), s.Name, cpu, ram, s.Port)
			if i == m.cursor {
				sb.WriteString(styleSelected.Render("▶ "+line) + "\n")
			} else {
				sb.WriteString(styleNormal.Render("  "+line) + "\n")
			}
		}
	}

	sb.WriteString("\n" + styleHelp.Render("↑/↓ navigate  Enter details  s start  x stop  r restart  l logs  c console  a add  e edit  d delete  q quit"))
	return styleBorder.Render(sb.String())
}

func (m *Model) viewDetail() string {
	s := m.selectedServer()
	if s == nil {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(styleTitle.Render(fmt.Sprintf(" ◈ %s ", s.Name)) + "\n\n")
	row := func(label, value string) {
		sb.WriteString(styleLabel.Render(label+":") + " " + styleValue.Render(value) + "\n")
	}
	row("ID", s.ID)
	row("Status", colorStatus(s.Status))
	row("Port", fmt.Sprintf("%d", s.Port))
	row("PID", fmt.Sprintf("%d", s.PID))
	row("Uptime", fmtUptime(s.Uptime))
	row("RAM used", fmtRAM(s.RAMUsed))
	row("RAM max", fmtRAM(s.RAMMax))
	row("CPU", fmt.Sprintf("%.1f%%", s.CPU))
	sb.WriteString("\n" + styleHelp.Render("s start  x stop  r restart  l logs  c console  esc back"))
	return styleBorder.Render(sb.String())
}

func (m *Model) viewLogs() string {
	var sb strings.Builder
	sb.WriteString(styleTitle.Render(fmt.Sprintf(" ◈ Logs — %s ", m.activeServer)) + "\n")
	for _, line := range m.visibleLines(m.height - 8) {
		sb.WriteString(styleLog.Render(line) + "\n")
	}
	sb.WriteString(styleHelp.Render("esc / q — back"))
	return styleBorder.Render(sb.String())
}

func (m *Model) viewConsole() string {
	var sb strings.Builder
	sb.WriteString(styleTitle.Render(fmt.Sprintf(" ◈ Console — %s ", m.activeServer)) + "\n")
	for _, line := range m.visibleLines(m.height - 10) {
		sb.WriteString(styleLog.Render(line) + "\n")
	}
	sb.WriteString("\n" + styleCursor.Render("> ") + m.input + "█")
	sb.WriteString(styleHelp.Render("\nenter send  esc back"))
	return styleBorder.Render(sb.String())
}

func (m *Model) viewAddForm() string {
	var sb strings.Builder
	sb.WriteString(styleTitle.Render(" ◈ Add Server ") + "\n\n")
	for i, f := range m.addFields {
		var val string
		if i == m.addFocused {
			val = styleCursor.Render(f.Value + "█")
		} else {
			val = styleValue.Render(f.Value)
		}
		hint := ""
		if f.Hint != "" {
			hint = " " + styleHelp.Render("("+f.Hint+")")
		}
		sb.WriteString(styleLabel.Render(f.Label+":") + " " + val + hint + "\n")
	}
	sb.WriteString("\n" + styleHelp.Render("Tab/↓ next  Shift+Tab/↑ prev  Enter next/submit  esc cancel"))
	return styleBorder.Render(sb.String())
}

func (m *Model) viewSettingsForm() string {
	var sb strings.Builder
	sb.WriteString(styleTitle.Render(fmt.Sprintf(" ◈ Settings — %s ", m.activeServer)) + "\n\n")
	for i, f := range m.setFields {
		var val string
		if i == m.setFocused {
			val = styleCursor.Render(f.Value + "█")
		} else {
			val = styleValue.Render(f.Value)
		}
		sb.WriteString(styleLabel.Render(f.Label+":") + " " + val + "\n")
	}
	sb.WriteString("\n" + styleHelp.Render("Tab/↓ next  Shift+Tab/↑ prev  Enter next/save  esc cancel"))
	return styleBorder.Render(sb.String())
}

func (m *Model) viewConfirmDelete() string {
	s := m.selectedServer()
	if s == nil {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(styleTitle.Render(" ◈ Confirm Delete ") + "\n\n")
	sb.WriteString(styleErr.Render(fmt.Sprintf("Delete server '%s' and ALL its files?", s.ID)) + "\n\n")
	sb.WriteString(styleHelp.Render("y — yes, delete    n / esc — cancel"))
	return styleBorder.Render(sb.String())
}

// ── Stats tab ─────────────────────────────────────────────────────────────────

func (m *Model) viewStats() string {
	var sb strings.Builder
	sb.WriteString(styleTitle.Render(" ◈ Daemon & Agent Status ") + "\n\n")

	if err := m.client.Ping(); err != nil {
		sb.WriteString(styleErr.Render("✗ Daemon: "+err.Error()) + "\n")
	} else {
		sb.WriteString(styleOK.Render("✔ Daemon running on 127.0.0.1:51200") + "\n")
	}

	sb.WriteString("\n")
	running := 0
	for _, s := range m.servers {
		if s.Status == "running" {
			running++
		}
	}
	if len(m.servers) > 0 {
		sb.WriteString(styleValue.Render(fmt.Sprintf("%d server(s) tracked", len(m.servers))) + "\n")
		sb.WriteString(styleOK.Render(fmt.Sprintf("%d running", running)) + "\n")
	} else {
		sb.WriteString(styleHelp.Render("No servers registered") + "\n")
	}

	sb.WriteString("\n" + styleHelp.Render("Tab to switch tabs"))
	return styleBorder.Render(sb.String())
}

// ── System tab ────────────────────────────────────────────────────────────────

func (m *Model) viewSystem() string {
	var sb strings.Builder
	sb.WriteString(styleTitle.Render(" ◈ System Resources ") + "\n\n")

	if m.sysStats == nil {
		sb.WriteString(styleHelp.Render("Loading...") + "\n")
	} else {
		s := m.sysStats
		row := func(label, value string) {
			sb.WriteString(styleLabel.Render(label+":") + " " + styleValue.Render(value) + "\n")
		}
		row("CPU", fmt.Sprintf("%.1f%%", s.CPUPercent))
		row("RAM used", fmt.Sprintf("%d MB / %d MB (%.0f%%)",
			s.RAMUsedMB, s.RAMTotalMB, percent(s.RAMUsedMB, s.RAMTotalMB)))
		row("Disk used", fmt.Sprintf("%.1f GB / %.1f GB (%.0f%%)",
			s.DiskUsedGB, s.DiskTotalGB, percentF(s.DiskUsedGB, s.DiskTotalGB)))
		sb.WriteString("\n")
		sb.WriteString(barLine("CPU ", s.CPUPercent, 100, 40) + "\n")
		sb.WriteString(barLine("RAM ", float64(s.RAMUsedMB), float64(s.RAMTotalMB), 40) + "\n")
		sb.WriteString(barLine("Disk", s.DiskUsedGB, s.DiskTotalGB, 40) + "\n")
	}

	sb.WriteString("\n" + styleHelp.Render("Auto-refresh every 2s  Tab to switch tabs"))
	return styleBorder.Render(sb.String())
}

func percent(used, total uint64) float64 {
	if total == 0 {
		return 0
	}
	return float64(used) / float64(total) * 100
}

func percentF(used, total float64) float64 {
	if total == 0 {
		return 0
	}
	return used / total * 100
}

func barLine(label string, used, total float64, width int) string {
	if total == 0 {
		total = 1
	}
	filled := int(used / total * float64(width))
	if filled > width {
		filled = width
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	pct := used / total * 100
	return styleLabel.Render(label) + " " +
		lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render(bar) +
		" " + styleHelp.Render(fmt.Sprintf("%.0f%%", pct))
}

func (m *Model) visibleLines(n int) []string {
	if n < 1 {
		n = 1
	}
	if len(m.logs) <= n {
		return m.logs
	}
	return m.logs[len(m.logs)-n:]
}
