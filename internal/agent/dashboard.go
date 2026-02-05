package agent

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Dashboard opens the interactive TUI
func Dashboard() error {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// Key bindings
type keyMap struct {
	Up      key.Binding
	Down    key.Binding
	Attach  key.Binding
	Logs    key.Binding
	Kill    key.Binding
	Refresh key.Binding
	Quit    key.Binding
	Help    key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Attach: key.NewBinding(
		key.WithKeys("enter", "a"),
		key.WithHelp("enter/a", "attach"),
	),
	Logs: key.NewBinding(
		key.WithKeys("l"),
		key.WithHelp("l", "logs"),
	),
	Kill: key.NewBinding(
		key.WithKeys("x"),
		key.WithHelp("x", "kill"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
}

// Styles
var (
	baseStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7C3AED"))

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED")).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Padding(0, 1)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#7C3AED"))

	runningCellStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981"))
	idleCellStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B"))
	doneCellStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	errorCellStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444"))

	previewStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF")).
			Italic(true).
			PaddingLeft(2)

	emptyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Italic(true).
			Align(lipgloss.Center).
			Padding(2, 0)
)

// Model
type model struct {
	table    table.Model
	agents   []Entry
	selected int
	width    int
	height   int
	showHelp bool
	message  string
}

type tickMsg time.Time
type refreshMsg struct{}

func initialModel() model {
	columns := []table.Column{
		{Title: "ID", Width: 10},
		{Title: "Project", Width: 20},
		{Title: "Agent", Width: 10},
		{Title: "Status", Width: 12},
		{Title: "Activity", Width: 10},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#7C3AED")).
		BorderBottom(true).
		Bold(true).
		Foreground(lipgloss.Color("#7C3AED"))
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#7C3AED")).
		Bold(true)
	t.SetStyles(s)

	m := model{
		table: t,
	}
	m.refreshAgents()

	return m
}

func (m *model) refreshAgents() {
	registry, err := LoadRegistry()
	if err != nil {
		m.message = fmt.Sprintf("Error: %v", err)
		return
	}

	m.agents = registry.All()
	rows := make([]table.Row, len(m.agents))

	for i, e := range m.agents {
		id := e.ID[:8]
		project := e.Project
		if e.Name != "" {
			project = e.Name
		}
		if len(project) > 18 {
			project = project[:15] + "..."
		}

		status := formatStatusIcon(e.Status)
		activity := formatActivity(e.LastOutputAt)

		rows[i] = table.Row{id, project, e.Agent, status, activity}
	}

	m.table.SetRows(rows)
}

func formatStatusIcon(status Status) string {
	switch status {
	case StatusRunning:
		return runningCellStyle.Render("⚡ running")
	case StatusIdle:
		return idleCellStyle.Render("⏳ idle")
	case StatusDone:
		return doneCellStyle.Render("✓ done")
	case StatusError:
		return errorCellStyle.Render("✗ error")
	default:
		return string(status)
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		m.message = "" // Clear message on keypress

		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, keys.Refresh):
			m.refreshAgents()
			m.message = "Refreshed"

		case key.Matches(msg, keys.Help):
			m.showHelp = !m.showHelp

		case key.Matches(msg, keys.Attach):
			if len(m.agents) > 0 {
				idx := m.table.Cursor()
				if idx < len(m.agents) {
					// For now, just show a message
					// TODO: Implement tmux attach
					m.message = fmt.Sprintf("Would attach to %s", m.agents[idx].ID[:8])
				}
			}

		case key.Matches(msg, keys.Logs):
			if len(m.agents) > 0 {
				idx := m.table.Cursor()
				if idx < len(m.agents) {
					// Show logs path for now
					m.message = fmt.Sprintf("Logs: %s", m.agents[idx].LogFile)
				}
			}

		case key.Matches(msg, keys.Kill):
			if len(m.agents) > 0 {
				idx := m.table.Cursor()
				if idx < len(m.agents) {
					err := Kill(m.agents[idx].ID)
					if err != nil {
						m.message = fmt.Sprintf("Error: %v", err)
					} else {
						m.message = fmt.Sprintf("Killed %s", m.agents[idx].ID[:8])
						m.refreshAgents()
					}
				}
			}
		}

	case tickMsg:
		m.refreshAgents()
		return m, tickCmd()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.table.SetWidth(msg.Width - 4)
		m.table.SetHeight(msg.Height - 12)
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m model) View() string {
	var b strings.Builder

	// Title
	title := titleStyle.Render("🤖 Agent Dashboard")
	b.WriteString(title)
	b.WriteString("\n\n")

	// Table or empty state
	if len(m.agents) == 0 {
		empty := emptyStyle.Render("No agents running\n\nStart one with: agent run copilot")
		b.WriteString(empty)
	} else {
		b.WriteString(m.table.View())

		// Preview of selected agent's last output
		idx := m.table.Cursor()
		if idx < len(m.agents) && m.agents[idx].LastOutputLine != "" {
			preview := previewStyle.Render("› " + m.agents[idx].LastOutputLine)
			b.WriteString("\n")
			b.WriteString(preview)
		}
	}

	// Status bar
	b.WriteString("\n\n")
	if m.message != "" {
		b.WriteString(statusBarStyle.Render(m.message))
	}

	// Help
	b.WriteString("\n")
	if m.showHelp {
		help := helpStyle.Render(
			"↑/k up • ↓/j down • enter/a attach • l logs • x kill • r refresh • q quit",
		)
		b.WriteString(help)
	} else {
		help := helpStyle.Render("? help • q quit")
		b.WriteString(help)
	}

	return baseStyle.Render(b.String())
}

// attachToTmux attaches to an agent's tmux session
func attachToTmux(entry *Entry) tea.Cmd {
	// This would need to know the tmux session/window
	// For now, we'd need to track this in the registry
	// or use a naming convention
	return nil
}
