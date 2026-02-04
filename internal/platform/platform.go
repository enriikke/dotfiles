package platform

import (
	"os"
	"runtime"
	"strings"
)

type Platform string

const (
	MacOS      Platform = "macos"
	Linux      Platform = "linux"
	Codespaces Platform = "codespaces"
	Unknown    Platform = "unknown"
)

type Info struct {
	Platform      Platform
	OS            string
	Arch          string
	IsCodespaces  bool
	IsRaspberryPi bool
}

func Detect() Info {
	info := Info{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}

	if os.Getenv("CODESPACES") != "" || os.Getenv("GITHUB_CODESPACE_TOKEN") != "" {
		info.IsCodespaces = true
		info.Platform = Codespaces
		return info
	}

	if runtime.GOOS == "linux" && (runtime.GOARCH == "arm64" || runtime.GOARCH == "arm") {
		if isRaspberryPi() {
			info.IsRaspberryPi = true
		}
	}

	switch runtime.GOOS {
	case "darwin":
		info.Platform = MacOS
	case "linux":
		info.Platform = Linux
	default:
		info.Platform = Unknown
	}

	return info
}

func isRaspberryPi() bool {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return false
	}
	content := strings.ToLower(string(data))
	return strings.Contains(content, "raspberry") || strings.Contains(content, "bcm2")
}

func (i Info) IsMacOS() bool {
	return i.Platform == MacOS
}

func (i Info) IsLinux() bool {
	return i.Platform == Linux || i.Platform == Codespaces
}

func (i Info) String() string {
	switch {
	case i.IsCodespaces:
		return "GitHub Codespaces"
	case i.IsRaspberryPi:
		return "Raspberry Pi"
	case i.Platform == MacOS:
		if i.Arch == "arm64" {
			return "macOS (Apple Silicon)"
		}
		return "macOS (Intel)"
	case i.Platform == Linux:
		return "Linux"
	default:
		return "Unknown"
	}
}
