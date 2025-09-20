package components

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SpinnerModel is a reusable spinner component
type SpinnerModel struct {
	spinner     spinner.Model
	message     string
	showSpinner bool
}

// NewSpinner creates a new spinner with a message
func NewSpinner(message string) SpinnerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorSecondary)
	return SpinnerModel{
		spinner:     s,
		message:     message,
		showSpinner: true,
	}
}

// Init initializes the spinner
func (m SpinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update handles spinner updates
func (m SpinnerModel) Update(msg tea.Msg) (SpinnerModel, tea.Cmd) {
	if !m.showSpinner {
		return m, nil
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

// View renders the spinner
func (m SpinnerModel) View() string {
	if !m.showSpinner {
		return ""
	}
	return fmt.Sprintf("%s %s", m.spinner.View(), m.message)
}

// SetMessage updates the spinner message
func (m *SpinnerModel) SetMessage(message string) {
	m.message = message
}

// Show shows the spinner
func (m *SpinnerModel) Show() {
	m.showSpinner = true
}

// Hide hides the spinner
func (m *SpinnerModel) Hide() {
	m.showSpinner = false
}

// LoadingScreen creates a full-screen loading display
type LoadingScreen struct {
	spinner SpinnerModel
	title   string
}

// NewLoadingScreen creates a new loading screen
func NewLoadingScreen(title string) LoadingScreen {
	return LoadingScreen{
		spinner: NewSpinner("Loading..."),
		title:   title,
	}
}

// Init initializes the loading screen
func (m LoadingScreen) Init() tea.Cmd {
	return m.spinner.Init()
}

// Update handles loading screen updates
func (m LoadingScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

// View renders the loading screen
func (m LoadingScreen) View() string {
	containerStyle := lipgloss.NewStyle().
		Width(60).
		Height(10).
		Align(lipgloss.Center, lipgloss.Center).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Padding(2)

	content := fmt.Sprintf("%s\n\n%s\n\n%s",
		TitleStyle.Render(m.title),
		m.spinner.View(),
		DescriptionStyle.Render("Please wait..."))

	return AppStyle.Render(containerStyle.Render(content))
}