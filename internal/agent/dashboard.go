package agent

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Dashboard opens the interactive TUI
func Dashboard() error {
	p := tea.NewProgram(newDashboardModel(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// Colors
var (
	purple     = lipgloss.Color("#7C3AED")
	green      = lipgloss.Color("#10B981")
	yellow     = lipgloss.Color("#F59E0B")
	red        = lipgloss.Color("#EF4444")
	gray       = lipgloss.Color("#6B7280")
	darkGray   = lipgloss.Color("#374151")
	white      = lipgloss.Color("#FFFFFF")
	lightGray  = lipgloss.Color("#9CA3AF")
	cyan       = lipgloss.Color("#06B6D4")
	background = lipgloss.Color("#1F2937")
)

// Styles
var (
	// Header styles
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(white).
			Background(purple).
			Padding(0, 1)

	statusReadyStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(green)

	statusBadgeStyle = lipgloss.NewStyle().
				Foreground(cyan).
				Bold(true)

	// Panel styles
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(darkGray)

	panelTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(purple).
			Padding(0, 1)

	panelFocusedStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(purple)

	// List item styles
	itemNormalStyle = lipgloss.NewStyle().
			Padding(0, 1)

	itemSelectedStyle = lipgloss.NewStyle().
				Background(purple).
				Foreground(white).
				Bold(true).
				Padding(0, 1)

	// Status indicators
	runningIndicator = lipgloss.NewStyle().Foreground(green).Bold(true)
	idleIndicator    = lipgloss.NewStyle().Foreground(yellow).Bold(true)
	doneIndicator    = lipgloss.NewStyle().Foreground(gray)
	errorIndicator   = lipgloss.NewStyle().Foreground(red).Bold(true)

	// Detail styles
	labelStyle = lipgloss.NewStyle().
			Foreground(gray).
			Width(12)

	valueStyle = lipgloss.NewStyle().
			Foreground(white)

	sectionStyle = lipgloss.NewStyle().
			Foreground(purple).
			Bold(true).
			MarginTop(1).
			MarginBottom(0)

	logStyle = lipgloss.NewStyle().
			Foreground(lightGray)

	// Help bar
	helpBarStyle = lipgloss.NewStyle().
			Foreground(gray).
			Background(darkGray).
			Padding(0, 1)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(cyan).
			Bold(true)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(gray)

	// Message styles
	messageStyle = lipgloss.NewStyle().
			Foreground(yellow).
			Italic(true)

	errorMsgStyle = lipgloss.NewStyle().
			Foreground(red)
)

// Key bindings
type dashboardKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Enter    key.Binding
	Tab      key.Binding
	Logs     key.Binding
	Kill     key.Binding
	Clean    key.Binding
	Refresh  key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Quit     key.Binding
}

var dashboardKeys = dashboardKeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "switch panel"),
	),
	Logs: key.NewBinding(
		key.WithKeys("l"),
		key.WithHelp("l", "logs"),
	),
	Kill: key.NewBinding(
		key.WithKeys("x"),
		key.WithHelp("x", "kill"),
	),
	Clean: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "clean"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	PageUp: key.NewBinding(
		key.WithKeys("pgup", "ctrl+u"),
		key.WithHelp("pgup", "scroll up"),
	),
	PageDown: key.NewBinding(
		key.WithKeys("pgdown", "ctrl+d"),
		key.WithHelp("pgdn", "scroll down"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

// Panel enum
type panel int

const (
	panelList panel = iota
	panelDetail
)

// Model
type dashboardModel struct {
	agents       []Entry
	selected     int
	focusedPanel panel
	logViewport  viewport.Model
	showLogs     bool
	width        int
	height       int
	message      string
	messageIsErr bool
	lastUpdate   time.Time
}

type tickMsg time.Time

func newDashboardModel() dashboardModel {
	vp := viewport.New(40, 10)
	vp.Style = logStyle

	m := dashboardModel{
		logViewport: vp,
		lastUpdate:  time.Now(),
	}
	m.refreshAgents()
	return m
}

func (m *dashboardModel) refreshAgents() {
	registry, err := LoadRegistry()
	if err != nil {
		m.message = fmt.Sprintf("Error: %v", err)
		m.messageIsErr = true
		return
	}

	m.agents = registry.All()
	m.lastUpdate = time.Now()

	// Validate selected index
	if m.selected >= len(m.agents) {
		m.selected = max(0, len(m.agents)-1)
	}

	// Update log viewport if showing logs
	if m.showLogs && len(m.agents) > 0 && m.selected < len(m.agents) {
		m.loadLogs()
	}
}

func (m *dashboardModel) loadLogs() {
	if m.selected >= len(m.agents) {
		return
	}

	entry := m.agents[m.selected]
	content := m.readLogTail(entry.LogFile, 50)
	m.logViewport.SetContent(content)
	m.logViewport.GotoBottom()
}

func (m *dashboardModel) readLogTail(path string, lines int) string {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Sprintf("Could not open log: %v", err)
	}
	defer file.Close()

	var allLines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}

	if len(allLines) == 0 {
		return "(no output yet)"
	}

	start := max(0, len(allLines)-lines)
	return strings.Join(allLines[start:], "\n")
}

func (m dashboardModel) Init() tea.Cmd {
	return tea.Batch(tickCmd())
}

func tickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m dashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Clear message on any keypress
		m.message = ""
		m.messageIsErr = false

		switch {
		case key.Matches(msg, dashboardKeys.Quit):
			return m, tea.Quit

		case key.Matches(msg, dashboardKeys.Up):
			if m.focusedPanel == panelList {
				if m.selected > 0 {
					m.selected--
					if m.showLogs {
						m.loadLogs()
					}
				}
			} else if m.showLogs {
				m.logViewport.LineUp(3)
			}

		case key.Matches(msg, dashboardKeys.Down):
			if m.focusedPanel == panelList {
				if m.selected < len(m.agents)-1 {
					m.selected++
					if m.showLogs {
						m.loadLogs()
					}
				}
			} else if m.showLogs {
				m.logViewport.LineDown(3)
			}

		case key.Matches(msg, dashboardKeys.PageUp):
			if m.showLogs && m.focusedPanel == panelDetail {
				m.logViewport.HalfViewUp()
			}

		case key.Matches(msg, dashboardKeys.PageDown):
			if m.showLogs && m.focusedPanel == panelDetail {
				m.logViewport.HalfViewDown()
			}

		case key.Matches(msg, dashboardKeys.Tab):
			if m.focusedPanel == panelList {
				m.focusedPanel = panelDetail
			} else {
				m.focusedPanel = panelList
			}

		case key.Matches(msg, dashboardKeys.Logs):
			m.showLogs = !m.showLogs
			if m.showLogs {
				m.loadLogs()
			}

		case key.Matches(msg, dashboardKeys.Kill):
			if len(m.agents) > 0 && m.selected < len(m.agents) {
				entry := m.agents[m.selected]
				if err := Kill(entry.ID); err != nil {
					m.message = fmt.Sprintf("Failed to kill: %v", err)
					m.messageIsErr = true
				} else {
					m.message = fmt.Sprintf("Killed %s", entry.ID[:8])
					m.refreshAgents()
				}
			}

		case key.Matches(msg, dashboardKeys.Clean):
			removed, err := Clean()
			if err != nil {
				m.message = fmt.Sprintf("Clean failed: %v", err)
				m.messageIsErr = true
			} else {
				m.message = fmt.Sprintf("Cleaned %d stale entries", removed)
				m.refreshAgents()
			}

		case key.Matches(msg, dashboardKeys.Refresh):
			m.refreshAgents()
			m.message = "Refreshed"
		}

	case tickMsg:
		m.refreshAgents()
		return m, tickCmd()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Update viewport size for logs
		detailWidth := m.width - m.listWidth() - 6
		logHeight := m.height - 16 // Account for header, details, help
		if logHeight < 5 {
			logHeight = 5
		}
		m.logViewport.Width = detailWidth - 4
		m.logViewport.Height = logHeight
	}

	if m.showLogs && m.focusedPanel == panelDetail {
		m.logViewport, cmd = m.logViewport.Update(msg)
	}

	return m, cmd
}

func (m dashboardModel) listWidth() int {
	// List panel is about 35% of width, min 30
	w := m.width * 35 / 100
	if w < 30 {
		w = 30
	}
	if w > 50 {
		w = 50
	}
	return w
}

func (m dashboardModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var b strings.Builder

	// Header bar
	b.WriteString(m.renderHeader())
	b.WriteString("\n")

	// Main content: list panel + detail panel
	listWidth := m.listWidth()
	detailWidth := m.width - listWidth - 3

	listPanel := m.renderListPanel(listWidth)
	detailPanel := m.renderDetailPanel(detailWidth)

	content := lipgloss.JoinHorizontal(lipgloss.Top, listPanel, detailPanel)
	b.WriteString(content)
	b.WriteString("\n")

	// Help bar
	b.WriteString(m.renderHelpBar())

	return b.String()
}

func (m dashboardModel) renderHeader() string {
	// Left side: status indicator + title
	var statusDot string
	runningCount := 0
	for _, a := range m.agents {
		if a.Status == StatusRunning {
			runningCount++
		}
	}

	if runningCount > 0 {
		statusDot = statusReadyStyle.Render("● Active")
	} else if len(m.agents) > 0 {
		statusDot = idleIndicator.Render("○ Idle")
	} else {
		statusDot = lipgloss.NewStyle().Foreground(gray).Render("○ Empty")
	}

	title := headerStyle.Render(" 🤖 Agent Dashboard ")

	// Right side: counts
	counts := statusBadgeStyle.Render(fmt.Sprintf("%d agents", len(m.agents)))

	// Message if any
	var msg string
	if m.message != "" {
		if m.messageIsErr {
			msg = errorMsgStyle.Render("  " + m.message)
		} else {
			msg = messageStyle.Render("  " + m.message)
		}
	}

	leftPart := lipgloss.JoinHorizontal(lipgloss.Center, statusDot, "  ", title, msg)

	// Calculate padding
	padding := m.width - lipgloss.Width(leftPart) - lipgloss.Width(counts) - 2
	if padding < 1 {
		padding = 1
	}

	return leftPart + strings.Repeat(" ", padding) + counts
}

func (m dashboardModel) renderListPanel(width int) string {
	style := panelStyle
	if m.focusedPanel == panelList {
		style = panelFocusedStyle
	}

	title := panelTitleStyle.Render("─ Agents ")
	innerWidth := width - 4

	var content strings.Builder

	if len(m.agents) == 0 {
		empty := lipgloss.NewStyle().
			Foreground(gray).
			Italic(true).
			Width(innerWidth).
			Align(lipgloss.Center).
			Render("\nNo agents running\n\nStart one with:\nagent run copilot")
		content.WriteString(empty)
	} else {
		for i, agent := range m.agents {
			// Status indicator
			var indicator string
			switch agent.Status {
			case StatusRunning:
				indicator = runningIndicator.Render("▶")
			case StatusIdle:
				indicator = idleIndicator.Render("◆")
			case StatusDone:
				indicator = doneIndicator.Render("✓")
			case StatusError:
				indicator = errorIndicator.Render("✗")
			}

			// Agent name/project
			name := agent.Project
			if agent.Name != "" {
				name = agent.Name
			}

			// Truncate if needed
			maxNameLen := innerWidth - 12
			if len(name) > maxNameLen {
				name = name[:maxNameLen-3] + "..."
			}

			// Agent type badge
			agentBadge := lipgloss.NewStyle().Foreground(cyan).Render(agent.Agent)

			line := fmt.Sprintf("%s %s  %s", indicator, name, agentBadge)

			// Apply selection style
			if i == m.selected {
				line = itemSelectedStyle.Width(innerWidth).Render(line)
			} else {
				line = itemNormalStyle.Width(innerWidth).Render(line)
			}

			content.WriteString(line)
			if i < len(m.agents)-1 {
				content.WriteString("\n")
			}
		}
	}

	// Calculate height for list panel
	listHeight := m.height - 5 // Header + help bar
	if listHeight < 5 {
		listHeight = 5
	}

	innerContent := lipgloss.NewStyle().
		Width(innerWidth).
		Height(listHeight).
		Render(content.String())

	panel := style.Width(width).Render(title + "\n" + innerContent)
	return panel
}

func (m dashboardModel) renderDetailPanel(width int) string {
	style := panelStyle
	if m.focusedPanel == panelDetail {
		style = panelFocusedStyle
	}

	title := panelTitleStyle.Render("─ Details ")
	innerWidth := width - 4

	var content strings.Builder

	if len(m.agents) == 0 || m.selected >= len(m.agents) {
		empty := lipgloss.NewStyle().
			Foreground(gray).
			Italic(true).
			Width(innerWidth).
			Render("\nSelect an agent to view details")
		content.WriteString(empty)
	} else {
		agent := m.agents[m.selected]

		// Agent info section
		content.WriteString(m.renderDetailRow("ID", agent.ID[:12]))
		content.WriteString(m.renderDetailRow("Agent", agent.Agent))
		content.WriteString(m.renderDetailRow("Project", agent.Project))
		content.WriteString(m.renderDetailRow("Directory", truncatePath(agent.Cwd, innerWidth-14)))
		content.WriteString(m.renderDetailRow("Status", m.renderStatusValue(agent.Status)))
		content.WriteString(m.renderDetailRow("Started", formatTimeAgo(agent.StartedAt)))
		content.WriteString(m.renderDetailRow("Activity", formatTimeAgo(agent.LastOutputAt)))
		content.WriteString(m.renderDetailRow("PID", fmt.Sprintf("%d", agent.PID)))

		if agent.Sandbox {
			content.WriteString(m.renderDetailRow("Sandbox", "yes (docker)"))
		}

		// Last output preview
		content.WriteString("\n")
		content.WriteString(sectionStyle.Render("─ Last Output "))
		content.WriteString("\n")
		if agent.LastOutputLine != "" {
			preview := lipgloss.NewStyle().
				Foreground(lightGray).
				Width(innerWidth).
				Render("› " + agent.LastOutputLine)
			content.WriteString(preview)
		} else {
			content.WriteString(lipgloss.NewStyle().Foreground(gray).Italic(true).Render("(no output)"))
		}

		// Logs section (if enabled)
		if m.showLogs {
			content.WriteString("\n\n")
			content.WriteString(sectionStyle.Render("─ Logs (l to toggle) "))
			content.WriteString("\n")
			content.WriteString(m.logViewport.View())
		}
	}

	// Calculate height for detail panel
	detailHeight := m.height - 5
	if detailHeight < 5 {
		detailHeight = 5
	}

	innerContent := lipgloss.NewStyle().
		Width(innerWidth).
		Height(detailHeight).
		Render(content.String())

	panel := style.Width(width).Render(title + "\n" + innerContent)
	return panel
}

func (m dashboardModel) renderDetailRow(label, value string) string {
	l := labelStyle.Render(label + ":")
	v := valueStyle.Render(value)
	return l + " " + v + "\n"
}

func (m dashboardModel) renderStatusValue(status Status) string {
	switch status {
	case StatusRunning:
		return runningIndicator.Render("● running")
	case StatusIdle:
		return idleIndicator.Render("◆ idle")
	case StatusDone:
		return doneIndicator.Render("✓ done")
	case StatusError:
		return errorIndicator.Render("✗ error")
	default:
		return string(status)
	}
}

func (m dashboardModel) renderHelpBar() string {
	helps := []struct{ key, desc string }{
		{"↑/k", "up"},
		{"↓/j", "down"},
		{"tab", "panel"},
		{"l", "logs"},
		{"x", "kill"},
		{"c", "clean"},
		{"r", "refresh"},
		{"q", "quit"},
	}

	var parts []string
	for _, h := range helps {
		part := helpKeyStyle.Render(h.key) + helpDescStyle.Render(":"+h.desc)
		parts = append(parts, part)
	}

	helpText := strings.Join(parts, "  ")
	return helpBarStyle.Width(m.width).Render(helpText)
}

// Helper functions

func formatTimeAgo(t time.Time) string {
	if t.IsZero() {
		return "-"
	}

	d := time.Since(t)
	switch {
	case d < time.Second:
		return "just now"
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func truncatePath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}
	// Show .../<last two parts>
	parts := strings.Split(path, "/")
	if len(parts) <= 2 {
		return path[:maxLen-3] + "..."
	}
	short := ".../" + strings.Join(parts[len(parts)-2:], "/")
	if len(short) > maxLen {
		return short[:maxLen-3] + "..."
	}
	return short
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
