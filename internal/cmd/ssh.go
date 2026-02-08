package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/enriikke/dotfiles/internal/onepassword"
	"github.com/enriikke/dotfiles/internal/ui"
	"github.com/spf13/cobra"
)

var (
	sshDryRun bool
	sshAll    bool
)

func init() {
	sshCmd.Flags().BoolVar(&sshDryRun, "dry-run", false, "Preview changes without making them")
	sshCmd.Flags().BoolVar(&sshAll, "all", false, "Download all keys and configs without prompting")
}

var sshCmd = &cobra.Command{
	Use:   "ssh",
	Short: "Set up SSH keys from 1Password",
	Long:  `Download SSH keys and config files from 1Password to ~/.ssh/`,
	RunE:  runSSH,
}

// Item types for the multi-select
type sshItem struct {
	Type  string // "key" or "config"
	Title string
	Name  string // filename for configs
}

func runSSH(cmd *cobra.Command, args []string) error {
	ui.PrintTitle("dotfiles ssh")
	fmt.Println()

	// Check prerequisites
	ui.PrintSection("Checking Prerequisites")
	op := onepassword.NewClient()

	if !op.IsInstalled() {
		ui.PrintError("1Password CLI (op) not found")
		ui.PrintInfo("Install with: brew install 1password-cli")
		return fmt.Errorf("1Password CLI not installed")
	}
	ui.PrintSuccess("1Password CLI found")

	if !op.IsSignedIn() {
		ui.PrintError("1Password CLI not signed in")
		ui.PrintInfo("Sign in with: op signin")
		return fmt.Errorf("1Password CLI not signed in")
	}
	ui.PrintSuccess("1Password CLI signed in")
	fmt.Println()

	// Gather available items
	ui.PrintSection("Discovering Available Items")

	var items []sshItem

	// List SSH keys
	keys, err := op.ListSSHKeys()
	if err != nil {
		ui.PrintWarning("Could not list SSH keys: " + err.Error())
	} else {
		for _, key := range keys {
			items = append(items, sshItem{Type: "key", Title: key.Title})
		}
		ui.PrintSuccess(fmt.Sprintf("Found %d SSH key(s)", len(keys)))
	}

	// Check for config notes
	configs := []struct {
		title string
		name  string
	}{
		{"SSH Work Config", "config.work"},
		{"SSH Personal Config", "config.personal"},
	}

	for _, cfg := range configs {
		if op.SecureNoteExists(cfg.title) {
			items = append(items, sshItem{Type: "config", Title: cfg.title, Name: cfg.name})
			ui.PrintSuccess(fmt.Sprintf("Found config: %s", cfg.title))
		}
	}

	if len(items) == 0 {
		ui.PrintError("No SSH keys or configs found in 1Password")
		return fmt.Errorf("nothing to download")
	}
	fmt.Println()

	// Select items to download
	var selected []string

	if sshAll {
		// Select all items
		for _, item := range items {
			selected = append(selected, item.Title)
		}
	} else {
		// Build options for multi-select
		var options []huh.Option[string]
		for _, item := range items {
			var label string
			if item.Type == "config" {
				label = fmt.Sprintf("%s → %s", item.Title, item.Name)
			} else {
				label = fmt.Sprintf("🔑 %s", item.Title)
			}
			options = append(options, huh.NewOption(label, item.Title).Selected(true))
		}

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewMultiSelect[string]().
					Title("Select items to download").
					Description("Use space to toggle, enter to confirm").
					Options(options...).
					Value(&selected),
			),
		)

		if err := form.Run(); err != nil {
			return fmt.Errorf("selection cancelled: %w", err)
		}
	}

	if len(selected) == 0 {
		ui.PrintWarning("No items selected")
		return nil
	}

	// Create selected set for lookup
	selectedSet := make(map[string]bool)
	for _, s := range selected {
		selectedSet[s] = true
	}

	if sshDryRun {
		ui.PrintWarning("Dry run mode - no changes will be made")
	}
	fmt.Println()

	// Ensure directories exist
	sshDir := filepath.Join(os.Getenv("HOME"), ".ssh")
	keyDir := filepath.Join(sshDir, "keys")

	if !sshDryRun {
		if err := os.MkdirAll(keyDir, 0700); err != nil {
			return fmt.Errorf("failed to create %s: %w", keyDir, err)
		}
	}

	// Process selected items
	ui.PrintSection("Downloading Selected Items")

	var successCount, errorCount int

	type downloadedKey struct {
		Title    string
		Filename string
	}
	var downloadedKeys []downloadedKey

	for _, item := range items {
		if !selectedSet[item.Title] {
			continue
		}

		if item.Type == "key" {
			filename, err := downloadSSHKey(op, item.Title, keyDir, sshDryRun)
			if err != nil {
				ui.PrintError(fmt.Sprintf("%s: %v", item.Title, err))
				errorCount++
			} else {
				downloadedKeys = append(downloadedKeys, downloadedKey{Title: item.Title, Filename: filename})
				successCount++
			}
		} else {
			destPath := filepath.Join(sshDir, item.Name)
			if err := downloadConfig(op, item.Title, destPath, sshDryRun); err != nil {
				ui.PrintError(fmt.Sprintf("%s: %v", item.Title, err))
				errorCount++
			} else {
				successCount++
			}
		}
	}

	// Set up key alias for github.com
	if len(downloadedKeys) > 0 {
		fmt.Println()
		ui.PrintSection("Key Aliases")

		var githubKeyFilename string

		if sshAll && len(downloadedKeys) > 1 {
			ui.PrintInfo("Run without --all to select the key alias for github.com")
		} else if len(downloadedKeys) == 1 {
			githubKeyFilename = downloadedKeys[0].Filename
			ui.PrintInfo(fmt.Sprintf("Using %s for github.com", downloadedKeys[0].Title))
		} else {
			var options []huh.Option[string]
			for _, k := range downloadedKeys {
				options = append(options, huh.NewOption(k.Title, k.Filename))
			}

			form := huh.NewForm(
				huh.NewGroup(
					huh.NewSelect[string]().
						Title("Which key should be used for github.com?").
						Options(options...).
						Value(&githubKeyFilename),
				),
			)

			if err := form.Run(); err != nil {
				ui.PrintWarning("Skipping key alias: " + err.Error())
			}
		}

		if githubKeyFilename != "" {
			createKeyAlias(keyDir, "github", githubKeyFilename, sshDryRun)
		}
	}

	// Summary
	fmt.Println()
	ui.PrintSection("Summary")

	if errorCount == 0 {
		ui.PrintSuccess(fmt.Sprintf("Downloaded %d item(s) to ~/.ssh/", successCount))
	} else {
		ui.PrintWarning(fmt.Sprintf("Downloaded %d item(s), %d failed", successCount, errorCount))
	}

	if !sshDryRun {
		fmt.Println()
		ui.PrintInfo("Test with: ssh -T git@github.com")
	}

	return nil
}

func downloadSSHKey(op *onepassword.Client, title, keyDir string, dryRun bool) (string, error) {
	// Determine key type and filename
	keyType := op.GetKeyType(title)
	prefix := "id_key"
	switch keyType {
	case "ed25519":
		prefix = "id_ed25519"
	case "rsa":
		prefix = "id_rsa"
	}

	safeName := sanitizeFilename(title)
	privatePath := filepath.Join(keyDir, fmt.Sprintf("%s_%s", prefix, safeName))
	publicPath := privatePath + ".pub"
	baseName := filepath.Base(privatePath)

	if dryRun {
		ui.PrintSuccess(fmt.Sprintf("%s → %s", title, baseName))
		return baseName, nil
	}

	// Download private key
	privateKey, err := op.GetPrivateKey(title)
	if err != nil {
		return "", fmt.Errorf("failed to get private key: %w", err)
	}

	if err := os.WriteFile(privatePath, []byte(privateKey), 0600); err != nil {
		return "", fmt.Errorf("failed to write private key: %w", err)
	}

	// Download public key
	publicKey, err := op.GetPublicKey(title)
	if err != nil {
		// Public key is optional, just warn
		ui.PrintWarning(fmt.Sprintf("%s: no public key found", title))
	} else {
		if err := os.WriteFile(publicPath, []byte(publicKey), 0644); err != nil {
			return "", fmt.Errorf("failed to write public key: %w", err)
		}
	}

	ui.PrintSuccess(fmt.Sprintf("%s → %s", title, baseName))
	return baseName, nil
}

func downloadConfig(op *onepassword.Client, title, destPath string, dryRun bool) error {
	if dryRun {
		ui.PrintSuccess(fmt.Sprintf("%s → %s", title, filepath.Base(destPath)))
		return nil
	}

	content, err := op.GetSecureNote(title)
	if err != nil {
		return err
	}

	if err := os.WriteFile(destPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	ui.PrintSuccess(fmt.Sprintf("%s → %s", title, filepath.Base(destPath)))
	return nil
}

func sanitizeFilename(s string) string {
	s = strings.ReplaceAll(s, " ", "_")
	reg := regexp.MustCompile(`[^a-zA-Z0-9._-]`)
	return reg.ReplaceAllString(s, "")
}

func createKeyAlias(keyDir, alias, target string, dryRun bool) {
	aliasPath := filepath.Join(keyDir, alias)
	aliasPubPath := aliasPath + ".pub"
	targetPub := target + ".pub"

	if dryRun {
		ui.PrintSuccess(fmt.Sprintf("%s → %s", alias, target))
		if _, err := os.Stat(filepath.Join(keyDir, targetPub)); err == nil {
			ui.PrintSuccess(fmt.Sprintf("%s.pub → %s", alias, targetPub))
		}
		return
	}

	// Remove existing symlinks
	_ = os.Remove(aliasPath)
	_ = os.Remove(aliasPubPath)

	if err := os.Symlink(target, aliasPath); err != nil {
		ui.PrintError(fmt.Sprintf("Failed to create symlink: %v", err))
		return
	}
	ui.PrintSuccess(fmt.Sprintf("%s → %s", alias, target))

	// Symlink public key if it exists
	if _, err := os.Stat(filepath.Join(keyDir, targetPub)); err == nil {
		if err := os.Symlink(targetPub, aliasPubPath); err != nil {
			ui.PrintError(fmt.Sprintf("Failed to create public key symlink: %v", err))
		} else {
			ui.PrintSuccess(fmt.Sprintf("%s.pub → %s", alias, targetPub))
		}
	}
}
