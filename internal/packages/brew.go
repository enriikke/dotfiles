package packages

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

type BrewInstaller struct {
	DryRun       bool
	sudoOnce     sync.Once
	sudoStopChan chan struct{}
}

func NewBrewInstaller(dryRun bool) *BrewInstaller {
	return &BrewInstaller{DryRun: dryRun}
}

// EnsureSudo authenticates sudo once and keeps the session alive in the
// background so that cask installs don't repeatedly prompt for a password.
func (b *BrewInstaller) EnsureSudo() error {
	var err error
	b.sudoOnce.Do(func() {
		if b.DryRun {
			return
		}
		sudo := exec.Command("sudo", "-v")
		sudo.Stdin = os.Stdin
		sudo.Stdout = os.Stdout
		sudo.Stderr = os.Stderr
		if e := sudo.Run(); e != nil {
			err = fmt.Errorf("sudo authentication required: %w", e)
			return
		}
		// Refresh sudo every 30s to keep it alive during long installs
		b.sudoStopChan = make(chan struct{})
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					exec.Command("sudo", "-v").Run()
				case <-b.sudoStopChan:
					return
				}
			}
		}()
	})
	return err
}

// StopSudo stops the background sudo keepalive.
func (b *BrewInstaller) StopSudo() {
	if b.sudoStopChan != nil {
		close(b.sudoStopChan)
	}
}

func (b *BrewInstaller) IsInstalled() bool {
	_, err := exec.LookPath("brew")
	return err == nil
}

func (b *BrewInstaller) Install() error {
	if b.IsInstalled() {
		return nil
	}

	if b.DryRun {
		return nil
	}

	if err := b.EnsureSudo(); err != nil {
		return err
	}

	cmd := exec.Command("/bin/bash", "-c", `curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh | /bin/bash`)
	cmd.Env = append(os.Environ(), "NONINTERACTIVE=1")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install Homebrew: %w", err)
	}

	return b.initShellEnv()
}

func (b *BrewInstaller) initShellEnv() error {
	var brewPath string

	if runtime.GOOS == "darwin" {
		if runtime.GOARCH == "arm64" {
			brewPath = "/opt/homebrew/bin/brew"
		} else {
			brewPath = "/usr/local/bin/brew"
		}
	} else {
		// Linux Homebrew
		brewPath = "/home/linuxbrew/.linuxbrew/bin/brew"
		if home := os.Getenv("HOME"); home != "" {
			altPath := filepath.Join(home, ".linuxbrew/bin/brew")
			if _, err := os.Stat(altPath); err == nil {
				brewPath = altPath
			}
		}
	}

	if _, err := os.Stat(brewPath); os.IsNotExist(err) {
		return fmt.Errorf("brew not found at expected path: %s", brewPath)
	}

	os.Setenv("PATH", filepath.Dir(brewPath)+":"+os.Getenv("PATH"))
	return nil
}

func (b *BrewInstaller) InstallPackages(brewfilePath string) error {
	if b.DryRun {
		return nil
	}

	if !b.IsInstalled() {
		if err := b.Install(); err != nil {
			return err
		}
	}

	if err := b.EnsureSudo(); err != nil {
		return err
	}

	cmd := exec.Command("brew", "bundle", "--file="+brewfilePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("brew bundle failed: %w", err)
	}

	return nil
}
