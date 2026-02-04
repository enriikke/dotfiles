package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	primaryColor = lipgloss.Color("#7C3AED")
	successColor = lipgloss.Color("#10B981")
	warningColor = lipgloss.Color("#F59E0B")
	errorColor   = lipgloss.Color("#EF4444")
	subtleColor  = lipgloss.Color("#6B7280")

	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)

	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(successColor)

	WarningStyle = lipgloss.NewStyle().
			Foreground(warningColor)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(errorColor)

	SubtleStyle = lipgloss.NewStyle().
			Foreground(subtleColor)

	CheckIcon   = SuccessStyle.Render("✓")
	CrossIcon   = ErrorStyle.Render("✗")
	ArrowIcon   = SubtleStyle.Render("→")
	WarningIcon = WarningStyle.Render("⚠")
	InfoIcon    = lipgloss.NewStyle().Foreground(primaryColor).Render("•")
)

func PrintTitle(text string) {
	fmt.Println(TitleStyle.Render(text))
}

func PrintHeader(text string) {
	fmt.Println()
	fmt.Println(HeaderStyle.Render("═══ " + text + " ═══"))
}

func PrintSuccess(text string) {
	fmt.Printf("%s %s\n", CheckIcon, text)
}

func PrintWarning(text string) {
	fmt.Printf("%s %s\n", WarningIcon, text)
}

func PrintError(text string) {
	fmt.Printf("%s %s\n", CrossIcon, text)
}

func PrintInfo(text string) {
	fmt.Printf("%s %s\n", InfoIcon, text)
}

func PrintStep(text string) {
	fmt.Printf("  %s %s\n", ArrowIcon, text)
}

func PrintList(items []string) {
	for _, item := range items {
		fmt.Printf("  %s %s\n", SubtleStyle.Render("•"), item)
	}
}
