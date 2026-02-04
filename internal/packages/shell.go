package packages

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func SetDefaultShell(dryRun bool) error {
	currentShell := os.Getenv("SHELL")
	if strings.Contains(currentShell, "zsh") {
		return nil
	}

	zshPath, err := exec.LookPath("zsh")
	if err != nil {
		return fmt.Errorf("zsh not found: %w", err)
	}

	if dryRun {
		return nil
	}

	if !isShellAllowed(zshPath) {
		cmd := exec.Command("sudo", "sh", "-c", fmt.Sprintf("echo '%s' >> /etc/shells", zshPath))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to add zsh to /etc/shells: %w", err)
		}
	}

	cmd := exec.Command("chsh", "-s", zshPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		cmd = exec.Command("sudo", "chsh", "-s", zshPath, os.Getenv("USER"))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to change shell: %w", err)
		}
	}

	return nil
}

func isShellAllowed(shellPath string) bool {
	data, err := os.ReadFile("/etc/shells")
	if err != nil {
		return false
	}
	return strings.Contains(string(data), shellPath)
}

func IsZshDefault() bool {
	return strings.Contains(os.Getenv("SHELL"), "zsh")
}
