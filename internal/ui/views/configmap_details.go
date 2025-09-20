package views

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/karthickk/k8s-manager/internal/services"
	"github.com/karthickk/k8s-manager/internal/ui/components"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConfigMapDetailsModel shows configmap details
type ConfigMapDetailsModel struct {
	namespace    string
	name         string
	configMap    *corev1.ConfigMap
	listView     *components.ListView
	viewport     viewport.Model
	viewMode     string // "list" or "detail"
	selectedKey  string
	ready        bool
	quitting     bool
	loading      bool
	errorMsg     string
}

// configMapLoadedMsg is sent when configmap is loaded
type configMapLoadedMsg struct {
	configMap *corev1.ConfigMap
	err       error
}

// NewConfigMapDetailsModel creates a new configmap details view
func NewConfigMapDetailsModel(namespace, name string) *ConfigMapDetailsModel {
	return &ConfigMapDetailsModel{
		namespace: namespace,
		name:      name,
		loading:   true,
		viewMode:  "list",
	}
}

// Init initializes the model
func (m *ConfigMapDetailsModel) Init() tea.Cmd {
	return m.loadConfigMap
}

// Update handles messages
func (m *ConfigMapDetailsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if m.viewMode == "detail" {
			headerHeight := 5
			footerHeight := 3
			verticalMargin := headerHeight + footerHeight

			if !m.ready {
				m.viewport = viewport.New(msg.Width, msg.Height-verticalMargin)
				m.viewport.Style = lipgloss.NewStyle()
				m.ready = true
			} else {
				m.viewport.Width = msg.Width
				m.viewport.Height = msg.Height - verticalMargin
			}
			m.updateViewport()
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q", "esc":
			if m.viewMode == "detail" {
				// Go back to list
				m.viewMode = "list"
				m.ready = false
				return m, nil
			}
			// Go back to ConfigMaps list
			return m, Navigate(ViewConfigMaps, nil)
		case "b":
			// Quick back navigation
			return m, Navigate(ViewConfigMaps, nil)
		case "e":
			if m.viewMode == "list" && m.listView != nil {
				selected := m.listView.GetSelected()
				if selected != nil {
					// TODO: Edit ConfigMap key
					return m, nil
				}
			}
		case "a":
			if m.viewMode == "list" {
				// TODO: Add new key to ConfigMap
				return m, nil
			}
		case "d":
			if m.viewMode == "list" && m.listView != nil {
				selected := m.listView.GetSelected()
				if selected != nil {
					// TODO: Delete ConfigMap key
					return m, nil
				}
			}
		case "enter", " ":
			if m.viewMode == "list" && m.listView != nil {
				selected := m.listView.GetSelected()
				if selected != nil {
					// View specific key
					m.selectedKey = selected.ID
					m.viewMode = "detail"
					return m, tea.WindowSize()
				}
			}
		}

		// Handle viewport controls in detail mode
		if m.viewMode == "detail" {
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}

	case configMapLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.errorMsg = msg.err.Error()
		} else {
			m.configMap = msg.configMap
			m.updateListView()
		}
		return m, nil
	}

	// Update list or viewport based on mode
	if m.viewMode == "list" && m.listView != nil && !m.loading {
		newList, cmd := m.listView.Update(msg)
		if list, ok := newList.(components.ListView); ok {
			m.listView = &list
		}
		return m, cmd
	} else if m.viewMode == "detail" && m.viewport.Height > 0 {
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	return m, nil
}

// View renders the view
func (m *ConfigMapDetailsModel) View() string {
	if m.quitting {
		return ""
	}

	if m.loading {
		return components.NewLoadingScreen("Loading ConfigMap Details").View()
	}

	if m.errorMsg != "" {
		return components.BoxStyle.Render(
			components.RenderTitle("ConfigMap Details", "") + "\n\n" +
				components.RenderMessage("error", m.errorMsg) + "\n\n" +
				components.HelpStyle.Render("Press 'q' to quit"),
		)
	}

	if m.viewMode == "detail" {
		return m.renderDetailView()
	}

	// List view
	if m.listView == nil {
		return "No configmap data available"
	}

	return m.listView.View()
}

// renderDetailView renders the detail view for a specific key
func (m *ConfigMapDetailsModel) renderDetailView() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	// Title
	title := fmt.Sprintf("📋 ConfigMap: %s / Key: %s", m.name, m.selectedKey)
	header := components.TitleStyle.Render(title)

	// Footer
	footerText := "q/esc: back to keys • ↑/↓: scroll"
	footer := components.HelpStyle.Render(footerText)

	return fmt.Sprintf("%s\n\n%s\n\n%s", header, m.viewport.View(), footer)
}

// updateListView updates the list with configmap keys
func (m *ConfigMapDetailsModel) updateListView() {
	if m.configMap == nil {
		return
	}

	listItems := []components.ListItem{}

	// Add configmap keys
	for key, value := range m.configMap.Data {
		lines := strings.Count(value, "\n") + 1
		size := len(value)
		description := fmt.Sprintf("Lines: %d, Size: %d bytes", lines, size)
		
		// Add preview if small enough
		if size <= 50 {
			preview := strings.ReplaceAll(value, "\n", "\\n")
			if len(preview) > 30 {
				preview = preview[:27] + "..."
			}
			description += fmt.Sprintf(" | %s", preview)
		}

		listItems = append(listItems, components.ListItem{
			ID:          key,
			Title:       key,
			Description: description,
			Icon:        "📄",
		})
	}

	title := fmt.Sprintf("📋 ConfigMap: %s", m.name)
	m.listView = components.NewListView(title, listItems)
	m.listView.SetHelpText("enter: view key • a: add key • e: edit • d: delete • esc/b: back • ctrl+c: quit")
}

// updateViewport updates the viewport with the selected key's content
func (m *ConfigMapDetailsModel) updateViewport() {
	if m.configMap == nil || m.selectedKey == "" {
		return
	}

	value, exists := m.configMap.Data[m.selectedKey]
	if !exists {
		m.viewport.SetContent("Key not found in configmap")
		return
	}

	// Format based on content type
	var content string
	if strings.Contains(m.selectedKey, "json") || isJSON(value) {
		// Pretty print JSON if possible
		content = formatJSON(value)
	} else if strings.Contains(m.selectedKey, "yaml") || strings.Contains(m.selectedKey, "yml") {
		// Keep YAML formatting
		content = value
	} else if strings.Contains(m.selectedKey, ".properties") {
		// Format properties files
		content = formatProperties(value)
	} else {
		// Regular text
		content = value
	}

	// Add some styling
	styledContent := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Render(content)

	m.viewport.SetContent(styledContent)
}

// loadConfigMap loads the configmap details
func (m *ConfigMapDetailsModel) loadConfigMap() tea.Msg {
	client, err := services.GetK8sClient()
	if err != nil {
		return configMapLoadedMsg{err: err}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	configMap, err := client.Clientset.CoreV1().ConfigMaps(m.namespace).Get(ctx, m.name, metav1.GetOptions{})
	if err != nil {
		return configMapLoadedMsg{err: err}
	}

	return configMapLoadedMsg{configMap: configMap}
}

// Helper functions

func formatProperties(props string) string {
	lines := strings.Split(props, "\n")
	var formatted []string
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			formatted = append(formatted, line)
			continue
		}
		
		// Add spacing around = for readability
		if idx := strings.Index(line, "="); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])
			formatted = append(formatted, fmt.Sprintf("%s = %s", key, value))
		} else {
			formatted = append(formatted, line)
		}
	}
	
	return strings.Join(formatted, "\n")
}

// ShowConfigMapDetails shows the configmap details view
func ShowConfigMapDetails(namespace, name string) tea.Cmd {
	return func() tea.Msg {
		model := NewConfigMapDetailsModel(namespace, name)
		p := tea.NewProgram(model, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return components.ErrorMsg{Error: err}
		}
		return nil
	}
}