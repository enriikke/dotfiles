package agent

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/creack/pty"
	"golang.org/x/term"
)

// Run starts an agent with tracking
func Run(args []string, name string) error {
	// Detect agent type from command
	agentType := detectAgentType(args)
	isSandbox := detectSandbox(args)

	// Get working directory info
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}
	project := filepath.Base(cwd)

	// Generate unique ID
	id := GenerateID()

	// Set up log file
	if err := os.MkdirAll(LogsDir(), 0755); err != nil {
		return fmt.Errorf("failed to create logs dir: %w", err)
	}
	logPath := filepath.Join(LogsDir(), id+".log")
	logFile, err := os.Create(logPath)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}
	defer logFile.Close()

	// Create command
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = cwd
	cmd.Env = os.Environ()

	// Start with PTY
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("failed to start pty: %w", err)
	}
	defer ptmx.Close()

	// Handle terminal resize
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
				// Ignore resize errors
			}
		}
	}()
	ch <- syscall.SIGWINCH // Initial resize

	// Set stdin to raw mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("failed to set raw mode: %w", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Register agent
	registry, err := LoadRegistry()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	entry := Entry{
		ID:           id,
		Agent:        agentType,
		Name:         name,
		Project:      project,
		Cwd:          cwd,
		PID:          cmd.Process.Pid,
		Sandbox:      isSandbox,
		Status:       StatusRunning,
		StartedAt:    time.Now(),
		LastOutputAt: time.Now(),
		LogFile:      logPath,
	}

	if err := registry.Add(entry); err != nil {
		return fmt.Errorf("failed to register agent: %w", err)
	}

	// Clean up on exit
	defer func() {
		registry, _ := LoadRegistry()
		if registry != nil {
			registry.Remove(id)
		}
	}()

	// Create output tracker
	tracker := &outputTracker{
		id:         id,
		registry:   registry,
		lastOutput: time.Now(),
	}

	// Start idle checker
	stopIdle := make(chan struct{})
	go tracker.idleChecker(stopIdle)
	defer close(stopIdle)

	// Copy stdin to pty
	go func() {
		io.Copy(ptmx, os.Stdin)
	}()

	// Copy pty output to stdout and log file, tracking activity
	reader := bufio.NewReader(ptmx)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				// Ignore read errors on exit
			}
			break
		}

		// Write to stdout
		os.Stdout.WriteString(line)

		// Write to log file
		logFile.WriteString(line)

		// Track activity
		tracker.recordOutput(line)
	}

	// Wait for command to finish
	err = cmd.Wait()

	// Update final status
	registry, _ = LoadRegistry()
	if registry != nil {
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		registry.Update(id, func(e *Entry) {
			if exitCode == 0 {
				e.Status = StatusDone
			} else {
				e.Status = StatusError
			}
			e.ExitCode = &exitCode
		})
	}

	return nil
}

// outputTracker tracks agent output for status updates
type outputTracker struct {
	id         string
	registry   *Registry
	lastOutput time.Time
	lastLine   string
}

func (t *outputTracker) recordOutput(line string) {
	t.lastOutput = time.Now()
	t.lastLine = truncateLine(line, 100)

	// Update registry (debounced - only update every second at most)
	go func() {
		t.registry.Update(t.id, func(e *Entry) {
			e.Status = StatusRunning
			e.LastOutputAt = t.lastOutput
			e.LastOutputLine = t.lastLine
		})
	}()
}

func (t *outputTracker) idleChecker(stop chan struct{}) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			if time.Since(t.lastOutput) > IdleThreshold {
				t.registry.Update(t.id, func(e *Entry) {
					if e.Status == StatusRunning {
						e.Status = StatusIdle
					}
				})
			}
		}
	}
}

// detectAgentType tries to identify the agent from the command
func detectAgentType(args []string) string {
	cmd := strings.ToLower(args[0])

	// Check for known agents
	agents := []string{"copilot", "claude", "codex", "gemini"}
	for _, agent := range agents {
		if strings.Contains(cmd, agent) {
			return agent
		}
		// Check args for sandbox commands
		for _, arg := range args[1:] {
			if strings.Contains(strings.ToLower(arg), agent) {
				return agent
			}
		}
	}

	// Default to command name
	return filepath.Base(args[0])
}

// detectSandbox checks if this is a docker sandbox command
func detectSandbox(args []string) bool {
	for i, arg := range args {
		if arg == "sandbox" && i > 0 && strings.Contains(args[i-1], "docker") {
			return true
		}
	}
	return false
}

// truncateLine truncates a line to maxLen characters
func truncateLine(line string, maxLen int) string {
	line = strings.TrimSpace(line)
	if len(line) > maxLen {
		return line[:maxLen-3] + "..."
	}
	return line
}
