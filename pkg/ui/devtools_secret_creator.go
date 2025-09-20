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

// SecretCreatorModel manages the creation of new secrets
type SecretCreatorModel struct {
	client         *k8s.Client
	step           int // 0: name, 1: namespace, 2: type, 3: add data, 4: review
	name           string
	namespace      string
	secretType     corev1.SecretType
	data           map[string]string
	nameInput      textinput.Model
	namespaceInput textinput.Model
	keyInput       textinput.Model
	valueInput     textinput.Model
	currentKey     string
	currentValue   string
	message        string
	messageType    string
	err            error
}

// NewSecretCreatorModel creates a new secret creator model
func NewSecretCreatorModel(namespace string) *SecretCreatorModel {
	nameInput := textinput.New()
	nameInput.Placeholder = "my-secret"
	nameInput.CharLimit = 63
	nameInput.Focus()

	namespaceInput := textinput.New()
	namespaceInput.Placeholder = "default"
	namespaceInput.CharLimit = 63
	if namespace != "" {
		namespaceInput.SetValue(namespace)
	}

	keyInput := textinput.New()
	keyInput.Placeholder = "key"
	keyInput.CharLimit = 256

	valueInput := textinput.New()
	valueInput.Placeholder = "value"
	valueInput.CharLimit = 1024

	return &SecretCreatorModel{
		step:           0,
		namespace:      namespace,
		secretType:     corev1.SecretTypeOpaque,
		data:           make(map[string]string),
		nameInput:      nameInput,
		namespaceInput: namespaceInput,
		keyInput:       keyInput,
		valueInput:     valueInput,
	}
}

func (m *SecretCreatorModel) Init() tea.Cmd {
	return m.loadClient
}

func (m *SecretCreatorModel) loadClient() tea.Msg {
	client, err := k8s.NewClient()
	if err != nil {
		return secretCreatorErrorMsg{err}
	}
	return secretCreatorClientMsg{client}
}

type secretCreatorErrorMsg struct{ err error }
type secretCreatorClientMsg struct{ client *k8s.Client }
type secretCreatedMsg struct{ success bool }

func (m *SecretCreatorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case secretCreatorClientMsg:
		m.client = msg.client
		if m.namespace == "" {
			m.namespace = m.client.GetNamespace()
			m.namespaceInput.SetValue(m.namespace)
		}
		return m, nil

	case secretCreatorErrorMsg:
		m.err = msg.err
		return m, nil

	case secretCreatedMsg:
		if msg.success {
			m.message = "Secret created successfully!"
			m.messageType = "success"
		}
		return m, tea.Quit

	case tea.KeyMsg:
		keyStr := msg.String()

		switch m.step {
		case 0: // Enter name
			switch keyStr {
			case "enter":
				m.name = m.nameInput.Value()
				if m.name == "" {
					m.message = "Name cannot be empty"
					m.messageType = "error"
				} else {
					m.step = 1
					m.namespaceInput.Focus()
					m.nameInput.Blur()
				}
				return m, nil

			case "esc", "ctrl+c":
				return m, tea.Quit

			default:
				var cmd tea.Cmd
				m.nameInput, cmd = m.nameInput.Update(msg)
				return m, cmd
			}

		case 1: // Enter namespace
			switch keyStr {
			case "enter":
				m.namespace = m.namespaceInput.Value()
				if m.namespace == "" {
					m.namespace = "default"
				}
				m.step = 2
				m.namespaceInput.Blur()
				return m, nil

			case "esc":
				m.step = 0
				m.nameInput.Focus()
				m.namespaceInput.Blur()
				return m, nil

			case "ctrl+c":
				return m, tea.Quit

			default:
				var cmd tea.Cmd
				m.namespaceInput, cmd = m.namespaceInput.Update(msg)
				return m, cmd
			}

		case 2: // Select type
			switch keyStr {
			case "1": // Opaque
				m.secretType = corev1.SecretTypeOpaque
				m.step = 3
				m.keyInput.Focus()
				return m, nil

			case "2": // Docker config
				m.secretType = corev1.SecretTypeDockerConfigJson
				m.step = 3
				m.keyInput.Focus()
				return m, nil

			case "3": // TLS
				m.secretType = corev1.SecretTypeTLS
				m.step = 3
				m.keyInput.Focus()
				return m, nil

			case "esc":
				m.step = 1
				m.namespaceInput.Focus()
				return m, nil

			case "ctrl+c":
				return m, tea.Quit
			}

		case 3: // Add data
			if m.keyInput.Focused() {
				switch keyStr {
				case "enter":
					m.currentKey = m.keyInput.Value()
					if m.currentKey != "" {
						m.keyInput.Blur()
						m.valueInput.Focus()
					}
					return m, nil

				case "esc":
					if len(m.data) > 0 {
						m.step = 4
						m.keyInput.Blur()
					} else {
						m.step = 2
						m.keyInput.Blur()
					}
					return m, nil

				case "ctrl+c":
					return m, tea.Quit

				default:
					var cmd tea.Cmd
					m.keyInput, cmd = m.keyInput.Update(msg)
					return m, cmd
				}
			} else if m.valueInput.Focused() {
				switch keyStr {
				case "enter":
					m.currentValue = m.valueInput.Value()
					if m.currentKey != "" {
						m.data[m.currentKey] = m.currentValue
						m.keyInput.SetValue("")
						m.valueInput.SetValue("")
						m.currentKey = ""
						m.currentValue = ""
						m.valueInput.Blur()
						m.keyInput.Focus()
						m.message = fmt.Sprintf("Added key: %s", m.currentKey)
						m.messageType = "success"
					}
					return m, nil

				case "esc":
					m.valueInput.Blur()
					m.keyInput.Focus()
					return m, nil

				case "ctrl+c":
					return m, tea.Quit

				default:
					var cmd tea.Cmd
					m.valueInput, cmd = m.valueInput.Update(msg)
					return m, cmd
				}
			} else {
				switch keyStr {
				case "a": // Add more
					m.keyInput.Focus()
					return m, nil

				case "c": // Continue to review
					if len(m.data) > 0 {
						m.step = 4
					} else {
						m.message = "Add at least one key-value pair"
						m.messageType = "warning"
					}
					return m, nil

				case "esc":
					m.step = 2
					return m, nil

				case "ctrl+c":
					return m, tea.Quit
				}
			}

		case 4: // Review and create
			switch keyStr {
			case "c": // Create
				return m, m.createSecret()

			case "b": // Back
				m.step = 3
				m.keyInput.Focus()
				return m, nil

			case "esc", "ctrl+c":
				return m, tea.Quit
			}
		}
	}

	return m, nil
}

func (m *SecretCreatorModel) View() string {
	var s strings.Builder

	s.WriteString("\033[H\033[2J") // Clear screen

	// Title
	s.WriteString(devToolsTitleStyle.Render("🔒 Create New Secret"))
	s.WriteString("\n\n")

	// Error
	if m.err != nil {
		s.WriteString(devToolsErrorStyle.Render("Error: " + m.err.Error()))
		s.WriteString("\n\n")
		s.WriteString(devToolsHelpStyle.Render("Press ctrl+c to exit"))
		return devToolsContainerStyle.Render(s.String())
	}

	switch m.step {
	case 0: // Name input
		s.WriteString(devToolsNumberStyle.Render("Step 1: Enter Secret Name"))
		s.WriteString("\n\n")
		s.WriteString("Name: ")
		s.WriteString(m.nameInput.View())
		s.WriteString("\n\n")
		s.WriteString(devToolsHelpStyle.Render("enter to continue • esc to cancel"))

	case 1: // Namespace input
		s.WriteString(devToolsNumberStyle.Render("Step 2: Enter Namespace"))
		s.WriteString("\n\n")
		s.WriteString("Namespace: ")
		s.WriteString(m.namespaceInput.View())
		s.WriteString("\n\n")
		s.WriteString(devToolsHelpStyle.Render("enter to continue • esc to go back"))

	case 2: // Type selection
		s.WriteString(devToolsNumberStyle.Render("Step 3: Select Secret Type"))
		s.WriteString("\n\n")

		s.WriteString(devToolsNumberStyle.Render("1.") + "  " + devToolsItemStyle.Render("Opaque"))
		s.WriteString("\n")
		s.WriteString(devToolsDescriptionStyle.Render("   Generic secret for storing arbitrary data"))
		s.WriteString("\n")

		s.WriteString(devToolsNumberStyle.Render("2.") + "  " + devToolsItemStyle.Render("Docker Config"))
		s.WriteString("\n")
		s.WriteString(devToolsDescriptionStyle.Render("   Docker registry authentication"))
		s.WriteString("\n")

		s.WriteString(devToolsNumberStyle.Render("3.") + "  " + devToolsItemStyle.Render("TLS"))
		s.WriteString("\n")
		s.WriteString(devToolsDescriptionStyle.Render("   TLS certificate and key"))
		s.WriteString("\n\n")

		s.WriteString(devToolsHelpStyle.Render("1-3 select type • esc to go back"))

	case 3: // Add data
		s.WriteString(devToolsNumberStyle.Render("Step 4: Add Data"))
		s.WriteString("\n\n")

		// Show existing data
		if len(m.data) > 0 {
			s.WriteString(devToolsInfoStyle.Render("Current data:"))
			s.WriteString("\n")
			for k, v := range m.data {
				displayValue := v
				if len(displayValue) > 30 {
					displayValue = displayValue[:27] + "..."
				}
				s.WriteString(fmt.Sprintf("  %s = %s\n",
					devToolsNumberStyle.Render(k),
					devToolsDescriptionStyle.Render(displayValue)))
			}
			s.WriteString("\n")
		}

		if m.keyInput.Focused() {
			s.WriteString("Key: ")
			s.WriteString(m.keyInput.View())
			s.WriteString("\n\n")
			s.WriteString(devToolsHelpStyle.Render("enter to add value • esc to finish"))
		} else if m.valueInput.Focused() {
			s.WriteString("Key: " + devToolsInfoStyle.Render(m.currentKey))
			s.WriteString("\nValue: ")
			s.WriteString(m.valueInput.View())
			s.WriteString("\n\n")
			s.WriteString(devToolsHelpStyle.Render("enter to save • esc to cancel"))
		} else {
			s.WriteString("\n")
			s.WriteString(devToolsHelpStyle.Render("a add more • c continue • esc to go back"))
		}

	case 4: // Review
		s.WriteString(devToolsNumberStyle.Render("Review and Create"))
		s.WriteString("\n\n")

		s.WriteString(fmt.Sprintf("Name: %s\n", devToolsInfoStyle.Render(m.name)))
		s.WriteString(fmt.Sprintf("Namespace: %s\n", devToolsInfoStyle.Render(m.namespace)))
		s.WriteString(fmt.Sprintf("Type: %s\n", devToolsInfoStyle.Render(string(m.secretType))))
		s.WriteString(fmt.Sprintf("Data Keys: %s\n", devToolsInfoStyle.Render(fmt.Sprintf("%d", len(m.data)))))

		s.WriteString("\nData:\n")
		for k, v := range m.data {
			displayValue := v
			if len(displayValue) > 50 {
				displayValue = displayValue[:47] + "..."
			}
			s.WriteString(fmt.Sprintf("  %s = %s\n",
				devToolsNumberStyle.Render(k),
				devToolsDescriptionStyle.Render(displayValue)))
		}

		s.WriteString("\n")
		s.WriteString(devToolsHelpStyle.Render("c create secret • b back • esc cancel"))
	}

	// Message
	if m.message != "" {
		s.WriteString("\n\n")
		switch m.messageType {
		case "success":
			s.WriteString(devToolsSuccessStyle.Render("✓ " + m.message))
		case "error":
			s.WriteString(devToolsErrorStyle.Render("✗ " + m.message))
		case "warning":
			s.WriteString(devToolsWarningStyle.Render("⚠ " + m.message))
		default:
			s.WriteString(devToolsInfoStyle.Render("• " + m.message))
		}
	}

	return devToolsContainerStyle.Render(s.String())
}

func (m *SecretCreatorModel) createSecret() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Convert string data to []byte
		byteData := make(map[string][]byte)
		for k, v := range m.data {
			byteData[k] = []byte(v)
		}

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      m.name,
				Namespace: m.namespace,
			},
			Type: m.secretType,
			Data: byteData,
		}

		_, err := m.client.Clientset.CoreV1().Secrets(m.namespace).Create(ctx, secret, metav1.CreateOptions{})
		if err != nil {
			return secretCreatorErrorMsg{err}
		}

		return secretCreatedMsg{success: true}
	}
}

// ShowSecretCreator shows the secret creation interface
func ShowSecretCreator(namespace string) error {
	model := NewSecretCreatorModel(namespace)
	p := tea.NewProgram(model, tea.WithAltScreen())

	result, err := p.Run()
	if err != nil {
		return err
	}

	if m, ok := result.(*SecretCreatorModel); ok && m.message == "Secret created successfully!" {
		fmt.Printf("\n✅ Secret '%s' created in namespace '%s'\n", m.name, m.namespace)
	}

	return nil
}