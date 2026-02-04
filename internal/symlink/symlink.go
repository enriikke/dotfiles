package symlink

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Manager struct {
	RepoPath  string
	HomeDir   string
	BackupDir string
	DryRun    bool
}

func NewManager(repoPath, homeDir string, dryRun bool) *Manager {
	backupDir := filepath.Join(homeDir, fmt.Sprintf(".dotfiles_backup_%s", time.Now().Format("20060102_150405")))
	return &Manager{
		RepoPath:  repoPath,
		HomeDir:   homeDir,
		BackupDir: backupDir,
		DryRun:    dryRun,
	}
}

type Result struct {
	Path   string
	Action string
	Error  error
}

func (m *Manager) Link(relativePath string) Result {
	sourcePath := filepath.Join(m.RepoPath, "home", relativePath)
	targetPath := filepath.Join(m.HomeDir, relativePath)

	result := Result{Path: relativePath}

	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		result.Error = fmt.Errorf("source does not exist: %s", sourcePath)
		return result
	}

	targetInfo, err := os.Lstat(targetPath)
	if err == nil {
		if targetInfo.Mode()&os.ModeSymlink != 0 {
			linkDest, err := os.Readlink(targetPath)
			if err == nil && linkDest == sourcePath {
				result.Action = "skipped"
				return result
			}
		}

		if err := m.backup(targetPath, relativePath); err != nil {
			result.Error = fmt.Errorf("failed to backup: %w", err)
			return result
		}
		result.Action = "backed_up"
	} else if !os.IsNotExist(err) {
		result.Error = fmt.Errorf("failed to check target: %w", err)
		return result
	}

	if m.DryRun {
		if result.Action == "" {
			result.Action = "would_create"
		}
		return result
	}

	parentDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		result.Error = fmt.Errorf("failed to create parent dir: %w", err)
		return result
	}

	if err := os.Symlink(sourcePath, targetPath); err != nil {
		result.Error = fmt.Errorf("failed to create symlink: %w", err)
		return result
	}

	if result.Action == "" {
		result.Action = "created"
	}
	return result
}

func (m *Manager) backup(targetPath, relativePath string) error {
	if m.DryRun {
		return nil
	}

	backupPath := filepath.Join(m.BackupDir, relativePath)

	if err := os.MkdirAll(filepath.Dir(backupPath), 0755); err != nil {
		return err
	}

	return os.Rename(targetPath, backupPath)
}

func (m *Manager) GetBackupDir() string {
	if _, err := os.Stat(m.BackupDir); os.IsNotExist(err) {
		return ""
	}
	return m.BackupDir
}

func (m *Manager) LinkAll(paths []string) []Result {
	results := make([]Result, 0, len(paths))
	for _, path := range paths {
		results = append(results, m.Link(path))
	}
	return results
}
