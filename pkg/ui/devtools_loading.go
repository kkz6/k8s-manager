package ui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Spinner frames for animated loading
var spinnerFrames = []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
var dotFrames = []string{"   ", ".  ", ".. ", "..."}
var barFrames = []string{"▱▱▱", "▰▱▱", "▰▰▱", "▰▰▰", "▱▰▰", "▱▱▰", "▱▱▱"}

// AnimatedSpinner represents an animated loading indicator
type AnimatedSpinner struct {
	frame   int
	style   string // "spinner", "dots", "bar"
	message string
}

// NewAnimatedSpinner creates a new loading spinner
func NewAnimatedSpinner(style string, message string) AnimatedSpinner {
	return AnimatedSpinner{
		frame:   0,
		style:   style,
		message: message,
	}
}

// spinnerTickMsg is sent for animation updates
type spinnerTickMsg time.Time

// Init starts the animation ticker
func (s AnimatedSpinner) Init() tea.Cmd {
	return s.tick()
}

// tick returns a command that sends a tick message after a delay
func (s AnimatedSpinner) tick() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return spinnerTickMsg(t)
	})
}

// Update handles tick messages to update the animation frame
func (s AnimatedSpinner) Update(msg tea.Msg) (AnimatedSpinner, tea.Cmd) {
	switch msg.(type) {
	case spinnerTickMsg:
		s.frame++
		return s, s.tick()
	}
	return s, nil
}

// View renders the current spinner frame
func (s AnimatedSpinner) View() string {
	var frames []string
	switch s.style {
	case "dots":
		frames = dotFrames
	case "bar":
		frames = barFrames
	default:
		frames = spinnerFrames
	}

	frame := frames[s.frame%len(frames)]

	spinnerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")). // Cyan color
		Bold(true)

	messageStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")) // Gray

	if s.message != "" {
		return spinnerStyle.Render(frame) + " " + messageStyle.Render(s.message)
	}
	return spinnerStyle.Render(frame)
}

// LoadingOverlay creates a full-screen loading overlay
func LoadingOverlay(message string, width, height int) string {
	spinner := NewAnimatedSpinner("spinner", message)
	content := spinner.View()

	overlayStyle := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Align(lipgloss.Center, lipgloss.Center)

	return overlayStyle.Render(content)
}

// InlineLoadingText creates an inline loading text with animation
func InlineLoadingText(message string, frame int) string {
	dots := strings.Repeat(".", (frame%4))
	spaces := strings.Repeat(" ", 3-len(dots))

	loadingStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243"))

	return loadingStyle.Render(message + dots + spaces)
}