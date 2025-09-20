package views

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/karthickk/k8s-manager/internal/services"
	"github.com/karthickk/k8s-manager/internal/ui/components"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SecretsViewModel represents the secrets list view
type SecretsViewModel struct {
	client       *services.K8sClient
	secrets      []corev1.Secret
	menu         *components.Menu
	loading      bool
	spinner      components.SpinnerModel
	err          error
	quitting     bool
	allNamespaces bool
}

// secretsFetchedMsg is sent when secrets are fetched
type secretsFetchedMsg struct {
	secrets []corev1.Secret
	err     error
}

// ShowSecretsView shows the interactive secrets view
func ShowSecretsView() error {
	client, err := services.GetK8sClient()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	model := &SecretsViewModel{
		client:  client,
		loading: true,
		spinner: components.NewSpinner("Loading Secrets..."),
	}

	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}

// Init initializes the model
func (m *SecretsViewModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Init(),
		m.fetchSecrets,
	)
}

// Update handles messages
func (m *SecretsViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "r":
			m.loading = true
			return m, m.fetchSecrets
		case "n":
			// Toggle namespace view
			m.allNamespaces = !m.allNamespaces
			m.loading = true
			return m, m.fetchSecrets
		case "b":
			// Quick back navigation
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		// Handle resize

	case secretsFetchedMsg:
		m.loading = false
		m.secrets = msg.secrets
		m.err = msg.err
		if m.err == nil {
			m.updateMenu()
		}
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	// Update menu
	if !m.loading && m.menu != nil {
		var cmd tea.Cmd
		// Check if enter was pressed
		if kMsg, ok := msg.(tea.KeyMsg); ok && (kMsg.String() == "enter" || kMsg.String() == " ") {
			selected := m.menu.GetSelected()
			if selected != nil {
				if selected.ID == "back" {
					m.quitting = true
					return m, tea.Quit
				} else {
					// View Secret details
					return m, m.viewSecretDetails(selected.ID)
				}
			}
		}
		
		newMenu, cmd := m.menu.Update(msg)
		if menu, ok := newMenu.(components.Menu); ok {
			m.menu = &menu
		}
		return m, cmd
	}

	return m, nil
}

// View renders the view
func (m *SecretsViewModel) View() string {
	if m.quitting {
		return ""
	}

	if m.loading {
		loadingView := components.NewLoadingScreen("Loading Secrets")
		return loadingView.View()
	}

	if m.err != nil {
		return components.BoxStyle.Render(
			components.RenderTitle("Secrets", "") + "\n\n" +
				components.RenderMessage("error", m.err.Error()) + "\n\n" +
				components.HelpStyle.Render("Press 'r' to retry, 'q' to quit"),
		)
	}

	// Show menu
	if m.menu == nil {
		return "No Secrets available"
	}
	
	return m.menu.View()
}

// fetchSecrets fetches the secrets list
func (m *SecretsViewModel) fetchSecrets() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	namespace := ""
	if !m.allNamespaces {
		namespace = services.GetCurrentNamespace()
	}

	listOptions := metav1.ListOptions{}

	secrets, err := m.client.Clientset.CoreV1().Secrets(namespace).List(ctx, listOptions)
	if err != nil {
		return secretsFetchedMsg{err: err}
	}

	return secretsFetchedMsg{secrets: secrets.Items}
}

// updateMenu updates the menu with secret data
func (m *SecretsViewModel) updateMenu() {
	menuItems := []components.MenuItem{}
	
	// Create menu items for each secret
	for _, secret := range m.secrets {
		age := services.FormatAge(secret.CreationTimestamp.Time)
		
		// Get secret type display
		secretType := string(secret.Type)
		if secretType == string(corev1.SecretTypeOpaque) {
			secretType = "Opaque"
		}
		
		// Create a formatted title with secret info
		title := secret.Name
		description := fmt.Sprintf("Type: %s, Keys: %d, Namespace: %s, Age: %s", 
			secretType, len(secret.Data), secret.Namespace, age)
		
		// Choose icon based on secret type
		icon := "🔐"
		if strings.Contains(string(secret.Type), "tls") {
			icon = "🔒"
		} else if strings.Contains(string(secret.Type), "docker") {
			icon = "🐳"
		} else if strings.Contains(string(secret.Type), "service-account") {
			icon = "👤"
		}
		
		menuItems = append(menuItems, components.MenuItem{
			ID:          fmt.Sprintf("%s/%s", secret.Namespace, secret.Name),
			Title:       title,
			Description: description,
			Icon:        icon,
		})
	}
	
	// Add back option
	menuItems = append(menuItems, components.MenuItem{
		ID:          "back",
		Title:       "Back to Main Menu",
		Description: "Return to the main menu",
		Icon:        "⬅️",
		Shortcut:    "b",
	})
	
	// Create title based on namespace
	title := fmt.Sprintf("🔐 Secrets (%d items)", len(m.secrets))
	if m.allNamespaces {
		title += " - All Namespaces"
	} else {
		title += fmt.Sprintf(" - Namespace: %s", services.GetCurrentNamespace())
	}
	
	// Create menu with DevTools style
	m.menu = components.NewDevToolsMenu(title, menuItems)
}

// viewSecretDetails shows secret details
func (m *SecretsViewModel) viewSecretDetails(id string) tea.Cmd {
	return func() tea.Msg {
		parts := strings.Split(id, "/")
		if len(parts) != 2 {
			return nil
		}
		
		namespace := parts[0]
		name := parts[1]
		
		// Show secret details view
		model := NewSecretDetailsModel(namespace, name)
		p := tea.NewProgram(model)
		if _, err := p.Run(); err != nil {
			return components.ErrorMsg{Error: err}
		}
		
		return nil
	}
}