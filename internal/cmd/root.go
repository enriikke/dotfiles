package cmd

import (
	"github.com/enriikke/dotfiles/internal/ui"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "dotfiles",
	Short: "A CLI for managing dotfiles",
	Long: `dotfiles is a CLI tool for setting up and managing your development environment.

It handles package installation, dotfile symlinking, and shell configuration
across macOS, Linux, and GitHub Codespaces.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(sshCmd)
	rootCmd.AddCommand(aiCmd)
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		ui.PrintInfo("dotfiles version " + Version)
	},
}

var Version = "dev"
