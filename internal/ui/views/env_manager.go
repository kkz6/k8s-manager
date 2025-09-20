package views

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/karthickk/k8s-manager/internal/services"
	"github.com/karthickk/k8s-manager/internal/ui/components"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EnvManagerModel manages environment variables for a pod
type EnvManagerModel struct {
	namespace     string
	podName       string
	deployment    string
	client        *services.K8sClient
	menu          *components.Menu
	envVars       []corev1.EnvVar
	loading       bool
	quitting      bool
	errorMsg      string
	successMsg    string
}

// envVarsLoadedMsg is sent when env vars are loaded
type envVarsLoadedMsg struct {
	envVars    []corev1.EnvVar
	deployment string
	err        error
}

// envUpdateMsg is sent when env vars are updated
type envUpdateMsg struct {
	success bool
	message string
}

// NewEnvManagerModel creates a new environment variable manager
func NewEnvManagerModel(namespace, podName string) *EnvManagerModel {
	return &EnvManagerModel{
		namespace: namespace,
		podName:   podName,
		loading:   true,
	}
}

// Init initializes the model
func (m *EnvManagerModel) Init() tea.Cmd {
	return m.loadEnvVars
}

// Update handles messages
func (m *EnvManagerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "a":
			// Add new env var
			return m, m.addEnvVar()
		case "e":
			// Edit existing env var
			if selected := m.menu.GetSelected(); selected != nil && selected.ID != "back" {
				return m, m.editEnvVar(selected.ID)
			}
		case "d":
			// Delete env var
			if selected := m.menu.GetSelected(); selected != nil && selected.ID != "back" {
				return m, m.deleteEnvVar(selected.ID)
			}
		case "r":
			// Restart pod
			return m, m.restartPod()
		}

		// Handle menu selection
		if msg.String() == "enter" || msg.String() == " " {
			selected := m.menu.GetSelected()
			if selected != nil {
				switch selected.ID {
				case "add":
					return m, m.addEnvVar()
				case "restart":
					return m, m.restartPod()
				case "back":
					m.quitting = true
					return m, tea.Quit
				}
			}
		}

	case envVarsLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.errorMsg = msg.err.Error()
		} else {
			m.envVars = msg.envVars
			m.deployment = msg.deployment
			m.updateMenu()
		}
		return m, nil

	case envUpdateMsg:
		if msg.success {
			m.successMsg = msg.message
			// Reload env vars
			return m, tea.Sequence(
				tea.Tick(time.Second*2, func(time.Time) tea.Msg {
					return clearMessageMsg{}
				}),
				m.loadEnvVars,
			)
		} else {
			m.errorMsg = msg.message
		}
		return m, nil

	case clearMessageMsg:
		m.successMsg = ""
		m.errorMsg = ""
		return m, nil
	}

	// Update menu
	if !m.loading && m.menu != nil {
		newMenu, cmd := m.menu.Update(msg)
		if menu, ok := newMenu.(components.Menu); ok {
			m.menu = &menu
		}
		return m, cmd
	}

	return m, nil
}

// View renders the view
func (m *EnvManagerModel) View() string {
	if m.quitting {
		return ""
	}

	if m.loading {
		return components.NewLoadingScreen("Loading Environment Variables").View()
	}

	if m.errorMsg != "" && m.menu == nil {
		return components.BoxStyle.Render(
			components.RenderTitle("Environment Variables", "") + "\n\n" +
				components.RenderMessage("error", m.errorMsg) + "\n\n" +
				components.HelpStyle.Render("Press 'q' to quit"),
		)
	}

	// Build view
	var sections []string

	// Add menu
	if m.menu != nil {
		sections = append(sections, m.menu.View())
	}

	// Add messages
	if m.successMsg != "" {
		successBox := lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("42")).
			Padding(0, 1).
			Render(m.successMsg)
		sections = append(sections, successBox)
	}

	if m.errorMsg != "" {
		errorBox := components.ErrorMessageStyle.Render(m.errorMsg)
		sections = append(sections, errorBox)
	}

	// Add help text
	helpText := "a: add • e: edit • d: delete • r: restart pod • q: back"
	sections = append(sections, components.HelpStyle.Render(helpText))

	return strings.Join(sections, "\n\n")
}

// updateMenu updates the menu with env vars
func (m *EnvManagerModel) updateMenu() {
	menuItems := []components.MenuItem{}

	// Add env vars
	for _, env := range m.envVars {
		value := env.Value
		if len(value) > 50 {
			value = value[:47] + "..."
		}
		menuItems = append(menuItems, components.MenuItem{
			ID:          env.Name,
			Title:       env.Name,
			Description: value,
			Icon:        "🔧",
		})
	}

	// Add actions
	menuItems = append(menuItems,
		components.MenuItem{
			ID:          "add",
			Title:       "Add Environment Variable",
			Description: "Add a new environment variable",
			Icon:        "➕",
			Shortcut:    "a",
		},
		components.MenuItem{
			ID:          "restart",
			Title:       "Restart Pod",
			Description: "Apply changes by restarting the pod",
			Icon:        "🔄",
			Shortcut:    "r",
		},
		components.MenuItem{
			ID:          "back",
			Title:       "Back to Pod Actions",
			Description: "Return without making changes",
			Icon:        "↩️",
			Shortcut:    "b",
		},
	)

	title := fmt.Sprintf("🔧 Environment Variables: %s", m.podName)
	if m.deployment != "" {
		title += fmt.Sprintf(" (Deployment: %s)", m.deployment)
	}

	m.menu = components.NewDevToolsMenu(title, menuItems)
}

// loadEnvVars loads environment variables from the pod
func (m *EnvManagerModel) loadEnvVars() tea.Msg {
	client, err := services.GetK8sClient()
	if err != nil {
		return envVarsLoadedMsg{err: err}
	}
	m.client = client

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get the pod
	pod, err := client.Clientset.CoreV1().Pods(m.namespace).Get(ctx, m.podName, metav1.GetOptions{})
	if err != nil {
		return envVarsLoadedMsg{err: err}
	}

	// Get deployment name from owner references
	deploymentName := ""
	for _, owner := range pod.OwnerReferences {
		if owner.Kind == "ReplicaSet" {
			// Extract deployment name from ReplicaSet name
			parts := strings.Split(owner.Name, "-")
			if len(parts) > 1 {
				deploymentName = strings.Join(parts[:len(parts)-1], "-")
				break
			}
		}
	}

	// Get env vars from the first container
	var envVars []corev1.EnvVar
	if len(pod.Spec.Containers) > 0 {
		envVars = pod.Spec.Containers[0].Env
	}

	return envVarsLoadedMsg{
		envVars:    envVars,
		deployment: deploymentName,
	}
}

// addEnvVar adds a new environment variable
func (m *EnvManagerModel) addEnvVar() tea.Cmd {
	return func() tea.Msg {
		// Create form for adding env var
		keyField := components.NewInputField("Key")
		keyField.Placeholder = "e.g., APP_URL"
		keyField.CharLimit = 64
		
		valueField := components.NewInputField("Value")
		valueField.Placeholder = "e.g., https://example.com"
		valueField.CharLimit = 256
		
		form := components.NewForm("Add Environment Variable", []*components.InputField{
			keyField,
			valueField,
		})
		
		p := tea.NewProgram(form)
		model, err := p.Run()
		if err != nil {
			return envUpdateMsg{
				success: false,
				message: fmt.Sprintf("Error: %v", err),
			}
		}
		
		formModel := model.(components.FormModel)
		if !formModel.IsSubmitted() {
			return envUpdateMsg{
				success: false,
				message: "Cancelled",
			}
		}
		
		values := formModel.GetValues()
		key := values["Key"]
		value := values["Value"]
		
		if key == "" {
			return envUpdateMsg{
				success: false,
				message: "Key cannot be empty",
			}
		}
		
		// Update deployment with new env var
		return m.updateDeploymentEnv(key, value, false)
	}
}

// editEnvVar edits an existing environment variable
func (m *EnvManagerModel) editEnvVar(name string) tea.Cmd {
	return func() tea.Msg {
		// Find current value
		var currentValue string
		for _, env := range m.envVars {
			if env.Name == name {
				currentValue = env.Value
				break
			}
		}
		
		// Create form for editing env var
		keyField := components.NewInputField("Key")
		keyField.SetValue(name)
		keyField.CharLimit = 64
		keyField.Blur() // Make it read-only by not focusing
		
		valueField := components.NewInputField("Value")
		valueField.SetValue(currentValue)
		valueField.CharLimit = 256
		
		form := components.NewForm("Edit Environment Variable", []*components.InputField{
			keyField,
			valueField,
		})
		
		// Focus on value field
		form.Fields[0].Blur()
		form.Fields[1].Focus()
		
		p := tea.NewProgram(form)
		model, err := p.Run()
		if err != nil {
			return envUpdateMsg{
				success: false,
				message: fmt.Sprintf("Error: %v", err),
			}
		}
		
		formModel := model.(components.FormModel)
		if !formModel.IsSubmitted() {
			return envUpdateMsg{
				success: false,
				message: "Cancelled",
			}
		}
		
		values := formModel.GetValues()
		newValue := values["Value"]
		
		// Update deployment with new value
		return m.updateDeploymentEnv(name, newValue, false)
	}
}

// deleteEnvVar deletes an environment variable
func (m *EnvManagerModel) deleteEnvVar(name string) tea.Cmd {
	return func() tea.Msg {
		return m.updateDeploymentEnv(name, "", true)
	}
}

// restartPod restarts the pod to apply changes
func (m *EnvManagerModel) restartPod() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		gracePeriod := int64(30)
		err := m.client.Clientset.CoreV1().Pods(m.namespace).Delete(ctx, m.podName, metav1.DeleteOptions{
			GracePeriodSeconds: &gracePeriod,
		})

		if err != nil {
			return envUpdateMsg{
				success: false,
				message: fmt.Sprintf("Error restarting pod: %v", err),
			}
		}

		return envUpdateMsg{
			success: true,
			message: "Pod restart initiated successfully",
		}
	}
}

// clearMessageMsg clears success/error messages
type clearMessageMsg struct{}

// updateDeploymentEnv updates the deployment with new environment variable
func (m *EnvManagerModel) updateDeploymentEnv(key, value string, delete bool) envUpdateMsg {
	if m.deployment == "" {
		return envUpdateMsg{
			success: false,
			message: "Cannot update env vars: Pod not managed by deployment",
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get the deployment
	deployment, err := m.client.Clientset.AppsV1().Deployments(m.namespace).Get(ctx, m.deployment, metav1.GetOptions{})
	if err != nil {
		return envUpdateMsg{
			success: false,
			message: fmt.Sprintf("Failed to get deployment: %v", err),
		}
	}

	// Update environment variables in all containers
	for i := range deployment.Spec.Template.Spec.Containers {
		container := &deployment.Spec.Template.Spec.Containers[i]
		
		if delete {
			// Remove the environment variable
			newEnv := []corev1.EnvVar{}
			for _, env := range container.Env {
				if env.Name != key {
					newEnv = append(newEnv, env)
				}
			}
			container.Env = newEnv
		} else {
			// Add or update the environment variable
			found := false
			for j, env := range container.Env {
				if env.Name == key {
					container.Env[j].Value = value
					found = true
					break
				}
			}
			
			if !found {
				container.Env = append(container.Env, corev1.EnvVar{
					Name:  key,
					Value: value,
				})
			}
		}
	}

	// Update the deployment
	_, err = m.client.Clientset.AppsV1().Deployments(m.namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return envUpdateMsg{
			success: false,
			message: fmt.Sprintf("Failed to update deployment: %v", err),
		}
	}

	if delete {
		return envUpdateMsg{
			success: true,
			message: fmt.Sprintf("Deleted environment variable %s. Pod will restart automatically.", key),
		}
	}

	return envUpdateMsg{
		success: true,
		message: fmt.Sprintf("Set %s=%s. Pod will restart automatically.", key, value),
	}
}