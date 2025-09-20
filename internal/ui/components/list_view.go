package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ListItem represents an item in the list
type ListItem struct {
	ID          string
	Title       string
	Description string
	Icon        string
	Data        interface{} // Store any additional data
}

// ListView is an improved list component without numbering
type ListView struct {
	Title        string
	Items        []ListItem
	selected     int
	focusedStyle lipgloss.Style
	normalStyle  lipgloss.Style
	helpText     string
	showIcons    bool
}

// NewListView creates a new list view
func NewListView(title string, items []ListItem) *ListView {
	return &ListView{
		Title:        title,
		Items:        items,
		selected:     0,
		focusedStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true),
		normalStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
		showIcons:    true,
	}
}

// SetHelpText sets the help text shown at the bottom
func (l *ListView) SetHelpText(text string) {
	l.helpText = text
}

// Init initializes the list view
func (l ListView) Init() tea.Cmd {
	return nil
}

// Update handles list view updates
func (l ListView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if l.selected > 0 {
				l.selected--
			} else {
				l.selected = len(l.Items) - 1 // Wrap to bottom
			}
		case "down", "j":
			if l.selected < len(l.Items)-1 {
				l.selected++
			} else {
				l.selected = 0 // Wrap to top
			}
		case "g", "home":
			l.selected = 0
		case "G", "end":
			l.selected = len(l.Items) - 1
		}
	}
	return l, nil
}

// View renders the list view
func (l ListView) View() string {
	if len(l.Items) == 0 {
		return "No items available"
	}

	var b strings.Builder

	// Title
	if l.Title != "" {
		titleStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("99")).
			Bold(true).
			MarginBottom(1)
		b.WriteString(titleStyle.Render(l.Title))
		b.WriteString("\n\n")
	}

	// Items
	for i, item := range l.Items {
		// Selection indicator
		if i == l.selected {
			b.WriteString(l.focusedStyle.Render("▸ "))
		} else {
			b.WriteString("  ")
		}

		// Icon
		if l.showIcons && item.Icon != "" {
			b.WriteString(item.Icon + " ")
		}

		// Title
		if i == l.selected {
			b.WriteString(l.focusedStyle.Render(item.Title))
		} else {
			b.WriteString(l.normalStyle.Render(item.Title))
		}

		// Description on the same line if short enough
		if item.Description != "" && len(item.Description) < 50 {
			descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
			b.WriteString(descStyle.Render(" - " + item.Description))
		}
		
		b.WriteString("\n")

		// Multi-line description
		if item.Description != "" && len(item.Description) >= 50 {
			descStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("244")).
				MarginLeft(4)
			b.WriteString(descStyle.Render(item.Description))
			b.WriteString("\n")
		}
	}

	// Help footer
	if l.helpText != "" {
		b.WriteString("\n")
		helpStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Border(lipgloss.NormalBorder(), true, false, false, false).
			BorderForeground(lipgloss.Color("240")).
			Padding(1, 0).
			MarginTop(1)
		b.WriteString(helpStyle.Render(l.helpText))
	}

	return b.String()
}

// GetSelected returns the currently selected item
func (l *ListView) GetSelected() *ListItem {
	if l.selected >= 0 && l.selected < len(l.Items) {
		return &l.Items[l.selected]
	}
	return nil
}

// SetSelected sets the selected index
func (l *ListView) SetSelected(index int) {
	if index >= 0 && index < len(l.Items) {
		l.selected = index
	}
}

// GetSelectedIndex returns the current selection index
func (l *ListView) GetSelectedIndex() int {
	return l.selected
}