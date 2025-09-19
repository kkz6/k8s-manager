package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	// DevTools style - clean and minimalist
	devToolsTitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("99")).
		Bold(true).
		MarginBottom(1)

	devToolsNumberStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Bold(true)

	devToolsItemStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	devToolsDescriptionStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		MarginLeft(4)

	devToolsSelectedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Bold(true)

	devToolsHelpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		MarginTop(2)

	devToolsContainerStyle = lipgloss.NewStyle().
		Padding(1, 2)
)

// DevToolsMenuItem represents a menu item in DevTools style
type DevToolsMenuItem struct {
	Number      string
	Title       string
	Description string
}

// DevToolsMenu represents the DevTools-style menu
type DevToolsMenu struct {
	title    string
	items    []DevToolsMenuItem
	selected int
	width    int
	height   int
	quitting bool
}

// NewDevToolsMenu creates a new DevTools-style menu
func NewDevToolsMenu(title string, items []DevToolsMenuItem) *DevToolsMenu {
	return &DevToolsMenu{
		title:    title,
		items:    items,
		selected: -1,
	}
}

func (m *DevToolsMenu) Init() tea.Cmd {
	return nil
}

func (m *DevToolsMenu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		keyStr := msg.String()

		// Handle number keys for instant selection
		if len(keyStr) == 1 && keyStr[0] >= '0' && keyStr[0] <= '9' {
			if keyStr[0] == '0' {
				// 0 is typically used for back/exit
				if len(m.items) > 9 {
					m.selected = 9 // Select the 10th item (index 9)
				} else {
					m.selected = len(m.items) - 1 // Select last item
				}
				return m, tea.Quit
			}

			num := int(keyStr[0] - '0')
			if num <= len(m.items) {
				m.selected = num - 1
				// Immediately quit with selection
				return m, tea.Quit
			}
		}

		// Handle arrow navigation
		switch keyStr {
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			} else if m.selected == -1 && len(m.items) > 0 {
				m.selected = len(m.items) - 1
			}

		case "down", "j":
			if m.selected < len(m.items)-1 {
				m.selected++
			} else if m.selected == -1 && len(m.items) > 0 {
				m.selected = 0
			}

		case "enter", " ":
			if m.selected >= 0 && m.selected < len(m.items) {
				// Quit with selection
				return m, tea.Quit
			} else if len(m.items) > 0 {
				// If nothing selected, select first item
				m.selected = 0
				return m, tea.Quit
			}

		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "esc":
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m *DevToolsMenu) View() string {
	if m.quitting {
		return ""
	}

	var s strings.Builder

	// Title
	s.WriteString(devToolsTitleStyle.Render(m.title))
	s.WriteString("\n\n")

	// Menu items
	for i, item := range m.items {
		// Number
		numberStr := devToolsNumberStyle.Render(item.Number + ".")

		// Title
		titleStr := item.Title
		if i == m.selected {
			titleStr = devToolsSelectedStyle.Render("▸ " + titleStr)
		} else {
			titleStr = "  " + devToolsItemStyle.Render(titleStr)
		}

		// Combine number and title
		s.WriteString(numberStr + titleStr)
		s.WriteString("\n")

		// Description (indented)
		if item.Description != "" {
			s.WriteString(devToolsDescriptionStyle.Render(item.Description))
			s.WriteString("\n")
		}
	}

	// Help footer
	s.WriteString("\n")
	helpText := "↑/k up • ↓/j down • 1-9 quick select • enter select • q quit"
	s.WriteString(devToolsHelpStyle.Render(helpText))

	return devToolsContainerStyle.Render(s.String())
}

// K8sManagerMenu creates the main menu for K8s Manager in DevTools style
func K8sManagerMenu() *DevToolsMenu {
	items := []DevToolsMenuItem{
		{
			Number:      "1",
			Title:       "Pods Manager",
			Description: "List, manage, and interact with Kubernetes pods",
		},
		{
			Number:      "2",
			Title:       "Deployments",
			Description: "Manage Kubernetes deployments and rollouts",
		},
		{
			Number:      "3",
			Title:       "Services",
			Description: "View and manage Kubernetes services",
		},
		{
			Number:      "4",
			Title:       "ConfigMaps & Secrets",
			Description: "Manage configuration and secret resources",
		},
		{
			Number:      "5",
			Title:       "Namespaces",
			Description: "Switch and manage Kubernetes namespaces",
		},
		{
			Number:      "6",
			Title:       "Nodes & Cluster",
			Description: "View cluster nodes and resource usage",
		},
		{
			Number:      "7",
			Title:       "Logs & Events",
			Description: "View pod logs and cluster events",
		},
		{
			Number:      "8",
			Title:       "Configuration",
			Description: "Manage K8s Manager settings and contexts",
		},
		{
			Number:      "9",
			Title:       "Exit",
			Description: "Quit the application",
		},
	}

	return NewDevToolsMenu("🚀 K8s Manager by Karthick", items)
}