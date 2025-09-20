package views

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/karthickk/k8s-manager/internal/ui/components"
)

// MainMenuModel handles the main menu navigation
type MainMenuModel struct {
	menu      components.Menu  // Changed from pointer to value
	quitting  bool
	nextView  string
}

// Init initializes the main menu
func (m MainMenuModel) Init() tea.Cmd {
	return nil
}

// Update handles main menu updates
func (m MainMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
	}

	// Update menu
	updatedMenu, cmd := m.menu.Update(msg)
	m.menu = updatedMenu.(components.Menu)

	// Check if enter was pressed and handle selection
	if kMsg, ok := msg.(tea.KeyMsg); ok {
		if kMsg.String() == "enter" || kMsg.String() == " " {
			selected := m.menu.GetSelected()
			if selected != nil {
				switch selected.ID {
				case "pods":
					m.nextView = "pods"
					return m, tea.Quit
				case "quit":
					m.quitting = true
					return m, tea.Quit
				default:
					// For unimplemented features, just return
					return m, nil
				}
			}
		}
		// Handle number key selection
		if len(kMsg.String()) == 1 && kMsg.String() >= "1" && kMsg.String() <= "9" {
			index := int(kMsg.String()[0] - '1')
			if index < len(m.menu.Items) {
				switch m.menu.Items[index].ID {
				case "pods":
					m.nextView = "pods"
					return m, tea.Quit
				case "quit":
					m.quitting = true
					return m, tea.Quit
				default:
					return m, nil
				}
			}
		}
	}

	return m, cmd
}

// View renders the main menu
func (m MainMenuModel) View() string {
	if m.quitting || m.nextView != "" {
		return ""
	}
	return m.menu.View()
}

// ShowMainMenu displays the main interactive menu
func ShowMainMenu() error {
	// Show splash screen first
	logo := components.NewLogo("K8S MANAGER")
	logo.Subtitle = "Your Kubernetes cluster management toolkit"
	logo.Author = "Karthick"
	logo.Email = "karthick@example.com"
	logo.Website = "k8s-manager.dev"

	splash := components.NewSplashScreen(logo, 1500*time.Millisecond)
	if p := tea.NewProgram(splash, tea.WithAltScreen()); p != nil {
		p.Run()
		fmt.Print("\033[H\033[2J") // Clear screen
	}

	// Main menu loop
	for {
		// Create menu items
		menuItems := []components.MenuItem{
			{
				ID:          "pods",
				Title:       "Pods Manager",
				Description: "List, manage, and interact with Kubernetes pods",
			},
			{
				ID:          "deployments",
				Title:       "Deployments",
				Description: "Manage Kubernetes deployments and rollouts",
			},
			{
				ID:          "services",
				Title:       "Services",
				Description: "View and manage Kubernetes services",
			},
			{
				ID:          "secrets",
				Title:       "ConfigMaps & Secrets",
				Description: "Manage configuration and secret resources",
			},
			{
				ID:          "namespaces",
				Title:       "Namespaces",
				Description: "Switch and manage Kubernetes namespaces",
			},
			{
				ID:          "nodes",
				Title:       "Nodes & Cluster",
				Description: "View cluster nodes and resource usage",
			},
			{
				ID:          "logs",
				Title:       "Logs & Events",
				Description: "View pod logs and cluster events",
			},
			{
				ID:          "config",
				Title:       "Configuration",
				Description: "Manage K8s Manager settings and contexts",
			},
			{
				ID:          "quit",
				Title:       "Exit",
				Description: "Quit the application",
			},
		}

		menu := components.NewDevToolsMenu("🚀 K8s Manager by Karthick", menuItems)
		model := MainMenuModel{
			menu: *menu,  // Dereference the pointer
		}
		
		p := tea.NewProgram(model, tea.WithAltScreen())
		finalModel, err := p.Run()
		if err != nil {
			return fmt.Errorf("error running main menu: %w", err)
		}

		// Check what was selected
		if menuModel, ok := finalModel.(MainMenuModel); ok {
			if menuModel.quitting {
				return nil
			}
			if menuModel.nextView == "pods" {
				// Clear screen and show pods view
				fmt.Print("\033[H\033[2J")
				if err := ShowPodsView(PodsOptions{}); err != nil {
					return err
				}
				// Continue the loop to show menu again
				continue
			}
		}
	}
}