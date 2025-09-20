package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/karthickk/k8s-manager/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SecretEditorModel represents the secret editor
type SecretEditorModel struct {
	secret       *corev1.Secret
	client       *k8s.Client
	keys         []string
	values       map[string]string
	selected     int
	editing      bool
	adding       bool
	keyInput     textinput.Model
	valueInput   textinput.Model
	currentKey   string
	message      string
	messageType  string
	quitting     bool
}

// NewSecretEditorModel creates a new secret editor
func NewSecretEditorModel(secret *corev1.Secret, client *k8s.Client) *SecretEditorModel {
	keyInput := textinput.New()
	keyInput.Placeholder = "Enter key name..."
	keyInput.CharLimit = 50

	valueInput := textinput.New()
	valueInput.Placeholder = "Enter value..."
	valueInput.CharLimit = 500

	// Decode secret data
	values := make(map[string]string)
	keys := make([]string, 0, len(secret.Data))

	for k, v := range secret.Data {
		keys = append(keys, k)
		values[k] = string(v) // Already decoded from base64
	}

	return &SecretEditorModel{
		secret:     secret,
		client:     client,
		keys:       keys,
		values:     values,
		keyInput:   keyInput,
		valueInput: valueInput,
		selected:   -1,
	}
}

func (m *SecretEditorModel) Init() tea.Cmd {
	return nil
}

func (m *SecretEditorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle input mode
		if m.editing || m.adding {
			switch msg.String() {
			case "esc":
				m.editing = false
				m.adding = false
				m.keyInput.Blur()
				m.valueInput.Blur()
				return m, nil

			case "tab":
				// Switch between key and value input when adding
				if m.adding {
					if m.keyInput.Focused() {
						m.keyInput.Blur()
						m.valueInput.Focus()
						return m, textinput.Blink
					} else {
						m.valueInput.Blur()
						m.keyInput.Focus()
						return m, textinput.Blink
					}
				}

			case "enter":
				if m.adding {
					// Add new key-value pair
					key := m.keyInput.Value()
					value := m.valueInput.Value()

					if key != "" && value != "" {
						m.values[key] = value
						if !contains(m.keys, key) {
							m.keys = append(m.keys, key)
						}
						m.message = fmt.Sprintf("Added %s", key)
						m.messageType = "success"

						// Clear inputs
						m.keyInput.SetValue("")
						m.valueInput.SetValue("")
						m.adding = false
						m.keyInput.Blur()
						m.valueInput.Blur()
					}
				} else if m.editing {
					// Update existing value
					value := m.valueInput.Value()
					if m.currentKey != "" && value != "" {
						m.values[m.currentKey] = value
						m.message = fmt.Sprintf("Updated %s", m.currentKey)
						m.messageType = "success"

						m.editing = false
						m.valueInput.Blur()
					}
				}
				return m, nil

			default:
				// Handle text input
				var cmd tea.Cmd
				if m.adding && m.keyInput.Focused() {
					m.keyInput, cmd = m.keyInput.Update(msg)
					return m, cmd
				} else if m.valueInput.Focused() {
					m.valueInput, cmd = m.valueInput.Update(msg)
					return m, cmd
				}
			}
		}

		// Handle navigation mode
		keyStr := msg.String()

		// Number keys for quick selection
		if len(keyStr) == 1 && keyStr[0] >= '1' && keyStr[0] <= '8' {
			num := int(keyStr[0] - '0')
			if num <= len(m.keys) {
				m.selected = num - 1
				// Start editing
				m.currentKey = m.keys[m.selected]
				m.valueInput.SetValue(m.values[m.currentKey])
				m.valueInput.Focus()
				m.editing = true
				return m, textinput.Blink
			}
		}

		switch keyStr {
		case "a": // Add new key-value
			m.adding = true
			m.keyInput.SetValue("")
			m.valueInput.SetValue("")
			m.keyInput.Focus()
			return m, textinput.Blink

		case "e": // Edit selected
			if m.selected >= 0 && m.selected < len(m.keys) {
				m.currentKey = m.keys[m.selected]
				m.valueInput.SetValue(m.values[m.currentKey])
				m.valueInput.Focus()
				m.editing = true
				return m, textinput.Blink
			}

		case "d": // Delete selected
			if m.selected >= 0 && m.selected < len(m.keys) {
				key := m.keys[m.selected]
				delete(m.values, key)
				m.keys = append(m.keys[:m.selected], m.keys[m.selected+1:]...)
				m.message = fmt.Sprintf("Deleted %s", key)
				m.messageType = "info"
				if m.selected >= len(m.keys) && m.selected > 0 {
					m.selected--
				}
			}

		case "s": // Save changes
			return m, m.saveSecret()

		case "up", "k":
			if m.selected > 0 {
				m.selected--
			} else if m.selected == -1 && len(m.keys) > 0 {
				m.selected = len(m.keys) - 1
			}

		case "down", "j":
			if m.selected < len(m.keys)-1 {
				m.selected++
			} else if m.selected == -1 && len(m.keys) > 0 {
				m.selected = 0
			}

		case "q", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m *SecretEditorModel) saveSecret() tea.Cmd {
	return func() tea.Msg {
		// Encode values to base64
		newData := make(map[string][]byte)
		for k, v := range m.values {
			newData[k] = []byte(v)
		}

		m.secret.Data = newData

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_, err := m.client.Clientset.CoreV1().Secrets(m.secret.Namespace).Update(
			ctx, m.secret, metav1.UpdateOptions{})

		if err != nil {
			return secretUpdateMsg{err: err}
		}

		return secretUpdateMsg{success: true}
	}
}

type secretUpdateMsg struct {
	success bool
	err     error
}

func (m *SecretEditorModel) View() string {
	var s strings.Builder

	s.WriteString("\033[H\033[2J") // Clear screen

	// Title
	s.WriteString(devToolsTitleStyle.Render(fmt.Sprintf("🔐 Edit Secret: %s", m.secret.Name)))
	s.WriteString("\n\n")

	// Show input fields when adding
	if m.adding {
		s.WriteString(devToolsNumberStyle.Render("Add New Key-Value Pair:"))
		s.WriteString("\n\n")
		s.WriteString("Key:   ")
		s.WriteString(m.keyInput.View())
		s.WriteString("\n")
		s.WriteString("Value: ")
		s.WriteString(m.valueInput.View())
		s.WriteString("\n\n")
		s.WriteString(devToolsHelpStyle.Render("Tab to switch • Enter to add • Esc to cancel"))
		return devToolsContainerStyle.Render(s.String())
	}

	// Show value editor when editing
	if m.editing {
		s.WriteString(devToolsNumberStyle.Render(fmt.Sprintf("Edit Value for '%s':", m.currentKey)))
		s.WriteString("\n\n")
		s.WriteString(m.valueInput.View())
		s.WriteString("\n\n")
		s.WriteString(devToolsHelpStyle.Render("Enter to save • Esc to cancel"))
		return devToolsContainerStyle.Render(s.String())
	}

	// List current key-value pairs
	if len(m.keys) == 0 {
		s.WriteString(devToolsDescriptionStyle.Render("No data in this secret"))
		s.WriteString("\n\n")
	} else {
		s.WriteString(devToolsNumberStyle.Render("Current Data:"))
		s.WriteString("\n\n")

		for i, key := range m.keys {
			// Number for quick selection
			if i < 8 {
				s.WriteString(devToolsNumberStyle.Render(fmt.Sprintf("%d.", i+1)))
			} else {
				s.WriteString("  ")
			}

			// Key name
			keyStr := key
			if i == m.selected {
				keyStr = devToolsSelectedStyle.Render("▸ " + keyStr)
			} else {
				keyStr = "  " + devToolsItemStyle.Render(keyStr)
			}
			s.WriteString(keyStr)
			s.WriteString("\n")

			// Value (masked)
			value := m.values[key]
			displayValue := value
			if len(displayValue) > 40 {
				displayValue = displayValue[:15] + "..." + displayValue[len(displayValue)-15:]
			}
			s.WriteString(devToolsDescriptionStyle.Render("   " + displayValue))
			s.WriteString("\n")
		}
	}

	// Message
	if m.message != "" {
		s.WriteString("\n")
		switch m.messageType {
		case "success":
			s.WriteString(devToolsSuccessStyle.Render("✓ " + m.message))
		case "error":
			s.WriteString(devToolsErrorStyle.Render("✗ " + m.message))
		default:
			s.WriteString(devToolsInfoStyle.Render("• " + m.message))
		}
	}

	// Help
	s.WriteString("\n\n")
	s.WriteString(devToolsHelpStyle.Render(
		"↑/k up • ↓/j down • 1-8 quick edit • a add • e edit • d delete • s save • q quit",
	))

	return devToolsContainerStyle.Render(s.String())
}

// CreateSecretModel represents the secret creation form
type CreateSecretModel struct {
	namespace    string
	client       *k8s.Client
	nameInput    textinput.Model
	typeSelector int
	data         map[string]string
	step         int // 0: name/type, 1: add data, 2: review
	message      string
}

// NewCreateSecretModel creates a new secret creation model
func NewCreateSecretModel(namespace string, client *k8s.Client) *CreateSecretModel {
	nameInput := textinput.New()
	nameInput.Placeholder = "my-secret"
	nameInput.CharLimit = 63
	nameInput.Focus()

	return &CreateSecretModel{
		namespace:    namespace,
		client:       client,
		nameInput:    nameInput,
		data:         make(map[string]string),
		typeSelector: 0,
	}
}

func (m *CreateSecretModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *CreateSecretModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.step {
		case 0: // Name and type selection
			switch msg.String() {
			case "enter":
				if m.nameInput.Value() != "" {
					m.step = 1
					m.nameInput.Blur()
				}
				return m, nil

			case "up", "k":
				if !m.nameInput.Focused() && m.typeSelector > 0 {
					m.typeSelector--
				}

			case "down", "j":
				if !m.nameInput.Focused() && m.typeSelector < 2 {
					m.typeSelector++
				}

			case "tab":
				if m.nameInput.Focused() {
					m.nameInput.Blur()
				} else {
					m.nameInput.Focus()
					return m, textinput.Blink
				}

			case "esc", "q":
				return m, tea.Quit

			default:
				if m.nameInput.Focused() {
					var cmd tea.Cmd
					m.nameInput, cmd = m.nameInput.Update(msg)
					return m, cmd
				}
			}

		case 1: // Add data
			switch msg.String() {
			case "c": // Continue to review
				if len(m.data) > 0 {
					m.step = 2
				} else {
					m.message = "Add at least one key-value pair"
				}

			case "esc", "q":
				return m, tea.Quit
			}

		case 2: // Review and create
			switch msg.String() {
			case "c": // Create secret
				return m, m.createSecret()

			case "b": // Back to edit
				m.step = 1

			case "esc", "q":
				return m, tea.Quit
			}
		}
	}

	return m, nil
}

func (m *CreateSecretModel) createSecret() tea.Cmd {
	return func() tea.Msg {
		// Prepare secret data
		secretData := make(map[string][]byte)
		for k, v := range m.data {
			secretData[k] = []byte(v)
		}

		// Determine secret type
		secretType := corev1.SecretTypeOpaque
		switch m.typeSelector {
		case 1:
			secretType = corev1.SecretTypeDockerConfigJson
		case 2:
			secretType = corev1.SecretTypeTLS
		}

		// Create secret object
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      m.nameInput.Value(),
				Namespace: m.namespace,
			},
			Type: secretType,
			Data: secretData,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_, err := m.client.Clientset.CoreV1().Secrets(m.namespace).Create(
			ctx, secret, metav1.CreateOptions{})

		if err != nil {
			return secretCreateMsg{err: err}
		}

		return secretCreateMsg{success: true}
	}
}

type secretCreateMsg struct {
	success bool
	err     error
}

func (m *CreateSecretModel) View() string {
	var s strings.Builder

	s.WriteString("\033[H\033[2J") // Clear screen

	// Title
	s.WriteString(devToolsTitleStyle.Render("🔐 Create New Secret"))
	s.WriteString("\n\n")

	switch m.step {
	case 0: // Name and type
		s.WriteString(devToolsNumberStyle.Render("Step 1: Basic Information"))
		s.WriteString("\n\n")

		s.WriteString("Secret Name:\n")
		s.WriteString(m.nameInput.View())
		s.WriteString("\n\n")

		s.WriteString("Secret Type:\n")
		types := []string{"Opaque (Generic)", "Docker Config", "TLS Certificate"}
		for i, t := range types {
			if i == m.typeSelector {
				s.WriteString(devToolsSelectedStyle.Render("▸ " + t))
			} else {
				s.WriteString("  " + devToolsItemStyle.Render(t))
			}
			s.WriteString("\n")
		}

		s.WriteString("\n")
		s.WriteString(devToolsHelpStyle.Render("Tab to switch • Enter to continue • Esc to cancel"))

	case 1: // Add data
		s.WriteString(devToolsNumberStyle.Render("Step 2: Add Data"))
		s.WriteString("\n\n")

		if len(m.data) == 0 {
			s.WriteString(devToolsDescriptionStyle.Render("No data added yet"))
		} else {
			for k, v := range m.data {
				s.WriteString(fmt.Sprintf("%s: %s\n", devToolsInfoStyle.Render(k),
					devToolsDescriptionStyle.Render(truncateValue(v, 40))))
			}
		}

		s.WriteString("\n")
		s.WriteString(devToolsInfoStyle.Render("Use the secret editor to add key-value pairs"))
		s.WriteString("\n\n")
		s.WriteString(devToolsHelpStyle.Render("c continue • esc cancel"))

	case 2: // Review
		s.WriteString(devToolsNumberStyle.Render("Step 3: Review and Create"))
		s.WriteString("\n\n")

		s.WriteString(fmt.Sprintf("Name: %s\n", devToolsInfoStyle.Render(m.nameInput.Value())))
		s.WriteString(fmt.Sprintf("Namespace: %s\n", devToolsInfoStyle.Render(m.namespace)))
		s.WriteString(fmt.Sprintf("Type: %s\n", devToolsInfoStyle.Render(getSecretTypeName(m.typeSelector))))
		s.WriteString(fmt.Sprintf("Data Keys: %d\n", len(m.data)))

		s.WriteString("\n")
		s.WriteString(devToolsHelpStyle.Render("c create • b back • esc cancel"))
	}

	if m.message != "" {
		s.WriteString("\n\n")
		s.WriteString(devToolsWarningStyle.Render(m.message))
	}

	return devToolsContainerStyle.Render(s.String())
}

func getSecretTypeName(selector int) string {
	switch selector {
	case 1:
		return "Docker Config"
	case 2:
		return "TLS Certificate"
	default:
		return "Opaque"
	}
}