package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/enriikke/dotfiles/internal/agents"
	"github.com/enriikke/dotfiles/internal/ui"
	"github.com/spf13/cobra"
)

var (
	aiAllFlag    bool
	aiAgentsFlag []string
)

var aiCmd = &cobra.Command{
	Use:   "ai",
	Short: "Install CLI AI coding agents",
	Long: `Install CLI AI coding agents for your development environment.

Supported agents:
  • codex   - OpenAI's CLI coding agent
  • claude  - Anthropic's CLI coding agent (Claude Code)
  • copilot - GitHub's CLI coding agent
  • gemini  - Google's CLI coding agent

Examples:
  dotfiles ai              # Interactive selection
  dotfiles ai --all        # Install all agents
  dotfiles ai --agent codex --agent claude  # Install specific agents`,
	RunE: runAI,
}

func init() {
	aiCmd.Flags().BoolVar(&aiAllFlag, "all", false, "Install all agents without prompting")
	aiCmd.Flags().StringArrayVar(&aiAgentsFlag, "agent", nil, "Install specific agent(s) by ID")
	aiCmd.Flags().BoolVar(&dryRunFlag, "dry-run", false, "Show what would be done without making changes")
}

func runAI(cmd *cobra.Command, args []string) error {
	ui.PrintTitle("🤖 AI Agents Setup")

	allAgents := agents.DefaultAgents()
	installer := agents.NewInstaller(dryRunFlag)

	// Check prerequisites
	ui.PrintHeader("Checking Prerequisites")
	npmAvailable := installer.CheckNPMAvailable()
	curlAvailable := installer.CheckCurlAvailable()

	if npmAvailable {
		ui.PrintSuccess("npm is available")
	} else {
		ui.PrintWarning("npm is not available - some agents will fail to install")
	}

	if curlAvailable {
		ui.PrintSuccess("curl is available")
	} else {
		ui.PrintWarning("curl is not available - some agents will fail to install")
	}

	var selectedAgents []agents.Agent

	if aiAllFlag {
		// Install all agents
		selectedAgents = allAgents
	} else if len(aiAgentsFlag) > 0 {
		// Install specific agents
		for _, id := range aiAgentsFlag {
			agent := agents.FindAgent(id)
			if agent == nil {
				ui.PrintError(fmt.Sprintf("Unknown agent: %s", id))
				ui.PrintInfo(fmt.Sprintf("Available agents: %s", strings.Join(agents.AgentIDs(), ", ")))
				return fmt.Errorf("unknown agent: %s", id)
			}
			selectedAgents = append(selectedAgents, *agent)
		}
	} else {
		// Interactive selection
		selected, err := selectAgentsInteractive(allAgents)
		if err != nil {
			return err
		}
		if len(selected) == 0 {
			ui.PrintInfo("No agents selected")
			return nil
		}
		selectedAgents = selected
	}

	// Install selected agents
	ui.PrintHeader("Installing Agents")

	if dryRunFlag {
		ui.PrintWarning("Dry run mode - no changes will be made")
	}

	results := installer.InstallAll(selectedAgents)

	// Print summary
	printAISummary(results, dryRunFlag)

	return nil
}

func selectAgentsInteractive(allAgents []agents.Agent) ([]agents.Agent, error) {
	ui.PrintHeader("Select Agents to Install")

	options := make([]huh.Option[string], len(allAgents))
	for i, agent := range allAgents {
		label := fmt.Sprintf("%s - %s", agent.Name, agent.Description)
		options[i] = huh.NewOption(label, agent.ID).Selected(true)
	}

	var selectedIDs []string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("AI Coding Agents").
				Description("Use space to toggle, enter to confirm").
				Options(options...).
				Value(&selectedIDs),
		),
	)

	if err := form.Run(); err != nil {
		return nil, err
	}

	selected := make([]agents.Agent, 0, len(selectedIDs))
	for _, id := range selectedIDs {
		if agent := agents.FindAgent(id); agent != nil {
			selected = append(selected, *agent)
		}
	}

	return selected, nil
}

func printAISummary(results []agents.InstallResult, dryRun bool) {
	ui.PrintHeader("Summary")

	var installed, skipped, failed int

	for _, result := range results {
		switch {
		case result.Error != nil:
			ui.PrintError(fmt.Sprintf("%s: %v", result.Agent.Name, result.Error))
			failed++
		case result.Skipped:
			ui.PrintStep(fmt.Sprintf("%s %s", result.Agent.Name, ui.SubtleStyle.Render("(already installed)")))
			skipped++
		case result.Success:
			if dryRun {
				ui.PrintSuccess(fmt.Sprintf("%s %s", result.Agent.Name, ui.SubtleStyle.Render("(would install)")))
			} else {
				ui.PrintSuccess(fmt.Sprintf("%s installed", result.Agent.Name))
			}
			installed++
		}
	}

	fmt.Println()
	if dryRun {
		ui.PrintInfo(fmt.Sprintf("Would install: %d, Already installed: %d, Failed: %d", installed, skipped, failed))
	} else {
		ui.PrintInfo(fmt.Sprintf("Installed: %d, Already installed: %d, Failed: %d", installed, skipped, failed))
	}

	if failed > 0 {
		fmt.Println()
		ui.PrintWarning("Some agents failed to install. Check the errors above.")
	}
}
