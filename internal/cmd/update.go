package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/enriikke/dotfiles/internal/ui"
	"github.com/spf13/cobra"
)

const (
	releaseAPI    = "https://api.github.com/repos/enriikke/dotfiles/releases/latest"
	checkFile     = ".dotfiles-update-check"
	checkInterval = 24 * time.Hour
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update the dotfiles CLI to the latest version",
	RunE:  runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func runUpdate(cmd *cobra.Command, args []string) error {
	ui.PrintTitle("dotfiles update")
	fmt.Println()

	ui.PrintInfo("Current version: " + Version)

	release, err := fetchLatestRelease()
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	current := strings.TrimPrefix(Version, "v")

	if latest == current {
		ui.PrintSuccess("Already up to date!")
		return nil
	}

	ui.PrintInfo("Latest version: " + release.TagName)
	ui.PrintStep("Downloading...")

	asset := findAsset(release)
	if asset == nil {
		return fmt.Errorf("no binary found for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not determine executable path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("could not resolve executable path: %w", err)
	}

	if err := downloadAndReplace(asset.BrowserDownloadURL, execPath); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	saveLastCheck(latest)
	ui.PrintSuccess(fmt.Sprintf("Updated to %s!", release.TagName))
	return nil
}

func fetchLatestRelease() (*githubRelease, error) {
	resp, err := http.Get(releaseAPI)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	return &release, nil
}

func findAsset(release *githubRelease) *githubAsset {
	target := fmt.Sprintf("dotfiles_%s_%s_%s.tar.gz",
		strings.TrimPrefix(release.TagName, "v"),
		runtime.GOOS, runtime.GOARCH)

	for i, asset := range release.Assets {
		if asset.Name == target {
			return &release.Assets[i]
		}
	}
	return nil
}

func downloadAndReplace(url, destPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned %d", resp.StatusCode)
	}

	tmpDir, err := os.MkdirTemp("", "dotfiles-update-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir) //nolint:errcheck

	tarPath := filepath.Join(tmpDir, "dotfiles.tar.gz")
	f, err := os.Create(tarPath)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = f.Close()
		return err
	}
	_ = f.Close()

	// Extract tar.gz
	extract := fmt.Sprintf("tar -xzf %s -C %s", tarPath, tmpDir)
	cmd := exec.Command("/bin/sh", "-c", extract)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to extract: %w", err)
	}

	newBin := filepath.Join(tmpDir, "dotfiles")
	if _, err := os.Stat(newBin); err != nil {
		return fmt.Errorf("extracted binary not found")
	}

	// Replace current binary
	if err := os.Rename(newBin, destPath); err != nil {
		// Cross-device rename: fall back to copy
		return copyFile(newBin, destPath)
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close() //nolint:errcheck

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer out.Close() //nolint:errcheck

	_, err = io.Copy(out, in)
	return err
}

// CheckForUpdate prints a warning if a newer version is available.
// It caches the check result to avoid hitting the API on every invocation.
func CheckForUpdate() {
	if Version == "dev" {
		return
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return
	}
	cachePath := filepath.Join(cacheDir, checkFile)

	// Only check once per day
	if info, err := os.Stat(cachePath); err == nil {
		if time.Since(info.ModTime()) < checkInterval {
			// Read cached latest version
			data, err := os.ReadFile(cachePath)
			if err != nil {
				return
			}
			cached := strings.TrimSpace(string(data))
			current := strings.TrimPrefix(Version, "v")
			if cached != "" && cached != current {
				ui.PrintWarning(fmt.Sprintf("Update available: v%s → v%s (run 'dotfiles update')", current, cached))
			}
			return
		}
	}

	// Check in background to not slow down the command
	go func() {
		release, err := fetchLatestRelease()
		if err != nil {
			return
		}
		latest := strings.TrimPrefix(release.TagName, "v")
		_ = os.MkdirAll(filepath.Dir(cachePath), 0755)
		_ = os.WriteFile(cachePath, []byte(latest), 0644)
	}()
}

func saveLastCheck(version string) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return
	}
	cachePath := filepath.Join(cacheDir, checkFile)
	_ = os.MkdirAll(filepath.Dir(cachePath), 0755)
	_ = os.WriteFile(cachePath, []byte(version), 0644)
}
