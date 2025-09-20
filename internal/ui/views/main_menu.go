package views

import (
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
				case "secrets":
					m.nextView = "configs"
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
				case "secrets":
					m.nextView = "configs"
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
	return RunApp()
}