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

// ConfigMapsViewModel represents the configmaps list view
type ConfigMapsViewModel struct {
	client       *services.K8sClient
	configMaps   []corev1.ConfigMap
	menu         *components.Menu
	loading      bool
	spinner      components.SpinnerModel
	err          error
	quitting     bool
	allNamespaces bool
}

// configMapsFetchedMsg is sent when configmaps are fetched
type configMapsFetchedMsg struct {
	configMaps []corev1.ConfigMap
	err        error
}

// ShowConfigMapsView shows the interactive configmaps view
func ShowConfigMapsView() error {
	client, err := services.GetK8sClient()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	model := &ConfigMapsViewModel{
		client:  client,
		loading: true,
		spinner: components.NewSpinner("Loading ConfigMaps..."),
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}

// Init initializes the model
func (m *ConfigMapsViewModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Init(),
		m.fetchConfigMaps,
	)
}

// Update handles messages
func (m *ConfigMapsViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "r":
			m.loading = true
			return m, m.fetchConfigMaps
		case "n":
			// Toggle namespace view
			m.allNamespaces = !m.allNamespaces
			m.loading = true
			return m, m.fetchConfigMaps
		case "b":
			// Quick back navigation
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		// Handle resize

	case configMapsFetchedMsg:
		m.loading = false
		m.configMaps = msg.configMaps
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
					// View ConfigMap details
					return m, m.viewConfigMapDetails(selected.ID)
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
func (m *ConfigMapsViewModel) View() string {
	if m.quitting {
		return ""
	}

	if m.loading {
		loadingView := components.NewLoadingScreen("Loading ConfigMaps")
		return loadingView.View()
	}

	if m.err != nil {
		return components.BoxStyle.Render(
			components.RenderTitle("ConfigMaps", "") + "\n\n" +
				components.RenderMessage("error", m.err.Error()) + "\n\n" +
				components.HelpStyle.Render("Press 'r' to retry, 'q' to quit"),
		)
	}

	// Show menu
	if m.menu == nil {
		return "No ConfigMaps available"
	}
	
	return m.menu.View()
}

// fetchConfigMaps fetches the configmaps list
func (m *ConfigMapsViewModel) fetchConfigMaps() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	namespace := ""
	if !m.allNamespaces {
		namespace = services.GetCurrentNamespace()
	}

	listOptions := metav1.ListOptions{}

	configMaps, err := m.client.Clientset.CoreV1().ConfigMaps(namespace).List(ctx, listOptions)
	if err != nil {
		return configMapsFetchedMsg{err: err}
	}

	return configMapsFetchedMsg{configMaps: configMaps.Items}
}

// updateMenu updates the menu with configmap data
func (m *ConfigMapsViewModel) updateMenu() {
	menuItems := []components.MenuItem{}
	
	// Create menu items for each configmap
	for _, cm := range m.configMaps {
		age := services.FormatAge(cm.CreationTimestamp.Time)
		
		// Create a formatted title with configmap info
		title := cm.Name
		description := fmt.Sprintf("Keys: %d, Namespace: %s, Age: %s", 
			len(cm.Data), cm.Namespace, age)
		
		menuItems = append(menuItems, components.MenuItem{
			ID:          fmt.Sprintf("%s/%s", cm.Namespace, cm.Name),
			Title:       title,
			Description: description,
			Icon:        "📋",
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
	title := fmt.Sprintf("📋 ConfigMaps (%d items)", len(m.configMaps))
	if m.allNamespaces {
		title += " - All Namespaces"
	} else {
		title += fmt.Sprintf(" - Namespace: %s", services.GetCurrentNamespace())
	}
	
	// Create menu with DevTools style
	m.menu = components.NewDevToolsMenu(title, menuItems)
}

// viewConfigMapDetails shows configmap details
func (m *ConfigMapsViewModel) viewConfigMapDetails(id string) tea.Cmd {
	return func() tea.Msg {
		parts := strings.Split(id, "/")
		if len(parts) != 2 {
			return nil
		}
		
		namespace := parts[0]
		name := parts[1]
		
		// TODO: Implement ConfigMap details view
		fmt.Printf("\nConfigMap: %s (namespace: %s)\n", name, namespace)
		fmt.Println("ConfigMap details view coming soon...")
		fmt.Println("\nPress Enter to continue...")
		fmt.Scanln()
		
		return nil
	}
}