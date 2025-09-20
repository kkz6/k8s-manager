package views

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/karthickk/k8s-manager/internal/ui/components"
)

// ConfigsMenuModel represents the configs submenu
type ConfigsMenuModel struct {
	menu     components.Menu
	quitting bool
	nextView string
}

// ShowConfigsMenu shows the ConfigMaps & Secrets submenu
func ShowConfigsMenu() error {
	menuItems := []components.MenuItem{
		{
			ID:          "configmaps",
			Title:       "ConfigMaps",
			Description: "View and manage Kubernetes ConfigMaps",
			Icon:        "📋",
			Shortcut:    "c",
		},
		{
			ID:          "secrets",
			Title:       "Secrets",
			Description: "View and manage Kubernetes Secrets",
			Icon:        "🔐",
			Shortcut:    "s",
		},
		{
			ID:          "back",
			Title:       "Back to Main Menu",
			Description: "Return to the main menu",
			Icon:        "⬅️",
			Shortcut:    "b",
		},
	}

	menu := components.NewDevToolsMenu("⚙️ ConfigMaps & Secrets", menuItems)
	model := ConfigsMenuModel{
		menu: *menu,
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("error running configs menu: %w", err)
	}

	// Check what was selected
	if menuModel, ok := finalModel.(ConfigsMenuModel); ok {
		switch menuModel.nextView {
		case "configmaps":
			fmt.Print("\033[H\033[2J")
			return ShowConfigMapsView()
		case "secrets":
			fmt.Print("\033[H\033[2J")
			return ShowSecretsView()
		}
	}

	return nil
}

// Init initializes the model
func (m ConfigsMenuModel) Init() tea.Cmd {
	return m.menu.Init()
}

// Update handles updates
func (m ConfigsMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "b":
			// Quick back navigation
			return m, tea.Quit
		case "c":
			// Quick navigation to ConfigMaps
			m.nextView = "configmaps"
			return m, tea.Quit
		case "s":
			// Quick navigation to Secrets
			m.nextView = "secrets"
			return m, tea.Quit
		}
	}

	// Update menu
	newMenu, cmd := m.menu.Update(msg)
	if menu, ok := newMenu.(components.Menu); ok {
		m.menu = menu
		// Check if an item was selected
		if kMsg, ok := msg.(tea.KeyMsg); ok && (kMsg.String() == "enter" || kMsg.String() == " ") {
			selected := m.menu.GetSelected()
			if selected != nil {
				switch selected.ID {
				case "configmaps":
					m.nextView = "configmaps"
					return m, tea.Quit
				case "secrets":
					m.nextView = "secrets"
					return m, tea.Quit
				case "back":
					m.quitting = true
					return m, tea.Quit
				}
			}
		}
	}

	return m, cmd
}

// View renders the view
func (m ConfigsMenuModel) View() string {
	if m.quitting || m.nextView != "" {
		return ""
	}
	return m.menu.View()
}