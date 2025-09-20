package views

import (
	"context"
	"encoding/base64"
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

// SecretDetailsModel shows secret details
type SecretDetailsModel struct {
	namespace    string
	name         string
	secret       *corev1.Secret
	listView     *components.ListView
	viewport     viewport.Model
	showDecoded  bool
	viewMode     string // "list" or "detail"
	selectedKey  string
	ready        bool
	quitting     bool
	loading      bool
	errorMsg     string
}

// secretLoadedMsg is sent when secret is loaded
type secretLoadedMsg struct {
	secret *corev1.Secret
	err    error
}

// NewSecretDetailsModel creates a new secret details view
func NewSecretDetailsModel(namespace, name string) *SecretDetailsModel {
	return &SecretDetailsModel{
		namespace:   namespace,
		name:        name,
		loading:     true,
		showDecoded: true,  // Show decoded by default
		viewMode:    "list",
	}
}

// Init initializes the model
func (m *SecretDetailsModel) Init() tea.Cmd {
	return m.loadSecret
}

// Update handles messages
func (m *SecretDetailsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			// Go back to Secrets list
			return m, Navigate(ViewSecrets, nil)
		case "b":
			// Quick back navigation
			return m, Navigate(ViewSecrets, nil)
		case "e":
			if m.viewMode == "list" && m.listView != nil {
				selected := m.listView.GetSelected()
				if selected != nil {
					// TODO: Edit Secret key
					return m, nil
				}
			}
		case "a":
			if m.viewMode == "list" {
				// Navigate to add key view
				return m, Navigate(ViewAddSecretKey, map[string]string{
					"namespace": m.namespace,
					"name":      m.name,
				})
			}
		case "x":
			if m.viewMode == "list" && m.listView != nil {
				selected := m.listView.GetSelected()
				if selected != nil {
					// TODO: Delete Secret key
					return m, nil
				}
			}
		case "d":
			// Toggle decoded/encoded view
			m.showDecoded = !m.showDecoded
			if m.viewMode == "detail" {
				m.updateViewport()
			}
			return m, nil
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

	case secretLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.errorMsg = msg.err.Error()
		} else {
			m.secret = msg.secret
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
func (m *SecretDetailsModel) View() string {
	if m.quitting {
		return ""
	}

	if m.loading {
		return components.NewLoadingScreen("Loading Secret Details").View()
	}

	if m.errorMsg != "" {
		return components.BoxStyle.Render(
			components.RenderTitle("Secret Details", "") + "\n\n" +
				components.RenderMessage("error", m.errorMsg) + "\n\n" +
				components.HelpStyle.Render("Press 'q' to quit"),
		)
	}

	if m.viewMode == "detail" {
		return m.renderDetailView()
	}

	// List view
	if m.listView == nil {
		return "No secret data available"
	}

	return m.listView.View()
}

// renderDetailView renders the detail view for a specific key
func (m *SecretDetailsModel) renderDetailView() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	// Title
	title := fmt.Sprintf("🔐 Secret: %s / Key: %s", m.name, m.selectedKey)
	if m.showDecoded {
		title += " (Decoded)"
	} else {
		title += " (Base64 Encoded)"
	}
	header := components.TitleStyle.Render(title)

	// Footer
	footerText := "d: toggle decode/encode • q/esc: back to keys • ↑/↓: scroll"
	footer := components.HelpStyle.Render(footerText)

	return fmt.Sprintf("%s\n\n%s\n\n%s", header, m.viewport.View(), footer)
}

// updateListView updates the list with secret keys
func (m *SecretDetailsModel) updateListView() {
	if m.secret == nil {
		return
	}

	listItems := []components.ListItem{}

	// Add secret keys
	for key := range m.secret.Data {
		valueSize := len(m.secret.Data[key])
		description := fmt.Sprintf("Size: %d bytes", valueSize)
		
		// Add preview
		if m.showDecoded {
			// Show decoded value
			decoded := string(m.secret.Data[key])
			preview := decoded
			if len(preview) > 40 {
				preview = preview[:37] + "..."
			}
			description = fmt.Sprintf("%s", preview)
		} else {
			// Show size when encoded
			description = fmt.Sprintf("Size: %d bytes (encoded)", valueSize)
		}

		listItems = append(listItems, components.ListItem{
			ID:          key,
			Title:       key,
			Description: description,
			Icon:        "🔑",
		})
	}

	title := fmt.Sprintf("🔐 Secret: %s", m.name)
	if m.secret.Type != "" {
		title += fmt.Sprintf(" (Type: %s)", m.secret.Type)
	}

	m.listView = components.NewListView(title, listItems)
	m.listView.SetHelpText("enter: view key • a: add key • e: edit • x: delete • d: toggle decode • esc/b: back • ctrl+c: quit")
}

// updateViewport updates the viewport with the selected key's content
func (m *SecretDetailsModel) updateViewport() {
	if m.secret == nil || m.selectedKey == "" {
		return
	}

	data, exists := m.secret.Data[m.selectedKey]
	if !exists {
		m.viewport.SetContent("Key not found in secret")
		return
	}

	var content string
	if m.showDecoded {
		// Show decoded content
		decoded := string(data)
		
		// Format based on content type
		if strings.Contains(m.selectedKey, "json") || isJSON(decoded) {
			// Pretty print JSON if possible
			content = formatJSON(decoded)
		} else if strings.Contains(m.selectedKey, "yaml") || strings.Contains(m.selectedKey, "yml") {
			// Keep YAML formatting
			content = decoded
		} else {
			// Regular text
			content = decoded
		}
	} else {
		// Show base64 encoded content with line breaks
		encoded := base64.StdEncoding.EncodeToString(data)
		// Break into 76-character lines for readability
		var lines []string
		for i := 0; i < len(encoded); i += 76 {
			end := i + 76
			if end > len(encoded) {
				end = len(encoded)
			}
			lines = append(lines, encoded[i:end])
		}
		content = strings.Join(lines, "\n")
	}

	// Add some styling
	styledContent := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Render(content)

	m.viewport.SetContent(styledContent)
}

// loadSecret loads the secret details
func (m *SecretDetailsModel) loadSecret() tea.Msg {
	client, err := services.GetK8sClient()
	if err != nil {
		return secretLoadedMsg{err: err}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	secret, err := client.Clientset.CoreV1().Secrets(m.namespace).Get(ctx, m.name, metav1.GetOptions{})
	if err != nil {
		return secretLoadedMsg{err: err}
	}

	return secretLoadedMsg{secret: secret}
}

// Helper functions

func isJSON(str string) bool {
	str = strings.TrimSpace(str)
	return (strings.HasPrefix(str, "{") && strings.HasSuffix(str, "}")) ||
		(strings.HasPrefix(str, "[") && strings.HasSuffix(str, "]"))
}

func formatJSON(jsonStr string) string {
	// Simple JSON formatter - in production, use encoding/json
	// For now, just add some basic indentation
	result := jsonStr
	result = strings.ReplaceAll(result, ",", ",\n  ")
	result = strings.ReplaceAll(result, "{", "{\n  ")
	result = strings.ReplaceAll(result, "}", "\n}")
	result = strings.ReplaceAll(result, "[", "[\n  ")
	result = strings.ReplaceAll(result, "]", "\n]")
	return result
}

// ShowSecretDetails shows the secret details view
func ShowSecretDetails(namespace, name string) tea.Cmd {
	return func() tea.Msg {
		model := NewSecretDetailsModel(namespace, name)
		p := tea.NewProgram(model, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return components.ErrorMsg{Error: err}
		}
		return nil
	}
}