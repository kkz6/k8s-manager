package ui

import (
	"context"
	"encoding/base64"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/karthickk/k8s-manager/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DevToolsSecretsModel represents the secrets view in DevTools style
type DevToolsSecretsModel struct {
	secrets       []SecretInfo
	filtered      []SecretInfo
	selected      int
	filterInput   textinput.Model
	filtering     bool
	loading       bool
	loadingAction bool // Track if we're loading an action
	spinner       AnimatedSpinner // Animated spinner
	client        *k8s.Client
	namespace     string
	allNamespaces bool
	message       string
	err           error
	secretSelected bool
}

// SecretInfo holds secret information
type SecretInfo struct {
	Name      string
	Namespace string
	Type      string
	DataCount int
	Age       string
	Secret    *corev1.Secret
}

// NewDevToolsSecretsModel creates a new secrets model
func NewDevToolsSecretsModel(namespace string, allNamespaces bool) *DevToolsSecretsModel {
	ti := textinput.New()
	ti.Placeholder = "Type to filter..."
	ti.CharLimit = 50

	return &DevToolsSecretsModel{
		filterInput:   ti,
		loading:       true,
		spinner:       NewAnimatedSpinner("spinner", "Loading secrets"),
		namespace:     namespace,
		allNamespaces: allNamespaces,
		selected:      -1,
	}
}

func (m *DevToolsSecretsModel) Init() tea.Cmd {
	return tea.Batch(
		m.loadSecrets,
		m.spinner.Init(),
	)
}

func (m *DevToolsSecretsModel) loadSecrets() tea.Msg {
	client, err := k8s.NewClient()
	if err != nil {
		return secretErrorMsg{err}
	}

	namespace := m.namespace
	if namespace == "" && !m.allNamespaces {
		namespace = client.GetNamespace()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var secrets *corev1.SecretList
	if m.allNamespaces {
		secrets, err = client.Clientset.CoreV1().Secrets("").List(ctx, metav1.ListOptions{})
	} else {
		secrets, err = client.Clientset.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
	}

	if err != nil {
		return secretErrorMsg{err}
	}

	secretInfos := make([]SecretInfo, 0, len(secrets.Items))
	for _, secret := range secrets.Items {
		info := SecretInfo{
			Name:      secret.Name,
			Namespace: secret.Namespace,
			Type:      string(secret.Type),
			DataCount: len(secret.Data),
			Age:       formatAge(secret.CreationTimestamp.Time),
			Secret:    &secret,
		}
		secretInfos = append(secretInfos, info)
	}

	return secretsLoadedMsg{secrets: secretInfos, client: client}
}

type secretErrorMsg struct{ err error }
type secretsLoadedMsg struct {
	secrets []SecretInfo
	client  *k8s.Client
}

func (m *DevToolsSecretsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Update spinner animation
	if m.loading || m.loadingAction {
		spinner, cmd := m.spinner.Update(msg)
		m.spinner = spinner
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m, nil

	case secretsLoadedMsg:
		m.loading = false
		m.secrets = msg.secrets
		m.filtered = msg.secrets
		m.client = msg.client
		return m, nil

	case secretErrorMsg:
		m.loading = false
		m.err = msg.err
		return m, nil

	case tea.KeyMsg:
		if m.filtering {
			switch msg.String() {
			case "esc":
				m.filtering = false
				m.filterInput.Blur()
				m.filterInput.SetValue("")
				m.filtered = m.secrets
				return m, nil

			case "enter":
				m.filtering = false
				m.filterInput.Blur()
				m.applyFilter()
				return m, nil

			default:
				var cmd tea.Cmd
				m.filterInput, cmd = m.filterInput.Update(msg)
				m.applyFilter()
				return m, cmd
			}
		}

		keyStr := msg.String()

		// Number keys for quick selection
		if len(keyStr) == 1 && keyStr[0] >= '1' && keyStr[0] <= '8' {
			num := int(keyStr[0] - '0')
			if num <= len(m.filtered) {
				m.selected = num - 1
				m.secretSelected = true
				m.loadingAction = true
				m.spinner = NewAnimatedSpinner("spinner", fmt.Sprintf("Loading secret %s", m.filtered[m.selected].Name))
				return m, tea.Batch(
					m.spinner.Init(),
					tea.Tick(time.Millisecond*300, func(t time.Time) tea.Msg {
						return tea.Quit()
					}),
				)
			}
		}

		// Special keys
		switch keyStr {
		case "9": // Create new secret
			m.selected = -2 // Special value for create
			m.secretSelected = true
			m.loadingAction = true
			m.spinner = NewAnimatedSpinner("spinner", "Loading secret creator")
			return m, tea.Batch(
				m.spinner.Init(),
				tea.Tick(time.Millisecond*300, func(t time.Time) tea.Msg {
					return tea.Quit()
				}),
			)

		case "0", "b": // Back to main menu
			return m, tea.Quit

		case "q", "ctrl+c", "esc":
			return m, tea.Quit

		case "/":
			m.filtering = true
			m.filterInput.Focus()
			return m, textinput.Blink

		case "up", "k":
			if m.selected > 0 {
				m.selected--
			} else if m.selected == -1 && len(m.filtered) > 0 {
				m.selected = len(m.filtered) - 1
			}

		case "down", "j":
			if m.selected < len(m.filtered)-1 {
				m.selected++
			} else if m.selected == -1 && len(m.filtered) > 0 {
				m.selected = 0
			}

		case "enter", " ":
			if m.selected >= 0 && m.selected < len(m.filtered) {
				m.secretSelected = true
				m.loadingAction = true
				m.spinner = NewAnimatedSpinner("spinner", fmt.Sprintf("Loading secret %s", m.filtered[m.selected].Name))
				return m, tea.Batch(
					m.spinner.Init(),
					tea.Tick(time.Millisecond*300, func(t time.Time) tea.Msg {
						return tea.Quit()
					}),
				)
			}

		case "r": // Refresh
			m.loading = true
			m.spinner = NewAnimatedSpinner("spinner", "Refreshing secrets")
			return m, tea.Batch(
				m.loadSecrets,
				m.spinner.Init(),
			)

		case "d": // Delete
			if m.selected >= 0 && m.selected < len(m.filtered) {
				return m, m.deleteSecret()
			}
		}
	}

	if len(cmds) > 0 {
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

func (m *DevToolsSecretsModel) View() string {
	var s strings.Builder

	if m.err != nil {
		s.WriteString(devToolsContainerStyle.Render(
			devToolsErrorStyle.Render("Error: " + m.err.Error()),
		))
		return s.String()
	}

	// Title
	title := "🔒 Kubernetes Secrets"
	if m.namespace != "" && !m.allNamespaces {
		title += fmt.Sprintf(" - %s namespace", m.namespace)
	} else if m.allNamespaces {
		title += " - all namespaces"
	}
	s.WriteString(devToolsTitleStyle.Render(title))
	s.WriteString("\n\n")

	if m.loading || m.loadingAction {
		s.WriteString("\n")
		s.WriteString(m.spinner.View())
		s.WriteString("\n")
		return devToolsContainerStyle.Render(s.String())
	}

	// Filter
	if m.filtering {
		s.WriteString("Filter: ")
		s.WriteString(m.filterInput.View())
		s.WriteString("\n\n")
	}

	// Secrets list
	if len(m.filtered) == 0 {
		s.WriteString(devToolsDescriptionStyle.Render("No secrets found"))
		if m.filterInput.Value() != "" {
			s.WriteString(devToolsDescriptionStyle.Render(fmt.Sprintf(" matching '%s'", m.filterInput.Value())))
		}
		s.WriteString("\n\n")
	} else {
		// Show first 8 secrets with numbers
		maxItems := 8
		if len(m.filtered) < maxItems {
			maxItems = len(m.filtered)
		}

		for i := 0; i < maxItems; i++ {
			secret := m.filtered[i]

			// Number
			numberStr := devToolsNumberStyle.Render(fmt.Sprintf("%d.", i+1))

			// Secret name
			secretStr := secret.Name

			// Add selection indicator
			if i == m.selected {
				secretStr = devToolsSelectedStyle.Render("▸ " + secretStr)
			} else {
				secretStr = "  " + devToolsItemStyle.Render(secretStr)
			}

			// Type indicator
			typeStr := m.getTypeString(secret.Type)

			s.WriteString(numberStr + secretStr + " " + typeStr)
			s.WriteString("\n")

			// Secret details (indented)
			details := fmt.Sprintf("Type: %s, Keys: %d, Age: %s",
				secret.Type, secret.DataCount, secret.Age)
			if m.allNamespaces {
				details = fmt.Sprintf("Namespace: %s, %s", secret.Namespace, details)
			}
			s.WriteString(devToolsDescriptionStyle.Render("   " + details))
			s.WriteString("\n")
		}

		if len(m.filtered) > maxItems {
			s.WriteString("\n")
			s.WriteString(devToolsDescriptionStyle.Render(
				fmt.Sprintf("   ... and %d more secrets (use arrows to navigate)", len(m.filtered)-maxItems)))
			s.WriteString("\n")
		}
	}

	// Options
	s.WriteString("\n")
	s.WriteString(devToolsNumberStyle.Render("9.") + "  " + devToolsItemStyle.Render("Create New Secret"))
	s.WriteString("\n")
	s.WriteString(devToolsDescriptionStyle.Render("   Create a new Kubernetes secret"))
	s.WriteString("\n")

	// Add back option
	s.WriteString(devToolsNumberStyle.Render("0.") + "  " + devToolsItemStyle.Render("Back to Main Menu"))
	s.WriteString("\n")
	s.WriteString(devToolsDescriptionStyle.Render("   Return to the main menu"))

	// Message
	if m.message != "" {
		s.WriteString("\n")
		s.WriteString(devToolsInfoStyle.Render(m.message))
	}

	// Help
	s.WriteString("\n\n")
	helpText := "↑/k up • ↓/j down • 1-8 select • 9 create • 0 back • / filter • d delete • r refresh • q quit"
	s.WriteString(devToolsHelpStyle.Render(helpText))

	return devToolsContainerStyle.Render(s.String())
}

func (m *DevToolsSecretsModel) getTypeString(secretType string) string {
	switch secretType {
	case "kubernetes.io/service-account-token":
		return devToolsInfoStyle.Render("[ServiceAccount]")
	case "kubernetes.io/dockerconfigjson":
		return devToolsInfoStyle.Render("[DockerConfig]")
	case "kubernetes.io/tls":
		return devToolsSuccessStyle.Render("[TLS]")
	case "Opaque":
		return devToolsItemStyle.Render("[Opaque]")
	default:
		return devToolsDescriptionStyle.Render("[" + secretType + "]")
	}
}

func (m *DevToolsSecretsModel) applyFilter() {
	filter := strings.ToLower(m.filterInput.Value())
	if filter == "" {
		m.filtered = m.secrets
	} else {
		filtered := []SecretInfo{}
		for _, secret := range m.secrets {
			if strings.Contains(strings.ToLower(secret.Name), filter) ||
				strings.Contains(strings.ToLower(secret.Namespace), filter) ||
				strings.Contains(strings.ToLower(secret.Type), filter) {
				filtered = append(filtered, secret)
			}
		}
		m.filtered = filtered
	}
	m.selected = -1
}

func (m *DevToolsSecretsModel) deleteSecret() tea.Cmd {
	if m.selected < 0 || m.selected >= len(m.filtered) {
		return nil
	}

	secret := m.filtered[m.selected]
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := m.client.Clientset.CoreV1().Secrets(secret.Namespace).Delete(
			ctx, secret.Name, metav1.DeleteOptions{})
		if err != nil {
			return secretErrorMsg{err}
		}

		return m.loadSecrets()
	}
}

// GetSelectedSecret returns the selected secret
func (m *DevToolsSecretsModel) GetSelectedSecret() *SecretInfo {
	if m.secretSelected && m.selected >= 0 && m.selected < len(m.filtered) {
		return &m.filtered[m.selected]
	}
	return nil
}

// IsCreateSelected returns true if create new secret was selected
func (m *DevToolsSecretsModel) IsCreateSelected() bool {
	return m.secretSelected && m.selected == -2
}

// Helper function
func formatAge(t time.Time) string {
	duration := time.Since(t)
	if duration.Hours() > 24*30 {
		return fmt.Sprintf("%dm", int(duration.Hours()/(24*30)))
	} else if duration.Hours() > 24 {
		return fmt.Sprintf("%dd", int(duration.Hours()/24))
	} else if duration.Hours() > 1 {
		return fmt.Sprintf("%dh", int(duration.Hours()))
	} else if duration.Minutes() > 1 {
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	}
	return fmt.Sprintf("%ds", int(duration.Seconds()))
}

// SecretActionsMenu creates actions for a specific secret
func SecretActionsMenu(secret SecretInfo) []DevToolsMenuItem {
	return []DevToolsMenuItem{
		{
			Number:      "1",
			Title:       "View Secret Data",
			Description: "Display decoded secret values",
		},
		{
			Number:      "2",
			Title:       "Edit Secret",
			Description: "Modify secret key-value pairs",
		},
		{
			Number:      "3",
			Title:       "Copy Secret",
			Description: "Duplicate secret to another namespace",
		},
		{
			Number:      "4",
			Title:       "Export as YAML",
			Description: "Export secret configuration",
		},
		{
			Number:      "5",
			Title:       "Export as ENV",
			Description: "Export as environment variables",
		},
		{
			Number:      "6",
			Title:       "Delete Secret",
			Description: "Permanently remove this secret",
		},
		{
			Number:      "0",
			Title:       "Back to Secrets",
			Description: "Return to secrets list",
		},
	}
}

// ViewSecretData displays the decoded secret data
func ViewSecretData(secret *corev1.Secret) string {
	var s strings.Builder

	s.WriteString(devToolsTitleStyle.Render(fmt.Sprintf("🔒 Secret: %s", secret.Name)))
	s.WriteString("\n\n")

	s.WriteString(devToolsInfoStyle.Render("Type: " + string(secret.Type)))
	s.WriteString("\n\n")

	// Sort keys for consistent display
	keys := make([]string, 0, len(secret.Data))
	for k := range secret.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Display each key-value pair
	for _, key := range keys {
		value := secret.Data[key]
		decodedValue := string(value) // Already decoded from base64

		s.WriteString(devToolsNumberStyle.Render(key + ":"))
		s.WriteString("\n")

		// Mask sensitive data partially
		displayValue := decodedValue
		if len(displayValue) > 20 {
			displayValue = displayValue[:8] + "..." + displayValue[len(displayValue)-8:]
		}

		s.WriteString(devToolsDescriptionStyle.Render("   " + displayValue))
		s.WriteString("\n")
	}

	return s.String()
}

// ExportSecretAsEnv exports secret as environment variables
func ExportSecretAsEnv(secret *corev1.Secret) string {
	var s strings.Builder

	// Sort keys for consistent display
	keys := make([]string, 0, len(secret.Data))
	for k := range secret.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Generate export statements
	for _, key := range keys {
		value := secret.Data[key]
		// Convert key to uppercase and replace non-alphanumeric chars with underscore
		envKey := strings.ToUpper(strings.ReplaceAll(key, "-", "_"))
		envKey = strings.ReplaceAll(envKey, ".", "_")

		s.WriteString(fmt.Sprintf("export %s='%s'\n", envKey, base64.StdEncoding.EncodeToString(value)))
	}

	return s.String()
}