package views

import (
	"context"
	"fmt"
	"sort"
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
	table        *components.Table
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
		if m.loading {
			return m, nil
		}

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
			if m.viewMode == "detail" {
				m.viewMode = "list"
				m.ready = false
				return m, nil
			}
			return m, Navigate(ViewConfigMaps, nil)
		case "e":
			if m.viewMode == "list" && m.table != nil {
				selected := m.table.GetSelected()
				if len(selected) > 0 {
					// Navigate to edit key view
					return m, Navigate(ViewEditConfigMapKey, map[string]string{
						"namespace": m.namespace,
						"name":      m.name,
						"key":       selected[0],
					})
				}
			}
		case "a":
			if m.viewMode == "list" {
				// Navigate to add key view
				return m, Navigate(ViewAddConfigMapKey, map[string]string{
					"namespace": m.namespace,
					"name":      m.name,
				})
			}
		case "enter", " ":
			if m.viewMode == "list" && m.table != nil {
				selected := m.table.GetSelected()
				if len(selected) > 0 {
					// View specific key
					m.selectedKey = selected[0]
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
			m.updateTable()
		}
		return m, nil
	}

	// Update table in list mode
	if m.viewMode == "list" && m.table != nil && !m.loading {
		newTable, cmd := m.table.Update(msg)
		m.table = newTable.(*components.Table)
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
		loadingStyle := components.InfoMessageStyle.Copy().
			Padding(2, 4)
		return "\n" + loadingStyle.Render("⏳ Loading ConfigMap...") + "\n\n"
	}

	if m.errorMsg != "" {
		errorStyle := components.ErrorMessageStyle.Copy().
			Padding(1, 2)
		helpStyle := components.HelpStyle.Copy().
			Padding(1, 2)
		return "\n" + errorStyle.Render(fmt.Sprintf("✗ Error: %s", m.errorMsg)) + "\n" +
			helpStyle.Render("Press 'q' to back") + "\n"
	}

	if m.viewMode == "detail" {
		return m.renderDetailView()
	}

	// List view
	if m.table == nil {
		emptyStyle := components.DescriptionStyle.Copy().
			Padding(2, 4)
		return "\n" + emptyStyle.Render("No configmap data available") + "\n"
	}

	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(m.table.View())
	b.WriteString("\n\n")

	// Help footer
	helpStyle := components.HelpStyle.Copy().
		Foreground(components.ColorMuted).
		Padding(1, 2)

	helpText := components.RenderKeyBinding("↑/↓/j/k", "navigate") + " • " +
		components.RenderKeyBinding("enter", "view value") + " • " +
		components.RenderKeyBinding("e", "edit") + " • " +
		components.RenderKeyBinding("a", "add") + " • " +
		components.RenderKeyBinding("q", "back")

	b.WriteString(helpStyle.Render(helpText))
	b.WriteString("\n")

	return b.String()
}

// renderDetailView renders the detail view for a specific key
func (m *ConfigMapDetailsModel) renderDetailView() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(components.ColorPrimary).
		Padding(0, 1)
	title := fmt.Sprintf("📋 %s / %s", m.name, m.selectedKey)
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

	// Viewport
	b.WriteString(m.viewport.View())
	b.WriteString("\n\n")

	// Footer
	helpStyle := components.HelpStyle.Copy().
		Foreground(components.ColorMuted).
		Padding(1, 2)
	footerText := components.RenderKeyBinding("↑/↓", "scroll") + " • " +
		components.RenderKeyBinding("q/esc/b", "back to keys")
	b.WriteString(helpStyle.Render(footerText))

	return b.String()
}

// updateTable updates the table with configmap keys
func (m *ConfigMapDetailsModel) updateTable() {
	if m.configMap == nil {
		return
	}

	// Sort keys for consistent display
	keys := make([]string, 0, len(m.configMap.Data))
	for key := range m.configMap.Data {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Create table
	columns := []components.TableColumn{
		{Title: "Key", Width: 50, Align: "left"},
		{Title: "Lines", Width: 10, Align: "center"},
		{Title: "Size", Width: 15, Align: "right"},
		{Title: "Preview", Width: 40, Align: "left"},
	}

	rows := []components.TableRow{}
	for _, key := range keys {
		value := m.configMap.Data[key]
		lines := strings.Count(value, "\n") + 1
		size := formatSize(len(value))

		// Create preview
		preview := strings.ReplaceAll(value, "\n", " ")
		preview = strings.ReplaceAll(preview, "\t", " ")
		preview = strings.TrimSpace(preview)
		if len(preview) > 40 {
			preview = preview[:37] + "..."
		}

		rows = append(rows, components.TableRow{
			key,
			fmt.Sprintf("%d", lines),
			size,
			preview,
		})
	}

	title := fmt.Sprintf("ConfigMap: %s (%s)", m.name, m.namespace)
	m.table = components.NewTable(title, columns)
	m.table.SetRows(rows)
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
		Foreground(lipgloss.Color("252")).
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

func formatSize(bytes int) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d bytes", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
}

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
