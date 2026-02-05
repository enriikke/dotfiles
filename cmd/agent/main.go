package main

import (
	"fmt"
	"os"

	"github.com/enriikke/dotfiles/internal/agent"
	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "agent",
	Short: "Manage AI coding agents",
	Long: `Agent is a CLI tool for running and monitoring AI coding agents.

It wraps any AI agent (copilot, claude, codex, gemini) and provides:
  • Activity tracking across all your projects
  • Central dashboard showing all running agents  
  • Smart status detection (running vs idle)

Examples:
  agent run copilot                    # Run copilot in current directory
  agent run claude                     # Run any CLI agent
  agent run docker sandbox run copilot # Run in Docker sandbox
  agent dashboard                      # View all running agents
  agent ls                             # List agents (non-interactive)`,
}

var runCmd = &cobra.Command{
	Use:   "run <command> [args...]",
	Short: "Run an AI agent with tracking",
	Long: `Run an AI agent wrapped with activity tracking.

The agent runs normally - you interact with it directly.
In the background, agent tracks activity and logs output.

Examples:
  agent run copilot
  agent run copilot -- -p "fix the tests"
  agent run docker sandbox run copilot .
  agent run -n my-feature docker sandbox run copilot .`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		return agent.Run(args, name)
	},
}

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Show interactive dashboard of all agents",
	Long:  `Open an interactive TUI showing all running agents across projects.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return agent.Dashboard()
	},
}

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List all running agents",
	Long:  `List all running agents in a simple table format.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return agent.List()
	},
}

var logsCmd = &cobra.Command{
	Use:   "logs <id>",
	Short: "View logs for an agent",
	Long:  `Tail the log file for a specific agent.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		lines, _ := cmd.Flags().GetInt("lines")
		follow, _ := cmd.Flags().GetBool("follow")
		return agent.Logs(args[0], lines, follow)
	},
}

var killCmd = &cobra.Command{
	Use:   "kill <id>",
	Short: "Kill a running agent",
	Long:  `Terminate a running agent by its ID.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return agent.Kill(args[0])
	},
}

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove stale agent entries",
	Long:  `Remove entries for agents that are no longer running.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return agent.Clean()
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("agent version %s\n", version)
	},
}

func init() {
	runCmd.Flags().StringP("name", "n", "", "Name for this agent instance")
	logsCmd.Flags().IntP("lines", "l", 50, "Number of lines to show")
	logsCmd.Flags().BoolP("follow", "f", false, "Follow log output")

	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(dashboardCmd)
	rootCmd.AddCommand(lsCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(killCmd)
	rootCmd.AddCommand(cleanCmd)
	rootCmd.AddCommand(versionCmd)
}
