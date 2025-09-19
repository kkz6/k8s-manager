package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("62")).
			Padding(0, 1).
			MarginBottom(1)

	itemStyle = lipgloss.NewStyle().
			PaddingLeft(4)

	selectedItemStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Foreground(lipgloss.Color("170"))

	paginationStyle = list.DefaultStyles().PaginationStyle.
			PaddingLeft(4)

	helpStyle = list.DefaultStyles().HelpStyle.
			PaddingLeft(4).
			PaddingBottom(1)

	quitTextStyle = lipgloss.NewStyle().
			Margin(1, 0, 2, 4)
)

type menuItem struct {
	title       string
	description string
	command     string
}

func (i menuItem) FilterValue() string { return i.title }
func (i menuItem) Title() string       { return i.title }
func (i menuItem) Description() string { return i.description }

type MainMenuModel struct {
	list     list.Model
	choice   string
	quitting bool
}

func (m MainMenuModel) Init() tea.Cmd {
	return nil
}

func (m MainMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Set the list dimensions based on the terminal size
		m.list.SetWidth(msg.Width)
		// Use most of the terminal height, leaving some space for title and help
		availableHeight := msg.Height - 4 // Reserve 4 lines for title and help text
		if availableHeight > 20 {
			availableHeight = 20 // Cap at 20 lines for better UX
		}
		m.list.SetHeight(availableHeight)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			if i, ok := m.list.SelectedItem().(menuItem); ok {
				m.choice = i.command
				m.quitting = true
				return m, tea.Quit
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m MainMenuModel) View() string {
	if m.choice != "" {
		return ""
	}
	if m.quitting {
		return quitTextStyle.Render("👋 Goodbye! Thank you for using K8s Manager.\n")
	}
	return "\n" + m.list.View()
}

// ShowMainMenu displays the interactive main menu and returns the selected command
func ShowMainMenu() string {
	items := []list.Item{
		menuItem{
			title:       "⚙️  Configuration",
			description: "Manage K8s Manager configuration (init, show, set, validate)",
			command:     "config",
		},
		menuItem{
			title:       "🔍 Get Pods",
			description: "List and filter pods in the cluster",
			command:     "pods",
		},
		menuItem{
			title:       "📝 View Logs",
			description: "View and follow pod logs",
			command:     "logs",
		},
		menuItem{
			title:       "⚡ Execute Command",
			description: "Execute commands in pods",
			command:     "exec",
		},
		menuItem{
			title:       "🔐 Manage Secrets",
			description: "View and manage Kubernetes secrets",
			command:     "secrets",
		},
		menuItem{
			title:       "❓ Help",
			description: "Show help and usage information",
			command:     "help",
		},
		menuItem{
			title:       "🚪 Exit",
			description: "Exit K8s Manager",
			command:     "exit",
		},
	}

	// Create a more compact delegate
	delegate := list.NewDefaultDelegate()
	delegate.SetHeight(2) // Make each item take 2 lines (more compact)
	delegate.SetSpacing(0) // No extra spacing between items
	delegate.ShowDescription = true

	// Style the delegate for better visibility
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("170")).
		BorderForeground(lipgloss.Color("170"))
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("244"))

	l := list.New(items, delegate, 0, 0) // Size will be set by WindowSizeMsg
	l.Title = "🚀 K8s Manager - Main Menu"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	m := MainMenuModel{list: l}

	p := tea.NewProgram(m, tea.WithAltScreen())
	result, err := p.Run()
	if err != nil {
		fmt.Printf("Error running menu: %v\n", err)
		return ""
	}

	if model, ok := result.(MainMenuModel); ok {
		return model.choice
	}

	return ""
}

// SubMenuModel represents a submenu with options
type SubMenuModel struct {
	choices  []string
	cursor   int
	selected string
	title    string
}

func (m SubMenuModel) Init() tea.Cmd {
	return nil
}

func (m SubMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.selected = ""
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}

		case "enter", " ":
			m.selected = m.choices[m.cursor]
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m SubMenuModel) View() string {
	s := fmt.Sprintf("\n%s\n\n", titleStyle.Render(m.title))

	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = "▸"
			s += selectedItemStyle.Render(fmt.Sprintf("%s %s", cursor, choice))
		} else {
			s += itemStyle.Render(fmt.Sprintf("%s %s", cursor, choice))
		}
		s += "\n"
	}

	s += "\n" + helpStyle.Render("↑/k up • ↓/j down • enter select • q/esc back")

	return s
}

// ShowSubMenu displays a submenu and returns the selected option
func ShowSubMenu(title string, options []string) string {
	m := SubMenuModel{
		choices: options,
		title:   title,
	}

	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		fmt.Printf("Error running submenu: %v\n", err)
		return ""
	}

	if model, ok := result.(SubMenuModel); ok {
		return model.selected
	}

	return ""
}

// ShowConfigMenu shows the configuration submenu
func ShowConfigMenu() string {
	options := []string{
		"Init - Initialize configuration",
		"Show - Show current configuration",
		"Set - Set configuration value",
		"Validate - Validate configuration",
		"Back - Return to main menu",
	}

	choice := ShowSubMenu("⚙️  Configuration Menu", options)

	switch {
	case strings.HasPrefix(choice, "Init"):
		return "init"
	case strings.HasPrefix(choice, "Show"):
		return "show"
	case strings.HasPrefix(choice, "Set"):
		return "set"
	case strings.HasPrefix(choice, "Validate"):
		return "validate"
	default:
		return ""
	}
}