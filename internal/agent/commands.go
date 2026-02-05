package agent

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	primaryColor = lipgloss.Color("#7C3AED")
	successColor = lipgloss.Color("#10B981")
	warningColor = lipgloss.Color("#F59E0B")
	errorColor   = lipgloss.Color("#EF4444")
	subtleColor  = lipgloss.Color("#6B7280")

	// Styles
	headerStyle  = lipgloss.NewStyle().Bold(true).Foreground(primaryColor)
	idStyle      = lipgloss.NewStyle().Foreground(subtleColor)
	projectStyle = lipgloss.NewStyle().Bold(true)
	runningStyle = lipgloss.NewStyle().Foreground(successColor)
	idleStyle    = lipgloss.NewStyle().Foreground(warningColor)
	doneStyle    = lipgloss.NewStyle().Foreground(subtleColor)
	errorStyle   = lipgloss.NewStyle().Foreground(errorColor)
	subtleStyle  = lipgloss.NewStyle().Foreground(subtleColor)
)

// List prints all agents in a simple table format
func List() error {
	registry, err := LoadRegistry()
	if err != nil {
		return err
	}

	agents := registry.All()
	if len(agents) == 0 {
		fmt.Println(subtleStyle.Render("No agents running"))
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, headerStyle.Render("ID\tPROJECT\tAGENT\tSTATUS\tLAST ACTIVITY"))

	for _, e := range agents {
		id := idStyle.Render(e.ID[:8])
		project := projectStyle.Render(e.Project)
		if e.Name != "" {
			project = projectStyle.Render(e.Name)
		}
		agent := e.Agent
		status := formatStatus(e.Status)
		activity := formatActivity(e.LastOutputAt)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", id, project, agent, status, activity)
	}

	w.Flush()
	return nil
}

// Logs shows the log file for an agent
func Logs(id string, lines int, follow bool) error {
	registry, err := LoadRegistry()
	if err != nil {
		return err
	}

	entry := registry.Find(id)
	if entry == nil {
		return fmt.Errorf("agent not found: %s", id)
	}

	if follow {
		// Tail -f style
		cmd := exec.Command("tail", "-f", "-n", fmt.Sprintf("%d", lines), entry.LogFile)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	// Read last N lines
	file, err := os.Open(entry.LogFile)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	// Simple approach: read all lines and show last N
	var allLines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}

	start := 0
	if len(allLines) > lines {
		start = len(allLines) - lines
	}

	for _, line := range allLines[start:] {
		fmt.Println(line)
	}

	return nil
}

// Kill terminates a running agent
func Kill(id string) error {
	registry, err := LoadRegistry()
	if err != nil {
		return err
	}

	entry := registry.Find(id)
	if entry == nil {
		return fmt.Errorf("agent not found: %s", id)
	}

	if entry.Status != StatusRunning && entry.Status != StatusIdle {
		return fmt.Errorf("agent is not running (status: %s)", entry.Status)
	}

	// Send SIGTERM to process
	process, err := os.FindProcess(entry.PID)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to kill process: %w", err)
	}

	fmt.Printf("Sent SIGTERM to agent %s (PID %d)\n", entry.ID[:8], entry.PID)
	return nil
}

// Clean removes stale agent entries
func Clean() error {
	registry, err := LoadRegistry()
	if err != nil {
		return err
	}

	var removed int
	var toRemove []string

	for _, e := range registry.All() {
		// Check if process is still running
		process, err := os.FindProcess(e.PID)
		if err != nil {
			toRemove = append(toRemove, e.ID)
			continue
		}

		// Check if process is alive (signal 0 doesn't kill, just checks)
		if err := process.Signal(syscall.Signal(0)); err != nil {
			toRemove = append(toRemove, e.ID)
		}
	}

	for _, id := range toRemove {
		registry.Remove(id)
		removed++
	}

	if removed > 0 {
		fmt.Printf("Removed %d stale agent(s)\n", removed)
	} else {
		fmt.Println("No stale agents found")
	}

	return nil
}

func formatStatus(status Status) string {
	switch status {
	case StatusRunning:
		return runningStyle.Render("⚡ running")
	case StatusIdle:
		return idleStyle.Render("⏳ idle")
	case StatusDone:
		return doneStyle.Render("✓ done")
	case StatusError:
		return errorStyle.Render("✗ error")
	default:
		return string(status)
	}
}

func formatActivity(t time.Time) string {
	if t.IsZero() {
		return subtleStyle.Render("-")
	}

	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	default:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
}
