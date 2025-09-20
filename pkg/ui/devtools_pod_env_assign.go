package ui

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/karthickk/k8s-manager/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PodEnvAssignModel manages environment assignment to pods
type PodEnvAssignModel struct {
	pod             *corev1.Pod
	client          *k8s.Client
	secrets         []SecretInfo
	configMaps      []ConfigMapInfo
	envVars         []EnvVarAssignment
	selected        int
	step            int // 0: select source, 1: select secret/cm, 2: select keys, 3: review, 4: direct input
	sourceType      int // 0: secret, 1: configmap, 2: direct value
	selectedItem    string
	selectedKeys    map[string]bool
	message         string
	messageType     string
	directEnvName   string
	directEnvValue  string
	inputMode       int // 0: name, 1: value
}

// ConfigMapInfo holds configmap information
type ConfigMapInfo struct {
	Name      string
	Namespace string
	DataCount int
	ConfigMap *corev1.ConfigMap
}

// EnvVarAssignment represents an environment variable to assign
type EnvVarAssignment struct {
	Name       string
	Value      string
	SourceType string // "secret", "configmap", "direct"
	SourceName string
	Key        string
}

// NewPodEnvAssignModel creates a new pod environment assignment model
func NewPodEnvAssignModel(pod *corev1.Pod, client *k8s.Client) *PodEnvAssignModel {
	return &PodEnvAssignModel{
		pod:          pod,
		client:       client,
		envVars:      []EnvVarAssignment{},
		selectedKeys: make(map[string]bool),
		selected:     0,
		step:         0,
	}
}

func (m *PodEnvAssignModel) Init() tea.Cmd {
	return m.loadResources
}

func (m *PodEnvAssignModel) loadResources() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Load secrets
	secrets, err := m.client.Clientset.CoreV1().Secrets(m.pod.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return resourceLoadMsg{err: err}
	}

	secretInfos := make([]SecretInfo, 0, len(secrets.Items))
	for _, secret := range secrets.Items {
		// Skip service account tokens
		if secret.Type == corev1.SecretTypeServiceAccountToken {
			continue
		}
		info := SecretInfo{
			Name:      secret.Name,
			Namespace: secret.Namespace,
			Type:      string(secret.Type),
			DataCount: len(secret.Data),
			Secret:    &secret,
		}
		secretInfos = append(secretInfos, info)
	}

	// Load configmaps
	configMaps, err := m.client.Clientset.CoreV1().ConfigMaps(m.pod.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return resourceLoadMsg{err: err}
	}

	configMapInfos := make([]ConfigMapInfo, 0, len(configMaps.Items))
	for _, cm := range configMaps.Items {
		info := ConfigMapInfo{
			Name:      cm.Name,
			Namespace: cm.Namespace,
			DataCount: len(cm.Data),
			ConfigMap: &cm,
		}
		configMapInfos = append(configMapInfos, info)
	}

	return resourceLoadMsg{
		secrets:    secretInfos,
		configMaps: configMapInfos,
	}
}

type resourceLoadMsg struct {
	secrets    []SecretInfo
	configMaps []ConfigMapInfo
	err        error
}

func (m *PodEnvAssignModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case resourceLoadMsg:
		if msg.err != nil {
			m.message = fmt.Sprintf("Error loading resources: %v", msg.err)
			m.messageType = "error"
		} else {
			m.secrets = msg.secrets
			m.configMaps = msg.configMaps
		}
		return m, nil

	case tea.KeyMsg:
		keyStr := msg.String()

		switch m.step {
		case 0: // Select source type
			switch keyStr {
			case "1": // Secret
				m.sourceType = 0
				m.step = 1
				return m, nil

			case "2": // ConfigMap
				m.sourceType = 1
				m.step = 1
				return m, nil

			case "3": // Direct value
				m.sourceType = 2
				m.step = 4 // Go to direct input step
				m.inputMode = 0 // Start with name input
				return m, nil

			case "0", "q", "esc":
				return m, tea.Quit
			}

		case 1: // Select secret or configmap
			if keyStr >= "1" && keyStr <= "9" {
				num := int(keyStr[0] - '0') - 1
				if m.sourceType == 0 && num < len(m.secrets) {
					m.selectedItem = m.secrets[num].Name
					m.step = 2
					m.loadKeys()
				} else if m.sourceType == 1 && num < len(m.configMaps) {
					m.selectedItem = m.configMaps[num].Name
					m.step = 2
					m.loadKeys()
				}
				return m, nil
			}

			switch keyStr {
			case "0", "b":
				m.step = 0
				return m, nil

			case "q", "esc":
				return m, tea.Quit
			}

		case 2: // Select keys
			if keyStr >= "1" && keyStr <= "9" {
				// Toggle key selection
				num := int(keyStr[0] - '0')
				keys := m.getAvailableKeys()
				if num <= len(keys) {
					key := keys[num-1]
					m.selectedKeys[key] = !m.selectedKeys[key]
				}
				return m, nil
			}

			switch keyStr {
			case "a": // Select all
				keys := m.getAvailableKeys()
				for _, key := range keys {
					m.selectedKeys[key] = true
				}
				return m, nil

			case "n": // Select none
				m.selectedKeys = make(map[string]bool)
				return m, nil

			case "c": // Continue
				if len(m.selectedKeys) > 0 {
					m.createEnvVars()
					m.step = 3
				} else {
					m.message = "Select at least one key"
					m.messageType = "warning"
				}
				return m, nil

			case "0", "b":
				m.step = 1
				m.selectedKeys = make(map[string]bool)
				return m, nil

			case "q", "esc":
				return m, tea.Quit
			}

		case 3: // Review and apply
			switch keyStr {
			case "a": // Apply
				return m, m.applyEnvVars()

			case "n": // Add another
				if m.sourceType == 2 {
					m.step = 4
					m.directEnvName = ""
					m.directEnvValue = ""
					m.inputMode = 0
				}
				return m, nil

			case "b": // Back
				if m.sourceType == 2 {
					m.step = 0
				} else {
					m.step = 2
				}
				return m, nil

			case "q", "esc":
				return m, tea.Quit
			}

		case 4: // Direct input
			// Handle character input for direct env vars
			if len(keyStr) == 1 && keyStr != " " {
				r := rune(keyStr[0])
				if m.inputMode == 0 {
					// Name: Allow letters, digits, underscore
					if unicode.IsLetter(r) || unicode.IsDigit(r) || keyStr == "_" {
						m.directEnvName += strings.ToUpper(keyStr)
					}
				} else {
					// Value: Allow any printable character
					if unicode.IsPrint(r) {
						m.directEnvValue += keyStr
					}
				}
				return m, nil
			}
			
			// Handle space in value mode
			if keyStr == " " && m.inputMode == 1 {
				m.directEnvValue += " "
				return m, nil
			}

			switch keyStr {
			case "backspace":
				if m.inputMode == 0 && len(m.directEnvName) > 0 {
					m.directEnvName = m.directEnvName[:len(m.directEnvName)-1]
				} else if m.inputMode == 1 && len(m.directEnvValue) > 0 {
					m.directEnvValue = m.directEnvValue[:len(m.directEnvValue)-1]
				}
				return m, nil

			case "enter", "tab":
				if m.inputMode == 0 && m.directEnvName != "" {
					m.inputMode = 1 // Move to value input
				} else if m.inputMode == 1 && m.directEnvValue != "" {
					// Add to env vars
					m.envVars = append(m.envVars, EnvVarAssignment{
						Name:       m.directEnvName,
						Value:      m.directEnvValue,
						SourceType: "direct",
					})
					m.directEnvName = ""
					m.directEnvValue = ""
					m.step = 3 // Go to review
				}
				return m, nil

			case "q", "esc":
				return m, tea.Quit
			}
		}

		// Navigation keys
		switch keyStr {
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}

		case "down", "j":
			if m.selected < m.getMaxSelection()-1 {
				m.selected++
			}

		case "enter", " ":
			// Handle selection based on step
			return m, m.handleEnterKey()
		}
	}

	return m, nil
}

func (m *PodEnvAssignModel) View() string {
	var s strings.Builder

	s.WriteString("\033[H\033[2J") // Clear screen

	// Title
	s.WriteString(devToolsTitleStyle.Render(fmt.Sprintf("🔧 Assign Environment to Pod: %s", m.pod.Name)))
	s.WriteString("\n\n")

	switch m.step {
	case 0: // Select source type
		s.WriteString(devToolsNumberStyle.Render("Select Environment Source:"))
		s.WriteString("\n\n")

		options := []struct {
			title string
			desc  string
			num   string
		}{
			{"From Secret", "Use values from Kubernetes secrets", "1"},
			{"From ConfigMap", "Use values from ConfigMaps", "2"},
			{"Direct Values", "Enter environment variables directly", "3"},
			{"Back", "Return to pod actions", "0"},
		}

		for i, opt := range options {
			if i == m.selected {
				s.WriteString(devToolsSelectedStyle.Render(fmt.Sprintf("▸ %s. %s", opt.num, opt.title)))
			} else {
				s.WriteString(fmt.Sprintf("  %s  %s", devToolsNumberStyle.Render(opt.num+"."), devToolsItemStyle.Render(opt.title)))
			}
			s.WriteString("\n")
			s.WriteString(devToolsDescriptionStyle.Render(fmt.Sprintf("   %s", opt.desc)))
			s.WriteString("\n")
			if i < len(options)-1 {
				s.WriteString("\n")
			}
		}

	case 1: // Select secret or configmap
		if m.sourceType == 0 {
			s.WriteString(devToolsNumberStyle.Render("Select Secret:"))
			s.WriteString("\n\n")

			if len(m.secrets) == 0 {
				s.WriteString(devToolsDescriptionStyle.Render("No secrets available in this namespace"))
			} else {
				for i, secret := range m.secrets {
					if i == m.selected {
						s.WriteString(devToolsSelectedStyle.Render(fmt.Sprintf("▸ %d. %s", i+1, secret.Name)))
					} else {
						s.WriteString(fmt.Sprintf("  %s  %s", devToolsNumberStyle.Render(fmt.Sprintf("%d.", i+1)), devToolsItemStyle.Render(secret.Name)))
					}
					s.WriteString("\n")
					s.WriteString(devToolsDescriptionStyle.Render(
						fmt.Sprintf("   Type: %s, Keys: %d", secret.Type, secret.DataCount)))
					s.WriteString("\n")
				}
			}
		} else {
			s.WriteString(devToolsNumberStyle.Render("Select ConfigMap:"))
			s.WriteString("\n\n")

			if len(m.configMaps) == 0 {
				s.WriteString(devToolsDescriptionStyle.Render("No configmaps available in this namespace"))
			} else {
				for i, cm := range m.configMaps {
					if i == m.selected {
						s.WriteString(devToolsSelectedStyle.Render(fmt.Sprintf("▸ %d. %s", i+1, cm.Name)))
					} else {
						s.WriteString(fmt.Sprintf("  %s  %s", devToolsNumberStyle.Render(fmt.Sprintf("%d.", i+1)), devToolsItemStyle.Render(cm.Name)))
					}
					s.WriteString("\n")
					s.WriteString(devToolsDescriptionStyle.Render(fmt.Sprintf("   Keys: %d", cm.DataCount)))
					s.WriteString("\n")
				}
			}
		}

		s.WriteString("\n")
		backIdx := len(m.secrets)
		if m.sourceType == 1 {
			backIdx = len(m.configMaps)
		}
		if m.selected == backIdx {
			s.WriteString(devToolsSelectedStyle.Render("▸ 0. Back"))
		} else {
			s.WriteString(fmt.Sprintf("  %s  %s", devToolsNumberStyle.Render("0."), devToolsItemStyle.Render("Back")))
		}

	case 2: // Select keys
		s.WriteString(devToolsNumberStyle.Render(fmt.Sprintf("Select Keys from %s:", m.selectedItem)))
		s.WriteString("\n\n")

		keys := m.getAvailableKeys()
		for i, key := range keys {
			checkbox := "☐"
			if m.selectedKeys[key] {
				checkbox = "☑"
			}

			if i == m.selected {
				s.WriteString(devToolsSelectedStyle.Render(fmt.Sprintf("▸ %d. %s %s", i+1, checkbox, key)))
			} else {
				s.WriteString(fmt.Sprintf("  %s  %s %s", devToolsNumberStyle.Render(fmt.Sprintf("%d.", i+1)), checkbox, devToolsItemStyle.Render(key)))
			}
			s.WriteString("\n")
		}

		s.WriteString("\n")
		s.WriteString(devToolsHelpStyle.Render("↑/↓ navigate • Enter/Space toggle • a select all • n none • c continue • 0 back"))

	case 3: // Review
		s.WriteString(devToolsNumberStyle.Render("Review Environment Variables:"))
		s.WriteString("\n\n")

		if len(m.envVars) == 0 {
			s.WriteString(devToolsDescriptionStyle.Render("No environment variables to add"))
		} else {
			for _, env := range m.envVars {
				if env.SourceType == "direct" {
					s.WriteString(fmt.Sprintf("%s = %s\n",
						devToolsInfoStyle.Render(env.Name),
						devToolsDescriptionStyle.Render(env.Value)))
				} else {
					s.WriteString(fmt.Sprintf("%s = %s:%s/%s\n",
						devToolsInfoStyle.Render(env.Name),
						devToolsDescriptionStyle.Render(env.SourceType),
						env.SourceName,
						env.Key))
				}
			}
		}

		s.WriteString("\n")
		s.WriteString(devToolsSuccessStyle.Render(fmt.Sprintf("Will add %d environment variables to pod", len(m.envVars))))
		s.WriteString("\n\n")
		if m.sourceType == 2 {
			s.WriteString(devToolsHelpStyle.Render("a apply • n add another • b back • q quit"))
		} else {
			s.WriteString(devToolsHelpStyle.Render("a apply • b back • q quit"))
		}

	case 4: // Direct input
		s.WriteString(devToolsNumberStyle.Render("Add Environment Variable:"))
		s.WriteString("\n\n")

		// Name input
		nameStyle := devToolsItemStyle
		if m.inputMode == 0 {
			nameStyle = devToolsSelectedStyle
		}
		s.WriteString(nameStyle.Render("Name:"))
		s.WriteString("  ")
		if m.directEnvName == "" {
			s.WriteString(devToolsDescriptionStyle.Render("Enter variable name"))
		} else {
			s.WriteString(devToolsInfoStyle.Render(m.directEnvName))
		}
		if m.inputMode == 0 {
			s.WriteString(devToolsSelectedStyle.Render("_")) // Cursor
		}
		s.WriteString("\n\n")

		// Value input
		valueStyle := devToolsItemStyle
		if m.inputMode == 1 {
			valueStyle = devToolsSelectedStyle
		}
		s.WriteString(valueStyle.Render("Value:"))
		s.WriteString(" ")
		if m.directEnvValue == "" {
			s.WriteString(devToolsDescriptionStyle.Render("Enter variable value"))
		} else {
			s.WriteString(devToolsInfoStyle.Render(m.directEnvValue))
		}
		if m.inputMode == 1 {
			s.WriteString(devToolsSelectedStyle.Render("_")) // Cursor
		}
		s.WriteString("\n\n")

		// Existing variables
		if len(m.envVars) > 0 {
			s.WriteString("\n")
			s.WriteString(devToolsDescriptionStyle.Render(fmt.Sprintf("Already added %d variable(s):", len(m.envVars))))
			s.WriteString("\n")
			for _, env := range m.envVars {
				if env.SourceType == "direct" {
					s.WriteString(devToolsDescriptionStyle.Render(fmt.Sprintf("  • %s = %s\n", env.Name, env.Value)))
				}
			}
		}

		s.WriteString("\n\n")
		s.WriteString(devToolsHelpStyle.Render("Type to enter • Tab/Enter next field • Enter (on value) save • Esc cancel"))
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

	// Add navigation help at bottom
	if m.step < 3 && m.step != 4 {
		s.WriteString("\n\n")
		s.WriteString(devToolsHelpStyle.Render("↑/k up • ↓/j down • Enter select • q/Esc quit"))
	}

	return devToolsContainerStyle.Render(s.String())
}

func (m *PodEnvAssignModel) loadKeys() {
	m.selectedKeys = make(map[string]bool)
	// Keys will be loaded from the selected secret or configmap
}

func (m *PodEnvAssignModel) getAvailableKeys() []string {
	var keys []string

	if m.sourceType == 0 { // Secret
		for _, secret := range m.secrets {
			if secret.Name == m.selectedItem {
				for key := range secret.Secret.Data {
					keys = append(keys, key)
				}
				break
			}
		}
	} else if m.sourceType == 1 { // ConfigMap
		for _, cm := range m.configMaps {
			if cm.Name == m.selectedItem {
				for key := range cm.ConfigMap.Data {
					keys = append(keys, key)
				}
				break
			}
		}
	}

	return keys
}

func (m *PodEnvAssignModel) createEnvVars() {
	m.envVars = []EnvVarAssignment{}

	for key, selected := range m.selectedKeys {
		if !selected {
			continue
		}

		envName := strings.ToUpper(strings.ReplaceAll(key, "-", "_"))
		envName = strings.ReplaceAll(envName, ".", "_")

		env := EnvVarAssignment{
			Name:       envName,
			Key:        key,
			SourceName: m.selectedItem,
		}

		if m.sourceType == 0 {
			env.SourceType = "secret"
		} else {
			env.SourceType = "configmap"
		}

		m.envVars = append(m.envVars, env)
	}
}

func (m *PodEnvAssignModel) applyEnvVars() tea.Cmd {
	return func() tea.Msg {
		// This would typically update the pod's deployment or create a patch
		// For now, we'll just show what would be applied

		// In a real implementation, you would:
		// 1. Find the deployment/statefulset that manages this pod
		// 2. Update its pod template with the new environment variables
		// 3. Trigger a rolling update

		return envApplyMsg{success: true}
	}
}

type envApplyMsg struct {
	success bool
	err     error
}

func (m *PodEnvAssignModel) getMaxSelection() int {
	switch m.step {
	case 0:
		return 4 // 3 options + back
	case 1:
		if m.sourceType == 0 {
			return len(m.secrets) + 1 // +1 for back
		}
		return len(m.configMaps) + 1 // +1 for back
	case 2:
		return len(m.getAvailableKeys())
	case 4:
		return 0 // No selection in direct input mode
	default:
		return 0
	}
}

func (m *PodEnvAssignModel) handleEnterKey() tea.Cmd {
	switch m.step {
	case 0: // Select source type
		switch m.selected {
		case 0: // Secret
			m.sourceType = 0
			m.step = 1
			m.selected = 0
		case 1: // ConfigMap
			m.sourceType = 1
			m.step = 1
			m.selected = 0
		case 2: // Direct values
			m.sourceType = 2
			m.step = 3
		case 3: // Back
			return tea.Quit
		}

	case 1: // Select secret or configmap
		if m.sourceType == 0 { // Secrets
			if m.selected < len(m.secrets) {
				m.selectedItem = m.secrets[m.selected].Name
				m.step = 2
				m.selected = 0
				m.loadKeys()
			} else if m.selected == len(m.secrets) { // Back
				m.step = 0
				m.selected = 0
			}
		} else { // ConfigMaps
			if m.selected < len(m.configMaps) {
				m.selectedItem = m.configMaps[m.selected].Name
				m.step = 2
				m.selected = 0
				m.loadKeys()
			} else if m.selected == len(m.configMaps) { // Back
				m.step = 0
				m.selected = 0
			}
		}

	case 2: // Select keys
		keys := m.getAvailableKeys()
		if m.selected < len(keys) {
			// Toggle key selection
			key := keys[m.selected]
			m.selectedKeys[key] = !m.selectedKeys[key]
		}
	}

	return nil
}

// ShowPodEnvAssignment shows the pod environment assignment interface
func ShowPodEnvAssignment(pod *corev1.Pod, client *k8s.Client) error {
	model := NewPodEnvAssignModel(pod, client)
	p := tea.NewProgram(model, tea.WithAltScreen())

	result, err := p.Run()
	if err != nil {
		return err
	}

	if m, ok := result.(*PodEnvAssignModel); ok && len(m.envVars) > 0 {
		fmt.Println("\n✅ Environment variables prepared for assignment:")
		for _, env := range m.envVars {
			fmt.Printf("  %s from %s:%s/%s\n", env.Name, env.SourceType, env.SourceName, env.Key)
		}
		fmt.Println("\nNote: Update the deployment to apply these changes permanently.")
	}

	return nil
}