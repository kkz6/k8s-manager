package views

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/karthickk/k8s-manager/internal/ui/components"
)

// PodDetailsModel shows command output in a scrollable view
type PodDetailsModel struct {
	title     string
	viewport  viewport.Model
	ready     bool
	content   string
	err       error
	quitting  bool
}

// NewPodDetailsModel creates a new pod details view
func NewPodDetailsModel(title string, content string) *PodDetailsModel {
	return &PodDetailsModel{
		title:   title,
		content: content,
	}
}

// Init initializes the model
func (m *PodDetailsModel) Init() tea.Cmd {
	return nil
}

// Update handles updates
func (m *PodDetailsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		headerHeight := 4
		footerHeight := 3
		verticalMargin := headerHeight + footerHeight

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMargin)
			m.viewport.SetContent(m.content)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMargin
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "enter":
			m.quitting = true
			return m, tea.Quit
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View renders the view
func (m *PodDetailsModel) View() string {
	if m.quitting {
		return ""
	}

	if !m.ready {
		return "\n  Initializing..."
	}

	// Header
	header := components.BoxStyle.Width(m.viewport.Width).Render(
		components.TitleStyle.Render(m.title),
	)

	// Footer
	footerText := "↑/↓: scroll • q/esc/enter: back"
	if m.viewport.AtTop() && m.viewport.AtBottom() {
		footerText = "q/esc/enter: back"
	}
	footer := components.HelpStyle.Render(footerText)

	// Viewport with content
	content := components.BoxStyle.Width(m.viewport.Width).Height(m.viewport.Height).Render(m.viewport.View())

	// Combine everything
	return fmt.Sprintf("%s\n%s\n%s", header, content, footer)
}

// ExecuteKubectlCommand runs a kubectl command and returns the output
func ExecuteKubectlCommand(args ...string) (string, error) {
	cmd := exec.Command("kubectl", args...)
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut

	err := cmd.Run()
	if err != nil {
		if errOut.Len() > 0 {
			return "", fmt.Errorf("%s", errOut.String())
		}
		return "", err
	}

	return out.String(), nil
}

// podDetailsResultMsg contains the result of executing a command
type podDetailsResultMsg struct {
	title   string
	content string
	err     error
}

// actionCompletedMsg is sent when an action completes
type actionCompletedMsg struct{}

// ShowPodDetails shows pod details in a scrollable view
func ShowPodDetails(namespace, name, action string) tea.Cmd {
	return func() tea.Msg {
		var title, content string
		var err error

		switch action {
		case "describe":
			title = fmt.Sprintf("📋 Pod Description: %s", name)
			content, err = ExecuteKubectlCommand("describe", "pod", name, "-n", namespace)

		case "logs":
			title = fmt.Sprintf("📝 Pod Logs: %s (last 100 lines)", name)
			content, err = ExecuteKubectlCommand("logs", name, "-n", namespace, "--tail=100")

		case "yaml":
			title = fmt.Sprintf("📄 Pod YAML: %s", name)
			content, err = ExecuteKubectlCommand("get", "pod", name, "-n", namespace, "-o", "yaml")

		default:
			return nil
		}

		if err != nil {
			content = fmt.Sprintf("Error: %v", err)
		}

		// Clean up the content
		content = strings.TrimSpace(content)
		if content == "" {
			content = "No output available"
		}

		return podDetailsResultMsg{
			title:   title,
			content: content,
			err:     err,
		}
	}
}