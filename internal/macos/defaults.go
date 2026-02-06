package macos

import (
	"os/exec"
)

// Default represents a single macOS defaults write command.
type Default struct {
	Domain string
	Key    string
	Type   string
	Value  string
	Desc   string
}

// AllDefaults returns the full set of sensible macOS defaults.
func AllDefaults() []Default {
	return []Default{
		// Dock & Mission Control
		{Domain: "com.apple.dock", Key: "mru-spaces", Type: "-bool", Value: "false", Desc: "Don't auto-rearrange Spaces based on recent use"},
		{Domain: "com.apple.dock", Key: "autohide", Type: "-bool", Value: "true", Desc: "Auto-hide the Dock"},
		{Domain: "com.apple.dock", Key: "show-recents", Type: "-bool", Value: "false", Desc: "Don't show recent apps in Dock"},
		{Domain: "com.apple.dock", Key: "minimize-to-application", Type: "-bool", Value: "true", Desc: "Minimize windows into app icon"},

		// Finder
		{Domain: "NSGlobalDomain", Key: "AppleShowAllExtensions", Type: "-bool", Value: "true", Desc: "Show all file extensions"},
		{Domain: "com.apple.finder", Key: "ShowPathbar", Type: "-bool", Value: "true", Desc: "Show path bar"},
		{Domain: "com.apple.finder", Key: "ShowStatusBar", Type: "-bool", Value: "true", Desc: "Show status bar"},
		{Domain: "com.apple.finder", Key: "FXDefaultSearchScope", Type: "-string", Value: "SCcf", Desc: "Search current folder by default"},
		{Domain: "com.apple.finder", Key: "_FXSortFoldersFirst", Type: "-bool", Value: "true", Desc: "Keep folders on top when sorting"},
		{Domain: "com.apple.finder", Key: "FXPreferredViewStyle", Type: "-string", Value: "clmv", Desc: "Use column view by default"},

		// Keyboard
		{Domain: "NSGlobalDomain", Key: "ApplePressAndHoldEnabled", Type: "-bool", Value: "false", Desc: "Disable press-and-hold, enable key repeat"},
		{Domain: "NSGlobalDomain", Key: "KeyRepeat", Type: "-int", Value: "2", Desc: "Fast key repeat rate"},
		{Domain: "NSGlobalDomain", Key: "InitialKeyRepeat", Type: "-int", Value: "15", Desc: "Short delay until key repeat"},
		{Domain: "NSGlobalDomain", Key: "NSAutomaticSpellingCorrectionEnabled", Type: "-bool", Value: "false", Desc: "Disable auto-correct"},

		// Screenshots
		{Domain: "com.apple.screencapture", Key: "type", Type: "-string", Value: "png", Desc: "Save screenshots as PNG"},
		{Domain: "com.apple.screencapture", Key: "disable-shadow", Type: "-bool", Value: "true", Desc: "Disable screenshot window shadow"},

		// General UX
		{Domain: "NSGlobalDomain", Key: "AppleShowScrollBars", Type: "-string", Value: "WhenScrolling", Desc: "Show scroll bars when scrolling"},
		{Domain: "NSGlobalDomain", Key: "NSNavPanelExpandedStateForSaveMode", Type: "-bool", Value: "true", Desc: "Expand save panel by default"},
		{Domain: "NSGlobalDomain", Key: "NSNavPanelExpandedStateForSaveMode2", Type: "-bool", Value: "true", Desc: "Expand save panel by default"},
		{Domain: "NSGlobalDomain", Key: "PMPrintingExpandedStateForPrint", Type: "-bool", Value: "true", Desc: "Expand print panel by default"},
		{Domain: "NSGlobalDomain", Key: "PMPrintingExpandedStateForPrint2", Type: "-bool", Value: "true", Desc: "Expand print panel by default"},
		{Domain: "com.apple.TextEdit", Key: "RichText", Type: "-int", Value: "0", Desc: "TextEdit uses plain text by default"},
	}
}

// Apply writes a single default and returns a description of what was done.
func Apply(d Default, dryRun bool) error {
	if dryRun {
		return nil
	}
	return exec.Command("defaults", "write", d.Domain, d.Key, d.Type, d.Value).Run()
}

// RestartAffectedApps restarts Dock and Finder so changes take effect.
func RestartAffectedApps(dryRun bool) error {
	if dryRun {
		return nil
	}
	for _, app := range []string{"Dock", "Finder", "SystemUIServer"} {
		// Not fatal — the app may not be running
		_ = exec.Command("killall", app).Run()
	}
	return nil
}
