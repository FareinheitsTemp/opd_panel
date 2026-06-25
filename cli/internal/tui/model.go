package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/FareinheitsTemp/opd_panel/cli/internal/client"
	"github.com/FareinheitsTemp/opd_panel/cli/internal/ipc"
)

// ── styles ──────────────────────────────────────────────────────────────────

var (
	styleTitle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).MarginBottom(1)
	styleSelected = lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("255")).PaddingLeft(1)
	styleNormal = lipgloss.NewStyle().PaddingLeft(1)
	styleHelp = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).MarginTop(1)
	styleLog = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	styleBorder = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("238")).Padding(0, 1)
	styleCursor = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))

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

// ── messages ─────────────────────────────────────────────────────────────────

type tickMsg struct{}
type serversMsg []ipc.ServerInfo
type logLineMsg string

// ── model ────────────────────────────────────────────────────────────────────

type view int

const (
	viewList view = iota
	viewLogs
	viewConsole
)

type Model struct {
	client    *client.Client
	program   *tea.Program // set after p.Run(); used to send messages from goroutines
	servers   []ipc.ServerInfo
	cursor    int
	curView   view
	logs      []string
	input     string
	width     int
	height    int
}

func NewModel(c *client.Client) *Model {
	return &Model{client: c}
}

// SetProgram must be called right after tea.NewProgram, before p.Run().
func (m *Model) SetProgram(p *tea.Program) {
	m.program = p
}

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

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

	case tickMsg:
		return m, tea.Batch(m.fetchServers(), tickEvery(2*time.Second))

	case serversMsg:
		m.servers = []ipc.ServerInfo(msg)
		if m.cursor >= len(m.servers) && len(m.servers) > 0 {
			m.cursor = len(m.servers) - 1
		}

	case logLineMsg:
		m.logs = append(m.logs, string(msg))
		if len(m.logs) > 500 {
			m.logs = m.logs[len(m.logs)-500:]
		}
		return m, nil

	case tea.KeyMsg:
		switch m.curView {
		case viewList:
			return m.updateList(msg)
		case viewLogs:
			return m.updateLogs(msg)
		case viewConsole:
			return m.updateConsole(msg)
		}
	}
	return m, nil
}

func (m *Model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.servers)-1 {
			m.cursor++
		}
	case "s":
		if len(m.servers) > 0 {
			id := m.servers[m.cursor].ID
			return m, func() tea.Msg { m.client.Start(id); return m.fetchServers()() }
		}
	case "x":
		if len(m.servers) > 0 {
			id := m.servers[m.cursor].ID
			return m, func() tea.Msg { m.client.Stop(id); return m.fetchServers()() }
		}
	case "r":
		if len(m.servers) > 0 {
			id := m.servers[m.cursor].ID
			return m, func() tea.Msg { m.client.Restart(id); return m.fetchServers()() }
		}
	case "l":
		if len(m.servers) > 0 {
			id := m.servers[m.cursor].ID
			m.curView = viewLogs
			m.logs = nil
			return m, m.startLogStream(id)
		}
	case "c":
		if len(m.servers) > 0 {
			id := m.servers[m.cursor].ID
			m.curView = viewConsole
			m.input = ""
			m.logs = nil
			return m, m.startLogStream(id)
		}
	}
	return m, nil
}

func (m *Model) updateLogs(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "q" || msg.String() == "esc" {
		m.curView = viewList
		m.logs = nil
	}
	return m, nil
}

func (m *Model) updateConsole(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.curView = viewList
		m.input = ""
	case "enter":
		if m.input != "" && len(m.servers) > 0 {
			id := m.servers[m.cursor].ID
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

// startLogStream opens the log subscription and pushes each line as a
// logLineMsg into the bubbletea program via p.Send() — the correct pattern
// for feeding external goroutine events into a tea.Program.
func (m *Model) startLogStream(id string) tea.Cmd {
	return func() tea.Msg {
		ch, err := m.client.StreamLogs(id)
		if err != nil {
			return nil
		}
		go func() {
			for line := range ch {
				if m.program != nil {
					m.program.Send(logLineMsg(line))
				}
			}
		}()
		return nil
	}
}

func (m *Model) View() string {
	switch m.curView {
	case viewLogs:
		return m.viewLogs()
	case viewConsole:
		return m.viewConsole()
	default:
		return m.viewList()
	}
}

func (m *Model) viewList() string {
	var sb strings.Builder
	sb.WriteString(styleTitle.Render(" ◈ OPD — Server Manager ") + "\n")

	if len(m.servers) == 0 {
		sb.WriteString(styleNormal.Render("No running servers. Use 'opd start <id>' or press 's' to start.") + "\n")
	} else {
		for i, s := range m.servers {
			ram := fmt.Sprintf("%dMB / %dMB", s.RAMUsed/1024/1024, s.RAMMax/1024/1024)
			cpu := fmt.Sprintf("%.1f%%", s.CPU)
			line := fmt.Sprintf("%-16s %-10s %-18s cpu:%-7s ram:%s",
				s.ID, colorStatus(s.Status), s.Name, cpu, ram)
			if i == m.cursor {
				sb.WriteString(styleSelected.Render("▶ "+line) + "\n")
			} else {
				sb.WriteString(styleNormal.Render("  "+line) + "\n")
			}
		}
	}

	sb.WriteString(styleHelp.Render("↑/↓ navigate  s start  x stop  r restart  l logs  c console  q quit"))
	return styleBorder.Render(sb.String())
}

func (m *Model) viewLogs() string {
	var sb strings.Builder
	id := ""
	if len(m.servers) > 0 {
		id = m.servers[m.cursor].ID
	}
	sb.WriteString(styleTitle.Render(fmt.Sprintf(" ◈ Logs — %s ", id)) + "\n")

	visible := m.logs
	maxLines := m.height - 8
	if maxLines < 5 {
		maxLines = 5
	}
	if len(visible) > maxLines {
		visible = visible[len(visible)-maxLines:]
	}
	for _, line := range visible {
		sb.WriteString(styleLog.Render(line) + "\n")
	}
	sb.WriteString(styleHelp.Render("esc / q — back to list"))
	return styleBorder.Render(sb.String())
}

func (m *Model) viewConsole() string {
	var sb strings.Builder
	id := ""
	if len(m.servers) > 0 {
		id = m.servers[m.cursor].ID
	}
	sb.WriteString(styleTitle.Render(fmt.Sprintf(" ◈ Console — %s ", id)) + "\n")

	visible := m.logs
	maxLines := m.height - 10
	if maxLines < 3 {
		maxLines = 3
	}
	if len(visible) > maxLines {
		visible = visible[len(visible)-maxLines:]
	}
	for _, line := range visible {
		sb.WriteString(styleLog.Render(line) + "\n")
	}
	sb.WriteString("\n")
	sb.WriteString(styleCursor.Render("> ") + m.input + "█")
	sb.WriteString(styleHelp.Render("\nenter — send  esc — back"))
	return styleBorder.Render(sb.String())
}
