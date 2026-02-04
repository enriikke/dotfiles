package agents

import (
	"fmt"
	"os"
	"os/exec"
)

// InstallResult represents the outcome of an agent installation
type InstallResult struct {
	Agent   Agent
	Success bool
	Skipped bool
	Error   error
}

// Installer handles AI agent installations
type Installer struct {
	DryRun bool
}

// NewInstaller creates a new agent installer
func NewInstaller(dryRun bool) *Installer {
	return &Installer{DryRun: dryRun}
}

// IsInstalled checks if an agent is already installed
func (i *Installer) IsInstalled(agent Agent) bool {
	if len(agent.CheckCmd) == 0 {
		return false
	}
	cmd := exec.Command(agent.CheckCmd[0], agent.CheckCmd[1:]...)
	return cmd.Run() == nil
}

// Install installs a single agent
func (i *Installer) Install(agent Agent) InstallResult {
	result := InstallResult{Agent: agent}

	if i.IsInstalled(agent) {
		result.Skipped = true
		result.Success = true
		return result
	}

	if i.DryRun {
		result.Success = true
		return result
	}

	var err error
	switch agent.InstallType {
	case InstallTypeNPM:
		err = i.installNPM(agent.Package)
	case InstallTypeCurl:
		err = i.installCurl(agent.Package)
	default:
		err = fmt.Errorf("unknown install type: %s", agent.InstallType)
	}

	if err != nil {
		result.Error = err
		return result
	}

	result.Success = true
	return result
}

// InstallAll installs multiple agents, continuing on errors
func (i *Installer) InstallAll(agents []Agent) []InstallResult {
	results := make([]InstallResult, 0, len(agents))
	for _, agent := range agents {
		results = append(results, i.Install(agent))
	}
	return results
}

// CheckNPMAvailable checks if npm is available
func (i *Installer) CheckNPMAvailable() bool {
	_, err := exec.LookPath("npm")
	return err == nil
}

// CheckCurlAvailable checks if curl is available
func (i *Installer) CheckCurlAvailable() bool {
	_, err := exec.LookPath("curl")
	return err == nil
}

func (i *Installer) installNPM(pkg string) error {
	if !i.CheckNPMAvailable() {
		return fmt.Errorf("npm is not installed. Install Node.js first (e.g., via fnm or brew install node)")
	}

	cmd := exec.Command("npm", "install", "-g", pkg)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("npm install failed: %w", err)
	}
	return nil
}

func (i *Installer) installCurl(url string) error {
	if !i.CheckCurlAvailable() {
		return fmt.Errorf("curl is not installed")
	}

	cmd := exec.Command("bash", "-c", fmt.Sprintf("curl -fsSL %s | bash", url))
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("curl install failed: %w", err)
	}
	return nil
}
