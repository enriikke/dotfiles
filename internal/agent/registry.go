// Package agent provides functionality for running and monitoring AI coding agents.
package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Status represents the current state of an agent
type Status string

const (
	StatusRunning Status = "running" // Agent is active (recent output)
	StatusIdle    Status = "idle"    // No output for threshold duration
	StatusDone    Status = "done"    // Agent exited successfully
	StatusError   Status = "error"   // Agent exited with error
)

// IdleThreshold is how long without output before marking as idle
const IdleThreshold = 60 * time.Second

// Entry represents a running agent in the registry
type Entry struct {
	ID             string    `json:"id"`
	Agent          string    `json:"agent"`           // e.g., "copilot", "claude"
	Name           string    `json:"name,omitempty"`  // User-provided name
	Project        string    `json:"project"`         // Directory name
	Cwd            string    `json:"cwd"`             // Full working directory
	PID            int       `json:"pid"`             // Process ID
	Sandbox        bool      `json:"sandbox"`         // Running in docker sandbox
	Status         Status    `json:"status"`          // Current status
	StartedAt      time.Time `json:"started_at"`      // When agent started
	LastOutputAt   time.Time `json:"last_output_at"`  // Last output timestamp
	LastOutputLine string    `json:"last_output_line"` // Last line of output (truncated)
	LogFile        string    `json:"log_file"`        // Path to log file
	ExitCode       *int      `json:"exit_code,omitempty"` // Exit code if done/error
}

// Registry manages the collection of running agents
type Registry struct {
	Agents []Entry `json:"agents"`
	mu     sync.RWMutex
	path   string
}

// RegistryDir returns the directory for agent data
func RegistryDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".agent")
}

// RegistryPath returns the path to the registry file
func RegistryPath() string {
	return filepath.Join(RegistryDir(), "status.json")
}

// LogsDir returns the directory for agent logs
func LogsDir() string {
	return filepath.Join(RegistryDir(), "logs")
}

// LoadRegistry loads the registry from disk
func LoadRegistry() (*Registry, error) {
	r := &Registry{
		path:   RegistryPath(),
		Agents: []Entry{},
	}

	data, err := os.ReadFile(r.path)
	if err != nil {
		if os.IsNotExist(err) {
			return r, nil
		}
		return nil, fmt.Errorf("failed to read registry: %w", err)
	}

	if err := json.Unmarshal(data, r); err != nil {
		return nil, fmt.Errorf("failed to parse registry: %w", err)
	}

	return r, nil
}

// Save writes the registry to disk
func (r *Registry) Save() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(r.path), 0755); err != nil {
		return fmt.Errorf("failed to create registry dir: %w", err)
	}

	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	if err := os.WriteFile(r.path, data, 0644); err != nil {
		return fmt.Errorf("failed to write registry: %w", err)
	}

	return nil
}

// Add adds a new agent entry
func (r *Registry) Add(entry Entry) error {
	r.mu.Lock()
	r.Agents = append(r.Agents, entry)
	r.mu.Unlock()
	return r.Save()
}

// Update updates an existing agent entry
func (r *Registry) Update(id string, fn func(*Entry)) error {
	r.mu.Lock()
	for i := range r.Agents {
		if r.Agents[i].ID == id {
			fn(&r.Agents[i])
			break
		}
	}
	r.mu.Unlock()
	return r.Save()
}

// Remove removes an agent entry
func (r *Registry) Remove(id string) error {
	r.mu.Lock()
	for i := range r.Agents {
		if r.Agents[i].ID == id {
			r.Agents = append(r.Agents[:i], r.Agents[i+1:]...)
			break
		}
	}
	r.mu.Unlock()
	return r.Save()
}

// Find finds an agent by ID (supports prefix matching)
func (r *Registry) Find(id string) *Entry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for i := range r.Agents {
		if r.Agents[i].ID == id {
			return &r.Agents[i]
		}
		// Prefix match
		if len(id) >= 4 && len(r.Agents[i].ID) >= len(id) && r.Agents[i].ID[:len(id)] == id {
			return &r.Agents[i]
		}
	}
	return nil
}

// Active returns all agents that are running or idle
func (r *Registry) Active() []Entry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var active []Entry
	for _, e := range r.Agents {
		if e.Status == StatusRunning || e.Status == StatusIdle {
			active = append(active, e)
		}
	}
	return active
}

// All returns all agents
func (r *Registry) All() []Entry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]Entry{}, r.Agents...)
}

// GenerateID creates a unique ID for a new agent
func GenerateID() string {
	// Simple ID: timestamp + random suffix
	return fmt.Sprintf("%x", time.Now().UnixNano())[:12]
}
