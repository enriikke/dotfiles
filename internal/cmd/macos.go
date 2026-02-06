package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/charmbracelet/huh"
	"github.com/enriikke/dotfiles/internal/ui"
	"github.com/spf13/cobra"
)

var macosCmd = &cobra.Command{
	Use:   "macos",
	Short: "Configure macOS system settings interactively",
	Long:  `Prompts to set your computer name on macOS.`,
	RunE:  runMacOS,
}

func init() {
	rootCmd.AddCommand(macosCmd)
}

func runMacOS(cmd *cobra.Command, args []string) error {
	if runtime.GOOS != "darwin" {
		ui.PrintError("This command is only available on macOS")
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	ui.PrintTitle("dotfiles macos")
	fmt.Println()

	if err := promptComputerName(); err != nil {
		ui.PrintWarning(fmt.Sprintf("Computer name: %v", err))
	}

	fmt.Println()
	ui.PrintSuccess("🎉 macOS settings configured!")
	return nil
}

func promptComputerName() error {
	ui.PrintSection("Computer Name")

	// Show current name
	current, err := exec.Command("scutil", "--get", "ComputerName").Output()
	if err == nil {
		ui.PrintInfo(fmt.Sprintf("Current name: %s", string(current[:len(current)-1])))
	}

	var name string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Enter a computer name").
				Description("Used for hostname, AirDrop, Terminal prompt, etc.").
				Value(&name),
		),
	)

	if err := form.Run(); err != nil {
		return fmt.Errorf("cancelled: %w", err)
	}

	if name == "" {
		ui.PrintStep("Skipped (empty name)")
		return nil
	}

	cmds := []struct {
		desc string
		args []string
	}{
		{"ComputerName", []string{"scutil", "--set", "ComputerName", name}},
		{"HostName", []string{"scutil", "--set", "HostName", name}},
		{"LocalHostName", []string{"scutil", "--set", "LocalHostName", name}},
	}

	for _, c := range cmds {
		cmd := exec.Command("sudo", c.args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			ui.PrintError(fmt.Sprintf("Failed to set %s: %v", c.desc, err))
		} else {
			ui.PrintSuccess(fmt.Sprintf("Set %s → %s", c.desc, name))
		}
	}

	return nil
}
