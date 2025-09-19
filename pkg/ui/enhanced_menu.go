package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Custom styles for the enhanced menu
var (
	appStyle = lipgloss.NewStyle().Padding(0, 1)

	titleBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("230")).
			Padding(0, 1).
			MarginBottom(1)

	selectedTitleStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, false, false, true).
				BorderForeground(lipgloss.Color("170")).
				Foreground(lipgloss.Color("170")).
				Bold(true).
				PaddingLeft(1)

	selectedDescStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("245")).
				PaddingLeft(3)

	normalTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				PaddingLeft(2)

	normalDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("246")).
			PaddingLeft(3)

	enhancedPaginationStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	activePageDotStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("170")).
				Bold(true)

	inactivePageDotStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240"))
)

// CompactDelegate is a custom delegate for compact menu items
type CompactDelegate struct {
	ShowDescription bool
	Styles          CompactDelegateStyles
}

type CompactDelegateStyles struct {
	SelectedTitle lipgloss.Style
	SelectedDesc  lipgloss.Style
	NormalTitle   lipgloss.Style
	NormalDesc    lipgloss.Style
}

func NewCompactDelegate() CompactDelegate {
	return CompactDelegate{
		ShowDescription: true,
		Styles: CompactDelegateStyles{
			SelectedTitle: selectedTitleStyle,
			SelectedDesc:  selectedDescStyle,
			NormalTitle:   normalTitleStyle,
			NormalDesc:    normalDescStyle,
		},
	}
}

func (d CompactDelegate) Height() int {
	if d.ShowDescription {
		return 2
	}
	return 1
}

func (d CompactDelegate) Spacing() int {
	return 0
}

func (d CompactDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

func (d CompactDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(menuItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()

	var title, desc string
	if isSelected {
		title = d.Styles.SelectedTitle.Render(i.Title())
		if d.ShowDescription && i.Description() != "" {
			desc = "\n" + d.Styles.SelectedDesc.Render(i.Description())
		}
	} else {
		title = d.Styles.NormalTitle.Render(i.Title())
		if d.ShowDescription && i.Description() != "" {
			desc = "\n" + d.Styles.NormalDesc.Render(i.Description())
		}
	}

	fmt.Fprintf(w, "%s%s", title, desc)
}

// EnhancedMenuModel represents the enhanced menu
type EnhancedMenuModel struct {
	list         list.Model
	width        int
	height       int
	choice       string
	quitting     bool
	initialized  bool
}

func (m EnhancedMenuModel) Init() tea.Cmd {
	return nil
}

func (m EnhancedMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if !m.initialized {
			// Calculate optimal dimensions
			contentWidth := min(100, msg.Width-2)

			// Calculate available height for list items
			// Reserve space for: title (3 lines) + help (2 lines) + padding (2 lines)
			reservedHeight := 7
			availableHeight := msg.Height - reservedHeight

			// Ensure minimum and maximum bounds
			if availableHeight < 10 {
				availableHeight = 10
			} else if availableHeight > 30 {
				availableHeight = 30
			}

			m.list.SetSize(contentWidth, availableHeight)
			m.initialized = true
		} else {
			// Just update width on subsequent resizes
			m.list.SetWidth(min(100, msg.Width-2))

			// Recalculate height
			reservedHeight := 7
			availableHeight := msg.Height - reservedHeight
			if availableHeight < 10 {
				availableHeight = 10
			} else if availableHeight > 30 {
				availableHeight = 30
			}
			m.list.SetHeight(availableHeight)
		}

		return m, nil

	case tea.KeyMsg:
		// Handle quit keys
		switch msg.String() {
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

func (m EnhancedMenuModel) View() string {
	if m.choice != "" {
		return ""
	}
	if m.quitting {
		return "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("170")).Render("👋 Goodbye! Thank you for using K8s Manager.") + "\n\n"
	}

	// Add padding and styling
	return appStyle.Render(m.list.View())
}

// Custom pagination to show dots
func customPaginator(currentPage, totalPages int) string {
	if totalPages <= 1 {
		return ""
	}

	var dots []string
	for i := 0; i < totalPages; i++ {
		if i == currentPage {
			dots = append(dots, activePageDotStyle.Render("●"))
		} else {
			dots = append(dots, inactivePageDotStyle.Render("○"))
		}
	}

	return enhancedPaginationStyle.Render(strings.Join(dots, " "))
}

// ShowEnhancedMainMenu displays the enhanced main menu
func ShowEnhancedMainMenu() string {
	items := []list.Item{
		menuItem{
			title:       "⚙️  Configuration",
			description: "Init, show, set, and validate configuration",
			command:     "config",
		},
		menuItem{
			title:       "🔍 Pods",
			description: "List, filter, and manage pods",
			command:     "pods",
		},
		menuItem{
			title:       "📝 Logs",
			description: "View and follow pod logs",
			command:     "logs",
		},
		menuItem{
			title:       "⚡ Execute",
			description: "Run commands in pods",
			command:     "exec",
		},
		menuItem{
			title:       "🔐 Secrets",
			description: "Manage Kubernetes secrets",
			command:     "secrets",
		},
		menuItem{
			title:       "📊 Resources",
			description: "View cluster resources and usage",
			command:     "resources",
		},
		menuItem{
			title:       "🔧 Port Forward",
			description: "Forward local ports to pods",
			command:     "port-forward",
		},
		menuItem{
			title:       "🚀 Deployments",
			description: "Manage deployments and rollouts",
			command:     "deployments",
		},
		menuItem{
			title:       "🔄 Services",
			description: "Manage Kubernetes services",
			command:     "services",
		},
		menuItem{
			title:       "📦 Namespaces",
			description: "Switch and manage namespaces",
			command:     "namespaces",
		},
		menuItem{
			title:       "❓ Help",
			description: "Show help information",
			command:     "help",
		},
		menuItem{
			title:       "🚪 Exit",
			description: "Exit K8s Manager",
			command:     "exit",
		},
	}

	// Create custom delegate for compact display
	delegate := NewCompactDelegate()

	// Create the list with proper initial size
	l := list.New(items, delegate, 0, 0)

	// Customize the list appearance
	l.Title = "🚀 K8s Manager - Main Menu"
	l.Styles.Title = titleBarStyle
	l.SetStatusBarItemName("option", "options")
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(false)
	l.SetShowPagination(true)
	l.Styles.PaginationStyle = enhancedPaginationStyle

	// Customize help keys display
	l.SetShowHelp(true)
	l.DisableQuitKeybindings() // We'll handle quit ourselves

	// Help styles
	l.Styles.HelpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginTop(1)

	m := EnhancedMenuModel{
		list: l,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	result, err := p.Run()
	if err != nil {
		fmt.Printf("Error running menu: %v\n", err)
		return ""
	}

	if model, ok := result.(EnhancedMenuModel); ok {
		return model.choice
	}

	return ""
}

