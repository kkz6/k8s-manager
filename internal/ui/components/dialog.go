package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DialogType represents the type of dialog
type DialogType int

const (
	DialogTypeConfirm DialogType = iota
	DialogTypeInfo
	DialogTypeError
	DialogTypeWarning
	DialogTypeSuccess
)

// Dialog is a reusable dialog component
type Dialog struct {
	Title      string
	Message    string
	Type       DialogType
	Options    []string
	selected   int
	width      int
	height     int
	onConfirm  func() tea.Cmd
	onCancel   func() tea.Cmd
}

// NewConfirmDialog creates a new confirmation dialog
func NewConfirmDialog(title, message string) *Dialog {
	return &Dialog{
		Title:   title,
		Message: message,
		Type:    DialogTypeConfirm,
		Options: []string{"Yes", "No"},
		selected: 1, // Default to "No" for safety
	}
}

// NewInfoDialog creates a new information dialog
func NewInfoDialog(title, message string) *Dialog {
	return &Dialog{
		Title:   title,
		Message: message,
		Type:    DialogTypeInfo,
		Options: []string{"OK"},
	}
}

// NewErrorDialog creates a new error dialog
func NewErrorDialog(title, message string) *Dialog {
	return &Dialog{
		Title:   title,
		Message: message,
		Type:    DialogTypeError,
		Options: []string{"OK"},
	}
}

// SetCallbacks sets the confirmation callbacks
func (d *Dialog) SetCallbacks(onConfirm, onCancel func() tea.Cmd) {
	d.onConfirm = onConfirm
	d.onCancel = onCancel
}

// Init initializes the dialog
func (d Dialog) Init() tea.Cmd {
	return nil
}

// Update handles dialog updates
func (d Dialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.width = msg.Width
		d.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h":
			if d.selected > 0 {
				d.selected--
			}

		case "right", "l":
			if d.selected < len(d.Options)-1 {
				d.selected++
			}

		case "tab":
			d.selected = (d.selected + 1) % len(d.Options)

		case "enter", " ":
			if d.Type == DialogTypeConfirm {
				if d.selected == 0 && d.onConfirm != nil {
					return d, d.onConfirm()
				} else if d.selected == 1 && d.onCancel != nil {
					return d, d.onCancel()
				}
			}
			return d, tea.Quit

		case "y", "Y":
			if d.Type == DialogTypeConfirm && d.onConfirm != nil {
				return d, d.onConfirm()
			}

		case "n", "N":
			if d.Type == DialogTypeConfirm && d.onCancel != nil {
				return d, d.onCancel()
			}

		case "esc":
			if d.onCancel != nil {
				return d, d.onCancel()
			}
			return d, tea.Quit
		}
	}

	return d, nil
}

// View renders the dialog
func (d Dialog) View() string {
	// Determine dialog style based on type
	var borderColor lipgloss.Color
	var icon string
	switch d.Type {
	case DialogTypeConfirm:
		borderColor = ColorWarning
		icon = "?"
	case DialogTypeInfo:
		borderColor = ColorInfo
		icon = "ℹ"
	case DialogTypeError:
		borderColor = ColorError
		icon = "✗"
	case DialogTypeWarning:
		borderColor = ColorWarning
		icon = "⚠"
	case DialogTypeSuccess:
		borderColor = ColorSuccess
		icon = "✓"
	}

	// Container style
	containerStyle := lipgloss.NewStyle().
		Width(50).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(2)

	var content strings.Builder

	// Title with icon
	titleWithIcon := fmt.Sprintf("%s %s", icon, d.Title)
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(borderColor).
		Width(46).
		Align(lipgloss.Center)
	content.WriteString(titleStyle.Render(titleWithIcon))
	content.WriteString("\n\n")

	// Message
	messageStyle := ItemStyle.Width(46).Align(lipgloss.Center)
	content.WriteString(messageStyle.Render(d.Message))
	content.WriteString("\n\n")

	// Options
	if len(d.Options) > 0 {
		var optionsLine strings.Builder
		for i, option := range d.Options {
			if i > 0 {
				optionsLine.WriteString("    ")
			}

			optionStyle := lipgloss.NewStyle().
				Padding(0, 2).
				Border(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("240"))

			if i == d.selected {
				optionStyle = optionStyle.
					Background(borderColor).
					Foreground(lipgloss.Color("0")).
					Bold(true).
					BorderForeground(borderColor)
			}

			optionsLine.WriteString(optionStyle.Render(option))
		}

		// Center the options
		optionsStr := optionsLine.String()
		optionsStyle := lipgloss.NewStyle().Width(46).Align(lipgloss.Center)
		content.WriteString(optionsStyle.Render(optionsStr))
	}

	// Help text for confirm dialogs
	if d.Type == DialogTypeConfirm {
		content.WriteString("\n\n")
		helpStyle := DescriptionStyle.Width(46).Align(lipgloss.Center)
		content.WriteString(helpStyle.Render("Press Y for Yes, N for No"))
	}

	// Center the dialog on screen
	dialogContent := containerStyle.Render(content.String())
	
	// Create overlay effect
	overlayStyle := lipgloss.NewStyle().
		Width(d.width).
		Height(d.height).
		Align(lipgloss.Center, lipgloss.Center)

	return overlayStyle.Render(dialogContent)
}

// ProgressDialog shows a progress indicator
type ProgressDialog struct {
	Title    string
	Message  string
	Progress float64 // 0.0 to 1.0
	spinner  SpinnerModel
}

// NewProgressDialog creates a new progress dialog
func NewProgressDialog(title, message string) *ProgressDialog {
	return &ProgressDialog{
		Title:   title,
		Message: message,
		spinner: NewSpinner(""),
	}
}

// SetProgress sets the progress value
func (p *ProgressDialog) SetProgress(progress float64) {
	p.Progress = progress
}

// Init initializes the progress dialog
func (p ProgressDialog) Init() tea.Cmd {
	return p.spinner.Init()
}

// Update handles progress dialog updates
func (p ProgressDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	p.spinner, cmd = p.spinner.Update(msg)
	return p, cmd
}

// View renders the progress dialog
func (p ProgressDialog) View() string {
	containerStyle := lipgloss.NewStyle().
		Width(60).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorInfo).
		Padding(2)

	var content strings.Builder

	// Title
	titleStyle := TitleStyle.Width(56).Align(lipgloss.Center)
	content.WriteString(titleStyle.Render(p.Title))
	content.WriteString("\n\n")

	// Progress bar
	barWidth := 50
	filled := int(p.Progress * float64(barWidth))
	empty := barWidth - filled

	progressBar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	progressStyle := lipgloss.NewStyle().Foreground(ColorSuccess)
	content.WriteString(progressStyle.Render(progressBar))
	content.WriteString("\n")

	// Percentage
	percentage := fmt.Sprintf("%.0f%%", p.Progress*100)
	percentStyle := ItemStyle.Width(56).Align(lipgloss.Center)
	content.WriteString(percentStyle.Render(percentage))
	content.WriteString("\n\n")

	// Message with spinner
	if p.Message != "" {
		messageStyle := DescriptionStyle.Width(56).Align(lipgloss.Center)
		messageWithSpinner := fmt.Sprintf("%s %s", p.spinner.View(), p.Message)
		content.WriteString(messageStyle.Render(messageWithSpinner))
	}

	return containerStyle.Render(content.String())
}