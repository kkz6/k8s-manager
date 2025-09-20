package components

import (
	"github.com/charmbracelet/lipgloss"
)

// Color palette
var (
	ColorPrimary   = lipgloss.Color("39")   // Blue
	ColorSecondary = lipgloss.Color("205")  // Pink
	ColorSuccess   = lipgloss.Color("42")   // Green
	ColorError     = lipgloss.Color("196")  // Red
	ColorWarning   = lipgloss.Color("214")  // Orange
	ColorInfo      = lipgloss.Color("86")   // Cyan
	ColorMuted     = lipgloss.Color("244")  // Gray
	ColorBorder    = lipgloss.Color("62")   // Dark blue
	ColorHighlight = lipgloss.Color("170")  // Purple
)

// Common styles
var (
	// Title and header styles
	TitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary).
		Background(lipgloss.Color("235")).
		Padding(0, 2).
		MarginBottom(1)

	SubtitleStyle = lipgloss.NewStyle().
		Foreground(ColorMuted).
		MarginBottom(1)

	HeaderStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorInfo).
		MarginBottom(1)

	// Container styles
	BoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Padding(1, 2).
		Margin(1, 0)

	InfoBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 2).
		MarginBottom(1)

	// Text styles
	SelectedStyle = lipgloss.NewStyle().
		Foreground(ColorHighlight).
		Background(lipgloss.Color("236")).
		Bold(true)

	ItemStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	DescriptionStyle = lipgloss.NewStyle().
		Foreground(ColorMuted).
		Italic(true)

	NumberStyle = lipgloss.NewStyle().
		Foreground(ColorSecondary).
		Bold(true)

	// Status styles
	StatusRunningStyle = lipgloss.NewStyle().
		Foreground(ColorSuccess).
		Bold(true)

	StatusPendingStyle = lipgloss.NewStyle().
		Foreground(ColorWarning).
		Bold(true)

	StatusErrorStyle = lipgloss.NewStyle().
		Foreground(ColorError).
		Bold(true)

	// Message styles
	SuccessMessageStyle = lipgloss.NewStyle().
		Foreground(ColorSuccess).
		Bold(true)

	ErrorMessageStyle = lipgloss.NewStyle().
		Foreground(ColorError).
		Bold(true)

	WarningMessageStyle = lipgloss.NewStyle().
		Foreground(ColorWarning).
		Bold(true)

	InfoMessageStyle = lipgloss.NewStyle().
		Foreground(ColorInfo)

	// Help styles
	HelpStyle = lipgloss.NewStyle().
		Foreground(ColorMuted).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(lipgloss.Color("240")).
		PaddingTop(1).
		MarginTop(1)

	KeyStyle = lipgloss.NewStyle().
		Foreground(ColorSecondary).
		Bold(true)

	// App container
	AppStyle = lipgloss.NewStyle().
		Padding(1, 2)
)

// Helper functions for consistent styling
func RenderTitle(title, subtitle string) string {
	var result string
	result += TitleStyle.Render(title)
	if subtitle != "" {
		result += "\n" + SubtitleStyle.Render(subtitle)
	}
	return result
}

func RenderStatus(status string) string {
	switch status {
	case "Running", "Succeeded", "Active", "Ready":
		return StatusRunningStyle.Render(status)
	case "Pending", "Creating", "Updating":
		return StatusPendingStyle.Render(status)
	case "Failed", "Error", "CrashLoopBackOff", "Terminating":
		return StatusErrorStyle.Render(status)
	default:
		return ItemStyle.Render(status)
	}
}

func RenderMessage(messageType, message string) string {
	switch messageType {
	case "success":
		return SuccessMessageStyle.Render("✓ " + message)
	case "error":
		return ErrorMessageStyle.Render("✗ " + message)
	case "warning":
		return WarningMessageStyle.Render("⚠ " + message)
	case "info":
		return InfoMessageStyle.Render("ℹ " + message)
	default:
		return ItemStyle.Render(message)
	}
}

func RenderKeyBinding(key, description string) string {
	return KeyStyle.Render(key) + " " + DescriptionStyle.Render(description)
}