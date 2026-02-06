package packages

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

type BrewInstaller struct {
	DryRun bool
}

func NewBrewInstaller(dryRun bool) *BrewInstaller {
	return &BrewInstaller{DryRun: dryRun}
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

	// Pre-authenticate sudo so Homebrew's installer finds an active session.
	// This lets us run in NONINTERACTIVE mode (skipping "Press RETURN") while
	// still satisfying the sudo requirement. In environments with passwordless
	// sudo (e.g. Codespaces), this succeeds silently.
	sudo := exec.Command("sudo", "-v")
	sudo.Stdin = os.Stdin
	sudo.Stdout = os.Stdout
	sudo.Stderr = os.Stderr
	if err := sudo.Run(); err != nil {
		return fmt.Errorf("sudo authentication required for Homebrew: %w", err)
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

	cmd := exec.Command("brew", "bundle", "--file="+brewfilePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("brew bundle failed: %w", err)
	}

	return nil
}
