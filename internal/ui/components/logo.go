package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Logo represents the application logo
type Logo struct {
	Title       string
	Subtitle    string
	Author      string
	Email       string
	Website     string
	ShowCredits bool
}

// NewLogo creates a new logo component
func NewLogo(title string) *Logo {
	return &Logo{
		Title:       title,
		ShowCredits: true,
	}
}

// LogoStyle returns the styled logo
func (l *Logo) Render() string {
	var b strings.Builder

	// K8S MANAGER ASCII art with gradient colors
	logoArt := []string{
		"██   ██  █████  ███████     ███    ███  █████  ███    ██  █████   ██████  ███████ ██████",
		"██  ██  ██   ██ ██          ████  ████ ██   ██ ████   ██ ██   ██ ██       ██      ██   ██",
		"█████    █████  ███████     ██ ████ ██ ███████ ██ ██  ██ ███████ ██   ███ █████   ██████",
		"██  ██  ██   ██      ██     ██  ██  ██ ██   ██ ██  ██ ██ ██   ██ ██    ██ ██      ██   ██",
		"██   ██  █████  ███████     ██      ██ ██   ██ ██   ████ ██   ██  ██████  ███████ ██   ██",
	}

	// Apply gradient colors (purple to cyan)
	colors := []string{"#8B5CF6", "#7C3AED", "#6D28D9", "#5B21B6", "#4C1D95"}
	for i, line := range logoArt {
		style := lipgloss.NewStyle().Foreground(lipgloss.Color(colors[i]))
		b.WriteString(style.Render(line))
		b.WriteString("\n")
	}

	// Subtitle
	if l.Subtitle != "" {
		b.WriteString("\n")
		subtitleStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Italic(true).
			MarginLeft(2)
		b.WriteString(subtitleStyle.Render(l.Subtitle))
		b.WriteString("\n")
	}

	// Credits box
	if l.ShowCredits && (l.Author != "" || l.Email != "" || l.Website != "") {
		b.WriteString("\n")
		
		creditBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 2).
			Width(44)

		var credits []string
		if l.Author != "" {
			credits = append(credits, "Created by: "+l.Author)
		}
		if l.Email != "" {
			credits = append(credits, "Email: "+l.Email)
		}
		if l.Website != "" {
			credits = append(credits, "Website: "+l.Website)
		}

		creditText := strings.Join(credits, "\n")
		b.WriteString(creditBox.Render(creditText))
		b.WriteString("\n")
	}

	// Center everything
	width := 90
	lines := strings.Split(b.String(), "\n")
	var centered strings.Builder
	
	for _, line := range lines {
		if line != "" {
			lineWidth := lipgloss.Width(line)
			if lineWidth < width {
				padding := (width - lineWidth) / 2
				centered.WriteString(strings.Repeat(" ", padding))
			}
			centered.WriteString(line)
			centered.WriteString("\n")
		}
	}

	return centered.String()
}

// WelcomeMessage returns a styled welcome message
func WelcomeMessage(message string) string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Bold(true).
		MarginTop(1).
		MarginBottom(2)
	
	return style.Render("✨ " + message)
}