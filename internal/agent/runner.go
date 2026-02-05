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

	// Find last non-empty line (after stripping ANSI codes)
	for i := len(lines) - 1; i >= 0; i-- {
		// Strip ANSI escape codes
		cleaned := stripAnsi(lines[i])
		cleaned = strings.TrimSpace(cleaned)

		// Skip empty lines
		if cleaned == "" {
			continue
		}

		// Skip common noise patterns
		if isNoisePattern(cleaned) {
			continue
		}

		t.lastLine = truncateLine(cleaned, 100)
		break
	}

	// Keep buffer from growing too large (keep last 2KB)
	if t.outputBuffer.Len() > 2048 {
		content := t.outputBuffer.String()
		t.outputBuffer.Reset()
		if len(content) > 1024 {
			t.outputBuffer.WriteString(content[len(content)-1024:])
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

// stripAnsi removes ANSI escape codes from a string
func stripAnsi(s string) string {
	// Match ANSI escape sequences: ESC[ ... m (SGR), ESC[ ... other codes
	// Also match OSC sequences: ESC] ... BEL/ST
	var result strings.Builder
	i := 0
	for i < len(s) {
		if i < len(s)-1 && (s[i] == '\033' || s[i] == '\x9b') {
			// Start of escape sequence
			if s[i] == '\033' && i+1 < len(s) {
				if s[i+1] == '[' {
					// CSI sequence: ESC [ ... letter
					j := i + 2
					for j < len(s) && ((s[j] >= '0' && s[j] <= '9') || s[j] == ';' || s[j] == '?' || s[j] == '!' || s[j] == '"' || s[j] == '\'' || s[j] == ' ') {
						j++
					}
					if j < len(s) {
						j++ // Skip final byte
					}
					i = j
					continue
				} else if s[i+1] == ']' {
					// OSC sequence: ESC ] ... BEL or ST
					j := i + 2
					for j < len(s) && s[j] != '\007' && s[j] != '\033' {
						j++
					}
					if j < len(s) {
						if s[j] == '\033' && j+1 < len(s) && s[j+1] == '\\' {
							j += 2
						} else {
							j++
						}
					}
					i = j
					continue
				} else if s[i+1] == '(' || s[i+1] == ')' {
					// Character set selection
					i += 3
					continue
				} else {
					// Other escape (skip 2 chars)
					i += 2
					continue
				}
			}
			i++
			continue
		}
		// Skip other control characters except newline/tab
		if s[i] < 32 && s[i] != '\n' && s[i] != '\t' && s[i] != '\r' {
			i++
			continue
		}
		result.WriteByte(s[i])
		i++
	}
	return result.String()
}

// isNoisePattern checks if a line is common TUI noise we should skip
func isNoisePattern(s string) bool {
	// Skip lines that are just box drawing characters, dashes, etc.
	noiseChars := "─│┌┐└┘├┤┬┴┼━┃┏┓┗┛┣┫┳┻╋═║╔╗╚╝╠╣╦╩╬-=_|+*░▒▓█▀▄■□▪▫●○◐◑◒◓◔◕◖◗"
	cleaned := strings.TrimSpace(s)

	if cleaned == "" {
		return true
	}

	// Check if line is mostly noise characters
	noiseCount := 0
	for _, r := range cleaned {
		if strings.ContainsRune(noiseChars, r) || r == ' ' {
			noiseCount++
		}
	}

	// If more than 80% noise, skip it
	if float64(noiseCount)/float64(len([]rune(cleaned))) > 0.8 {
		return true
	}

	return false
}
