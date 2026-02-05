package agent

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
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
		reg, _ := LoadRegistry()
		if reg != nil {
			reg.Remove(id)
		}
	}()

	// Create output tracker
	tracker := newOutputTracker(id, logFile)

	// Start idle checker and status updater
	stopTracker := make(chan struct{})
	go tracker.run(stopTracker)
	defer close(stopTracker)

	// Copy stdin to pty
	go func() {
		io.Copy(ptmx, os.Stdin)
	}()

	// Copy pty output to stdout and track activity
	// Use a buffer to read chunks instead of waiting for newlines
	buf := make([]byte, 4096)
	for {
		n, err := ptmx.Read(buf)
		if err != nil {
			if err != io.EOF {
				// Ignore read errors on exit
			}
			break
		}
		if n > 0 {
			chunk := buf[:n]

			// Write to stdout
			os.Stdout.Write(chunk)

			// Write to log file
			logFile.Write(chunk)

			// Track activity
			tracker.recordOutput(chunk)
		}
	}

	// Wait for command to finish
	err = cmd.Wait()

	// Update final status
	exitCode := 0
	status := StatusDone
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
		status = StatusError
	}

	reg, _ := LoadRegistry()
	if reg != nil {
		reg.Update(id, func(e *Entry) {
			e.Status = status
			e.ExitCode = &exitCode
		})
	}

	return nil
}

// outputTracker tracks agent output for status updates
type outputTracker struct {
	id           string
	logFile      *os.File
	lastOutput   time.Time
	lastLine     string
	outputBuffer strings.Builder
	mu           sync.Mutex
}

func newOutputTracker(id string, logFile *os.File) *outputTracker {
	return &outputTracker{
		id:         id,
		logFile:    logFile,
		lastOutput: time.Now(),
	}
}

func (t *outputTracker) recordOutput(chunk []byte) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.lastOutput = time.Now()

	// Accumulate output to extract last meaningful line
	t.outputBuffer.Write(chunk)

	// Extract last non-empty line from buffer
	content := t.outputBuffer.String()
	lines := strings.Split(content, "\n")

	// Find last non-empty line
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		// Skip empty lines and common control sequences
		if line != "" && !strings.HasPrefix(line, "\033[") {
			t.lastLine = truncateLine(line, 100)
			break
		}
	}

	// Keep buffer from growing too large (keep last 1KB)
	if t.outputBuffer.Len() > 1024 {
		content := t.outputBuffer.String()
		t.outputBuffer.Reset()
		if len(content) > 512 {
			t.outputBuffer.WriteString(content[len(content)-512:])
		}
	}
}

func (t *outputTracker) run(stop chan struct{}) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			t.updateRegistry()
		}
	}
}

func (t *outputTracker) updateRegistry() {
	t.mu.Lock()
	lastOutput := t.lastOutput
	lastLine := t.lastLine
	t.mu.Unlock()

	// Determine status based on idle threshold
	status := StatusRunning
	if time.Since(lastOutput) > IdleThreshold {
		status = StatusIdle
	}

	// Load fresh registry and update
	registry, err := LoadRegistry()
	if err != nil {
		return
	}

	registry.Update(t.id, func(e *Entry) {
		e.Status = status
		e.LastOutputAt = lastOutput
		if lastLine != "" {
			e.LastOutputLine = lastLine
		}
	})
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
