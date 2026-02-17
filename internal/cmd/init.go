package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/enriikke/dotfiles/internal/config"
	"github.com/enriikke/dotfiles/internal/macos"
	"github.com/enriikke/dotfiles/internal/packages"
	"github.com/enriikke/dotfiles/internal/platform"
	"github.com/enriikke/dotfiles/internal/symlink"
	"github.com/enriikke/dotfiles/internal/ui"
	"github.com/spf13/cobra"
)

var (
	repoFlag   string
	dryRunFlag bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize your development environment",
	Long: `Initialize sets up your development environment by:
  1. Installing packages via Homebrew
  2. Symlinking dotfiles to your home directory
  3. Configuring macOS system settings (macOS only)
  4. Setting zsh as your default shell`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().StringVar(&repoFlag, "repo", "", "Path to dotfiles repository")
	initCmd.Flags().BoolVar(&dryRunFlag, "dry-run", false, "Show what would be done without making changes")
}

func runInit(cmd *cobra.Command, args []string) error {
	ui.PrintTitle("🚀 dotfiles init")

	ui.PrintHeader("Detecting Platform")
	plat := platform.Detect()
	ui.PrintSuccess(fmt.Sprintf("Platform: %s", plat.String()))

	ui.PrintHeader("Finding Dotfiles Repository")
	repoPath, err := findRepo(nil)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Could not find dotfiles repository: %v", err))
		return err
	}
	ui.PrintSuccess(fmt.Sprintf("Found repo: %s", repoPath))

	cfg, err := config.Load(repoPath)
	if err != nil {
		ui.PrintWarning(fmt.Sprintf("Could not load config, using defaults: %v", err))
		cfg = config.DefaultConfig()
	}

	if dryRunFlag {
		ui.PrintWarning("Dry run mode - no changes will be made")
	}

	ui.PrintHeader("Installing Packages")
	if err := installPackages(plat, repoPath); err != nil {
		ui.PrintError(fmt.Sprintf("Package installation failed: %v", err))
	}

	ui.PrintHeader("Symlinking Dotfiles")
	if err := symlinkDotfiles(repoPath, cfg); err != nil {
		ui.PrintError(fmt.Sprintf("Symlink failed: %v", err))
		return err
	}

	if plat.IsMacOS() {
		ui.PrintHeader("macOS Settings")
		if err := configureMacOS(); err != nil {
			ui.PrintWarning(fmt.Sprintf("macOS settings failed: %v", err))
		}
	}

	ui.PrintHeader("Shell Configuration")
	if err := configureShell(); err != nil {
		ui.PrintWarning(fmt.Sprintf("Could not set default shell: %v", err))
	}

	printSummary(plat)
	return nil
}

func findRepo(_ *config.Config) (string, error) {
	if repoFlag != "" {
		expanded := expandPath(repoFlag)
		if isValidRepo(expanded) {
			return expanded, nil
		}
		return "", fmt.Errorf("specified repo path is not valid: %s", expanded)
	}

	cwd, err := os.Getwd()
	if err == nil && isValidRepo(cwd) {
		return cwd, nil
	}

	dotfiles := expandPath("~/.dotfiles")
	if isValidRepo(dotfiles) {
		return dotfiles, nil
	}

	return "", fmt.Errorf("could not find dotfiles repository at ~/.dotfiles. Use --repo to specify the path")
}

func isValidRepo(path string) bool {
	if _, err := os.Stat(filepath.Join(path, "dotfiles.yaml")); err == nil {
		return true
	}
	if _, err := os.Stat(filepath.Join(path, "home")); err == nil {
		return true
	}
	return false
}

func expandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[1:])
		}
	}
	return path
}

func installPackages(plat platform.Info, repoPath string) error {
	ui.PrintInfo("Using Homebrew for package management")

	installer := packages.NewBrewInstaller(dryRunFlag)
	defer installer.StopSudo()

	if !installer.IsInstalled() {
		ui.PrintStep("Installing Homebrew...")
		if err := installer.Install(); err != nil {
			return err
		}
		ui.PrintSuccess("Homebrew installed")
	} else {
		ui.PrintSuccess("Homebrew already installed")
	}

	// Install core packages (all platforms)
	brewfile := filepath.Join(repoPath, "Brewfile")
	if _, err := os.Stat(brewfile); err == nil {
		ui.PrintStep("Installing packages from Brewfile...")
		if err := installer.InstallPackages(brewfile); err != nil {
			return err
		}
		ui.PrintSuccess("Core packages installed")
	} else {
		ui.PrintWarning("No Brewfile found, skipping core packages")
	}

	// Install macOS-specific packages (fonts, apps)
	if plat.IsMacOS() {
		brewfileMacOS := filepath.Join(repoPath, "Brewfile.macos")
		if _, err := os.Stat(brewfileMacOS); err == nil {
			ui.PrintStep("Installing macOS packages (fonts, apps)...")
			if err := installer.InstallPackages(brewfileMacOS); err != nil {
				return err
			}
			ui.PrintSuccess("macOS packages installed")
		}
	}

	return nil
}

func symlinkDotfiles(repoPath string, cfg *config.Config) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not get home directory: %w", err)
	}

	manager := symlink.NewManager(repoPath, homeDir, dryRunFlag)
	results := manager.LinkAll(cfg.Symlinks)

	var created, skipped, backedUp, errors int
	for _, result := range results {
		switch {
		case result.Error != nil:
			ui.PrintError(fmt.Sprintf("%s: %v", result.Path, result.Error))
			errors++
		case result.Action == "created" || result.Action == "would_create":
			ui.PrintSuccess(fmt.Sprintf("%s %s", result.Path, ui.SubtleStyle.Render("("+result.Action+")")))
			created++
		case result.Action == "backed_up":
			ui.PrintSuccess(fmt.Sprintf("%s %s", result.Path, ui.WarningStyle.Render("(backed up existing)")))
			backedUp++
		case result.Action == "skipped":
			ui.PrintStep(fmt.Sprintf("%s %s", result.Path, ui.SubtleStyle.Render("(already linked)")))
			skipped++
		}
	}

	if backupDir := manager.GetBackupDir(); backupDir != "" {
		ui.PrintWarning(fmt.Sprintf("Existing files backed up to: %s", backupDir))
	}

	ui.PrintInfo(fmt.Sprintf("Summary: %d created, %d skipped, %d backed up, %d errors", created, skipped, backedUp, errors))

	if errors > 0 {
		return fmt.Errorf("%d symlink errors occurred", errors)
	}
	return nil
}

func configureMacOS() error {
	defaults := macos.AllDefaults()
	var errors int

	for _, d := range defaults {
		if err := macos.Apply(d, dryRunFlag); err != nil {
			ui.PrintError(fmt.Sprintf("%s: %v", d.Desc, err))
			errors++
		} else {
			action := "set"
			if dryRunFlag {
				action = "would set"
			}
			ui.PrintSuccess(fmt.Sprintf("%s %s", d.Desc, ui.SubtleStyle.Render("("+action+")")))
		}
	}

	if err := macos.DisableSpotlightShortcuts(dryRunFlag); err != nil {
		ui.PrintError(fmt.Sprintf("Disable Spotlight keyboard shortcuts: %v", err))
		errors++
	} else {
		action := "set"
		if dryRunFlag {
			action = "would set"
		}
		ui.PrintSuccess(fmt.Sprintf("Disable Spotlight keyboard shortcuts for Raycast %s", ui.SubtleStyle.Render("("+action+")")))
	}

	if !dryRunFlag {
		ui.PrintStep("Restarting affected apps (Dock, Finder)...")
		if err := macos.RestartAffectedApps(dryRunFlag); err != nil {
			ui.PrintWarning(fmt.Sprintf("Failed to restart apps: %v", err))
		}
	}

	if errors > 0 {
		return fmt.Errorf("%d defaults failed to apply", errors)
	}
	return nil
}

func configureShell() error {
	if packages.IsZshDefault() {
		ui.PrintSuccess("zsh is already the default shell")
		return nil
	}

	ui.PrintStep("Setting zsh as default shell...")
	if err := packages.SetDefaultShell(dryRunFlag); err != nil {
		return err
	}
	ui.PrintSuccess("Default shell set to zsh")
	return nil
}

func printSummary(plat platform.Info) {
	ui.PrintHeader("Setup Complete!")
	fmt.Println()
	ui.PrintSuccess("🎉 Your development environment is ready!")
	fmt.Println()
	ui.PrintInfo("Next steps:")
	steps := []string{
		"Restart your terminal or run: exec zsh",
		"Run 'dotfiles ssh' to set up SSH keys from 1Password",
		"Run 'dotfiles ai' to install AI coding agents",
	}
	if plat.IsMacOS() {
		steps = append(steps,
			"Run 'dotfiles macos' to set computer name",
			"Set Caps Lock → Control in Settings → Keyboard → Keyboard Shortcuts → Modifier Keys",
			"Configure Trackpad in Settings → Trackpad (tap to click, three-finger drag, etc.)",
		)
	}
	ui.PrintList(steps)
}
