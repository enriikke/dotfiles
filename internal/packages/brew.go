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

	cmd := exec.Command("/bin/bash", "-c", `$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)`)
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
