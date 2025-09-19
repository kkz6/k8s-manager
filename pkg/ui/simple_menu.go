package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

var (
	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("170")).
			Bold(true).
			Background(lipgloss.Color("236"))

	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true).
			Padding(1, 0)

	descStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			PaddingLeft(2)
)

type MenuItem struct {
	Title       string
	Description string
	Command     string
}

type SimpleMenuModel struct {
	items    []MenuItem
	cursor   int
	selected string
	width    int
	height   int
	quitting bool
}

func NewSimpleMenu(items []MenuItem) SimpleMenuModel {
	return SimpleMenuModel{
		items:  items,
		cursor: 0,
	}
}

func (m SimpleMenuModel) Init() tea.Cmd {
	return nil
}

func (m SimpleMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}

		case "home", "g":
			m.cursor = 0

		case "end", "G":
			m.cursor = len(m.items) - 1

		case "enter", " ":
			m.selected = m.items[m.cursor].Command
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m SimpleMenuModel) View() string {
	if m.quitting && m.selected == "" {
		return "\n👋 Goodbye! Thank you for using K8s Manager.\n"
	}
	if m.selected != "" {
		return ""
	}

	var s strings.Builder

	// Header
	s.WriteString(headerStyle.Render("🚀 K8s Manager - Main Menu"))
	s.WriteString("\n")
	s.WriteString(strings.Repeat("─", min(m.width, 80)))
	s.WriteString("\n\n")

	// Calculate how many items can fit on screen
	availableHeight := m.height - 8 // Reserve space for header and help
	if availableHeight <= 0 {
		availableHeight = 10
	}

	// Calculate visible range
	visibleItems := min(len(m.items), availableHeight/2) // 2 lines per item
	startIdx := 0
	endIdx := len(m.items)

	// Scroll view if cursor is outside visible range
	if visibleItems < len(m.items) {
		if m.cursor >= visibleItems {
			startIdx = m.cursor - visibleItems + 1
		}
		endIdx = min(startIdx+visibleItems, len(m.items))
		if endIdx-startIdx < visibleItems {
			startIdx = max(0, endIdx-visibleItems)
		}
	}

	// Show scroll indicator if needed
	if startIdx > 0 {
		s.WriteString(descStyle.Render("  ↑ more above\n"))
	}

	// Menu items
	for i := startIdx; i < endIdx; i++ {
		item := m.items[i]
		cursor := "  "
		if m.cursor == i {
			cursor = "▸ "
		}

		// Title line
		title := fmt.Sprintf("%s%s", cursor, item.Title)
		if m.cursor == i {
			s.WriteString(selectedStyle.Render(title))
		} else {
			s.WriteString(normalStyle.Render(title))
		}
		s.WriteString("\n")

		// Description line (indented)
		if item.Description != "" {
			desc := fmt.Sprintf("    %s", item.Description)
			s.WriteString(descStyle.Render(desc))
			s.WriteString("\n")
		}
	}

	// Show scroll indicator if needed
	if endIdx < len(m.items) {
		s.WriteString(descStyle.Render("  ↓ more below\n"))
	}

	// Help text
	s.WriteString("\n")
	s.WriteString(strings.Repeat("─", min(m.width, 80)))
	s.WriteString("\n")
	helpText := "↑/k: up • ↓/j: down • enter: select • q: quit"
	s.WriteString(descStyle.Render(helpText))

	return s.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ShowSimpleMainMenu displays a simpler, more compact menu
func ShowSimpleMainMenu() string {
	items := []MenuItem{
		{
			Title:       "⚙️  Configuration",
			Description: "Manage K8s Manager configuration",
			Command:     "config",
		},
		{
			Title:       "🔍 Get Pods",
			Description: "List and filter pods in the cluster",
			Command:     "pods",
		},
		{
			Title:       "📝 View Logs",
			Description: "View and follow pod logs",
			Command:     "logs",
		},
		{
			Title:       "⚡ Execute Command",
			Description: "Execute commands in pods",
			Command:     "exec",
		},
		{
			Title:       "🔐 Manage Secrets",
			Description: "View and manage Kubernetes secrets",
			Command:     "secrets",
		},
		{
			Title:       "📊 Resources",
			Description: "View cluster resources and usage",
			Command:     "resources",
		},
		{
			Title:       "🔧 Port Forward",
			Description: "Forward local ports to pods",
			Command:     "port-forward",
		},
		{
			Title:       "❓ Help",
			Description: "Show help and usage information",
			Command:     "help",
		},
		{
			Title:       "🚪 Exit",
			Description: "Exit K8s Manager",
			Command:     "exit",
		},
	}

	// Get terminal size for initial setup
	width, height, _ := term.GetSize(0)
	if width == 0 {
		width = 80
	}
	if height == 0 {
		height = 24
	}

	m := NewSimpleMenu(items)
	m.width = width
	m.height = height

	p := tea.NewProgram(m, tea.WithAltScreen())
	result, err := p.Run()
	if err != nil {
		fmt.Printf("Error running menu: %v\n", err)
		return ""
	}

	if model, ok := result.(SimpleMenuModel); ok {
		return model.selected
	}

	return ""
}