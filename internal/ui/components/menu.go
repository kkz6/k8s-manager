package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MenuItem represents a single menu item
type MenuItem struct {
	ID          string
	Title       string
	Description string
	Icon        string
	Shortcut    string
	Action      func() tea.Cmd
}

// Menu is a reusable menu component
type Menu struct {
	Title        string
	Items        []MenuItem
	selected     int
	width        int
	height       int
	showNumbers  bool
	showIcons    bool
	focusedStyle lipgloss.Style
	normalStyle  lipgloss.Style
	style        MenuStyle
}

// MenuStyle represents different menu styles
type MenuStyle int

const (
	DefaultMenuStyle MenuStyle = iota
	DevToolsMenuStyle
)

// NewMenu creates a new menu
func NewMenu(title string, items []MenuItem) *Menu {
	return &Menu{
		Title:        title,
		Items:        items,
		selected:     0,
		showNumbers:  true,
		showIcons:    true,
		focusedStyle: SelectedStyle,
		normalStyle:  ItemStyle,
		style:        DefaultMenuStyle,
	}
}

// NewDevToolsMenu creates a new DevTools-style menu
func NewDevToolsMenu(title string, items []MenuItem) *Menu {
	m := NewMenu(title, items)
	m.style = DevToolsMenuStyle
	// DevTools style - clean and minimalist
	m.focusedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Bold(true)
	m.normalStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))
	return m
}

// Init initializes the menu
func (m Menu) Init() tea.Cmd {
	return nil
}

// Update handles menu navigation
func (m Menu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}

		case "down", "j":
			if m.selected < len(m.Items)-1 {
				m.selected++
			}

		case "home", "g":
			m.selected = 0

		case "end", "G":
			m.selected = len(m.Items) - 1

		case "enter", " ":
			// Just update selection, don't execute action
			return m, nil

		// Handle number shortcuts
		default:
			if len(msg.String()) == 1 && msg.String() >= "1" && msg.String() <= "9" {
				index := int(msg.String()[0] - '1')
				if index < len(m.Items) {
					m.selected = index
					// Don't execute action, just update selection
					return m, nil
				}
			}
		}
	}

	return m, nil
}

// View renders the menu
func (m Menu) View() string {
	if m.style == DevToolsMenuStyle {
		return m.viewDevTools()
	}
	return m.viewDefault()
}

// viewDefault renders the default menu style
func (m Menu) viewDefault() string {
	var b strings.Builder

	// Title
	if m.Title != "" {
		b.WriteString(TitleStyle.Width(60).Align(lipgloss.Center).Render(m.Title))
		b.WriteString("\n\n")
	}

	// Menu items
	for i, item := range m.Items {
		var line strings.Builder

		// Selection indicator
		if i == m.selected {
			line.WriteString("▸ ")
		} else {
			line.WriteString("  ")
		}

		// Number
		if m.showNumbers && i < 9 {
			line.WriteString(NumberStyle.Render(fmt.Sprintf("%d.", i+1)))
			line.WriteString(" ")
		} else {
			line.WriteString("   ")
		}

		// Icon
		if m.showIcons && item.Icon != "" {
			line.WriteString(item.Icon)
			line.WriteString(" ")
		}

		// Title
		titleStr := item.Title
		if item.Shortcut != "" {
			titleStr += fmt.Sprintf(" (%s)", item.Shortcut)
		}

		// Apply style based on selection
		if i == m.selected {
			line.WriteString(m.focusedStyle.Render(titleStr))
		} else {
			line.WriteString(m.normalStyle.Render(titleStr))
		}

		b.WriteString(line.String())
		b.WriteString("\n")

		// Description (only for selected item)
		if i == m.selected && item.Description != "" {
			b.WriteString(strings.Repeat(" ", 6))
			b.WriteString(DescriptionStyle.Render(item.Description))
			b.WriteString("\n")
		}
	}

	// Help
	b.WriteString("\n")
	help := []string{
		RenderKeyBinding("↑/k", "up"),
		RenderKeyBinding("↓/j", "down"),
		RenderKeyBinding("enter", "select"),
		RenderKeyBinding("1-9", "quick select"),
	}
	b.WriteString(HelpStyle.Render(strings.Join(help, " • ")))

	return BoxStyle.Render(b.String())
}

// viewDevTools renders the DevTools-style menu
func (m Menu) viewDevTools() string {
	var b strings.Builder

	// Title with DevTools styling
	if m.Title != "" {
		titleStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("99")).
			Bold(true).
			MarginBottom(1)
		b.WriteString(titleStyle.Render(m.Title))
		b.WriteString("\n\n")
	}

	// Menu items
	for i, item := range m.Items {
		// Number with DevTools styling
		if m.showNumbers && i < 9 {
			numberStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("86")).
				Bold(true)
			b.WriteString(numberStyle.Render(fmt.Sprintf("%d.", i+1)))
		}

		// Title
		titleStr := item.Title
		if i == m.selected {
			titleStr = "▸ " + titleStr
			b.WriteString(m.focusedStyle.Render(titleStr))
		} else {
			titleStr = "  " + titleStr
			b.WriteString(m.normalStyle.Render(titleStr))
		}
		b.WriteString("\n")

		// Description (always shown in DevTools style)
		if item.Description != "" {
			descStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("244")).
				MarginLeft(4)
			b.WriteString(descStyle.Render(item.Description))
			b.WriteString("\n")
		}
	}

	// Help footer with DevTools styling
	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		MarginTop(2)
	helpText := "↑/k up • ↓/j down • 1-9 quick select • enter select • q quit"
	b.WriteString(helpStyle.Render(helpText))

	// Container with padding
	containerStyle := lipgloss.NewStyle().
		Padding(1, 2)

	return containerStyle.Render(b.String())
}

// GetSelected returns the currently selected item
func (m Menu) GetSelected() *MenuItem {
	if m.selected >= 0 && m.selected < len(m.Items) {
		return &m.Items[m.selected]
	}
	return nil
}

// SetSelected sets the selected index
func (m *Menu) SetSelected(index int) {
	if index >= 0 && index < len(m.Items) {
		m.selected = index
	}
}