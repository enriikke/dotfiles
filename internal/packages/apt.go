package packages

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type AptInstaller struct {
	DryRun bool
}

func NewAptInstaller(dryRun bool) *AptInstaller {
	return &AptInstaller{DryRun: dryRun}
}

func (a *AptInstaller) IsAvailable() bool {
	_, err := exec.LookPath("apt-get")
	return err == nil
}

func (a *AptInstaller) InstallPackages(packagesFile string) error {
	packages, err := a.readPackageList(packagesFile)
	if err != nil {
		return fmt.Errorf("failed to read packages file: %w", err)
	}

	if len(packages) == 0 {
		return nil
	}

	if a.DryRun {
		return nil
	}

	updateCmd := exec.Command("sudo", "apt-get", "update", "-qq")
	updateCmd.Stdout = os.Stdout
	updateCmd.Stderr = os.Stderr
	if err := updateCmd.Run(); err != nil {
		return fmt.Errorf("apt-get update failed: %w", err)
	}

	args := append([]string{"apt-get", "install", "-y", "-qq"}, packages...)
	installCmd := exec.Command("sudo", args...)
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr

	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("apt-get install failed: %w", err)
	}

	return nil
}

func (a *AptInstaller) readPackageList(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var packages []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		packages = append(packages, line)
	}

	return packages, scanner.Err()
}
