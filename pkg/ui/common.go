package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Common styles for the entire application
var (
	// Main container styles
	AppStyle = lipgloss.NewStyle().
		Margin(1, 2).
		Padding(1, 2)

	// Title and header styles
	TitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		Background(lipgloss.Color("235")).
		Padding(0, 2).
		MarginBottom(1)

	HeaderStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		BorderStyle(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 2).
		MarginBottom(1)

	// List and menu styles
	ListStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2)

	ItemStyle = lipgloss.NewStyle().
		PaddingLeft(2)

	SelectedItemStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("170")).
		Background(lipgloss.Color("235")).
		Bold(true).
		PaddingLeft(1).
		PaddingRight(1)

	NumberStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Width(4)

	// Status styles
	StatusRunningStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("42")).
		Bold(true)

	StatusPendingStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Bold(true)

	StatusErrorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)

	StatusInfoStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("86"))

	// Help and footer styles
	HelpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderForeground(lipgloss.Color("237")).
		MarginTop(1).
		Padding(1, 2)

	FooterStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginTop(1).
		Align(lipgloss.Center)

	// Message styles
	SuccessMessageStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("42")).
		Bold(true).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("42")).
		Padding(0, 2).
		MarginTop(1).
		MarginBottom(1)

	ErrorMessageStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).
		Padding(0, 2).
		MarginTop(1).
		MarginBottom(1)

	InfoMessageStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("86")).
		Padding(0, 2).
		MarginTop(1).
		MarginBottom(1)

	// Box styles for content areas
	ContentBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		MarginBottom(1)

	// Spinner style
	SpinnerStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("205"))
)

// MenuItem represents a menu item with keyboard navigation
type MenuItem struct {
	Title       string
	Description string
	Icon        string
	Number      int
	Handler     func() tea.Cmd
	SubItems    []MenuItem
}

// ListItem represents a generic list item
type ListItem struct {
	ID          string
	Title       string
	Description string
	Status      string
	Details     map[string]string
	Icon        string
}

// NavigationKeys defines key bindings for navigation
type NavigationKeys struct {
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	Enter    key.Binding
	Back     key.Binding
	Quit     key.Binding
	Search   key.Binding
	Refresh  key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Home     key.Binding
	End      key.Binding
	Number   key.Binding
	Help     key.Binding
}

// DefaultNavigationKeys returns default key bindings
func DefaultNavigationKeys() NavigationKeys {
	return NavigationKeys{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "left"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "right"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter", " "),
			key.WithHelp("enter/space", "select"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc", "backspace"),
			key.WithHelp("esc/backspace", "back"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q/ctrl+c", "quit"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r", "f5"),
			key.WithHelp("r/f5", "refresh"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "b"),
			key.WithHelp("pgup/b", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "f"),
			key.WithHelp("pgdn/f", "page down"),
		),
		Home: key.NewBinding(
			key.WithKeys("home", "g"),
			key.WithHelp("home/g", "first"),
		),
		End: key.NewBinding(
			key.WithKeys("end", "G"),
			key.WithHelp("end/G", "last"),
		),
		Number: key.NewBinding(
			key.WithKeys("1", "2", "3", "4", "5", "6", "7", "8", "9"),
			key.WithHelp("1-9", "quick select"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
	}
}

// List represents a scrollable list with enhanced navigation
type List struct {
	Items        []ListItem
	MenuItems    []MenuItem
	cursor       int
	offset       int
	height       int
	width        int
	showNumbers  bool
	multiSelect  bool
	selected     map[int]bool
	searchMode   bool
	searchQuery  string
	keys         NavigationKeys
}

// NewList creates a new list
func NewList(items []ListItem, showNumbers bool) *List {
	return &List{
		Items:       items,
		cursor:      0,
		offset:      0,
		height:      20,
		width:       80,
		showNumbers: showNumbers,
		selected:    make(map[int]bool),
		keys:        DefaultNavigationKeys(),
	}
}

// NewMenu creates a new menu from menu items
func NewMenu(items []MenuItem) *List {
	return &List{
		MenuItems:   items,
		cursor:      0,
		offset:      0,
		height:      20,
		width:       80,
		showNumbers: true,
		selected:    make(map[int]bool),
		keys:        DefaultNavigationKeys(),
	}
}

// Update handles list updates
func (l *List) Update(msg tea.Msg) (*List, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle number navigation
		if msg.String() >= "1" && msg.String() <= "9" {
			num := int(msg.String()[0] - '0')
			if l.MenuItems != nil {
				if num <= len(l.MenuItems) {
					l.cursor = num - 1
					if l.MenuItems[l.cursor].Handler != nil {
						return l, l.MenuItems[l.cursor].Handler()
					}
				}
			} else if l.Items != nil {
				if num <= len(l.Items) {
					l.cursor = num - 1
				}
			}
			return l, nil
		}

		switch {
		case key.Matches(msg, l.keys.Up):
			l.MoveUp()
		case key.Matches(msg, l.keys.Down):
			l.MoveDown()
		case key.Matches(msg, l.keys.PageUp):
			l.PageUp()
		case key.Matches(msg, l.keys.PageDown):
			l.PageDown()
		case key.Matches(msg, l.keys.Home):
			l.MoveToStart()
		case key.Matches(msg, l.keys.End):
			l.MoveToEnd()
		case key.Matches(msg, l.keys.Enter):
			if l.multiSelect {
				l.ToggleSelection()
			} else if l.MenuItems != nil && l.cursor < len(l.MenuItems) {
				if l.MenuItems[l.cursor].Handler != nil {
					return l, l.MenuItems[l.cursor].Handler()
				}
			}
			return l, nil
		case key.Matches(msg, l.keys.Search):
			l.searchMode = !l.searchMode
			l.searchQuery = ""
		case key.Matches(msg, l.keys.Back):
			if l.searchMode {
				l.searchMode = false
				l.searchQuery = ""
			}
		default:
			if l.searchMode {
				l.searchQuery += msg.String()
				l.FilterItems(l.searchQuery)
			}
		}
	}
	return l, nil
}

// MoveUp moves cursor up
func (l *List) MoveUp() {
	if l.cursor > 0 {
		l.cursor--
		if l.cursor < l.offset {
			l.offset = l.cursor
		}
	}
}

// MoveDown moves cursor down
func (l *List) MoveDown() {
	maxItems := len(l.Items)
	if l.MenuItems != nil {
		maxItems = len(l.MenuItems)
	}

	if l.cursor < maxItems-1 {
		l.cursor++
		if l.cursor >= l.offset+l.height {
			l.offset++
		}
	}
}

// PageUp moves up by page
func (l *List) PageUp() {
	l.cursor -= l.height
	if l.cursor < 0 {
		l.cursor = 0
	}
	l.offset = l.cursor
}

// PageDown moves down by page
func (l *List) PageDown() {
	maxItems := len(l.Items)
	if l.MenuItems != nil {
		maxItems = len(l.MenuItems)
	}

	l.cursor += l.height
	if l.cursor >= maxItems {
		l.cursor = maxItems - 1
	}
	if l.cursor >= l.offset+l.height {
		l.offset = l.cursor - l.height + 1
	}
}

// MoveToStart moves to the first item
func (l *List) MoveToStart() {
	l.cursor = 0
	l.offset = 0
}

// MoveToEnd moves to the last item
func (l *List) MoveToEnd() {
	if l.MenuItems != nil {
		l.cursor = len(l.MenuItems) - 1
	} else {
		l.cursor = len(l.Items) - 1
	}
	if l.cursor >= l.height {
		l.offset = l.cursor - l.height + 1
	}
}

// ToggleSelection toggles selection for multi-select lists
func (l *List) ToggleSelection() {
	if l.multiSelect {
		l.selected[l.cursor] = !l.selected[l.cursor]
	}
}

// FilterItems filters items based on search query
func (l *List) FilterItems(query string) {
	// Implementation would filter l.Items based on query
	// This is a placeholder for the actual filtering logic
}

// View renders the list
func (l *List) View() string {
	var s strings.Builder

	// Determine what to render
	var items []string
	maxItems := 0

	if l.MenuItems != nil {
		maxItems = len(l.MenuItems)
		for i := l.offset; i < maxItems && i < l.offset+l.height; i++ {
			item := l.MenuItems[i]
			line := l.renderMenuItem(item, i, i == l.cursor)
			items = append(items, line)
		}
	} else {
		maxItems = len(l.Items)
		for i := l.offset; i < maxItems && i < l.offset+l.height; i++ {
			item := l.Items[i]
			line := l.renderListItem(item, i, i == l.cursor)
			items = append(items, line)
		}
	}

	// Join all items
	content := strings.Join(items, "\n")

	// Add scroll indicators
	if l.offset > 0 {
		s.WriteString("▲ More above\n")
	}

	s.WriteString(content)

	if l.offset+l.height < maxItems {
		s.WriteString("\n▼ More below")
	}

	// Add search mode indicator
	if l.searchMode {
		s.WriteString(fmt.Sprintf("\n\n🔍 Search: %s", l.searchQuery))
	}

	return ListStyle.Render(s.String())
}

// renderMenuItem renders a single menu item
func (l *List) renderMenuItem(item MenuItem, index int, selected bool) string {
	var s strings.Builder

	// Number prefix for quick navigation
	if l.showNumbers && item.Number > 0 && item.Number <= 9 {
		s.WriteString(NumberStyle.Render(fmt.Sprintf("[%d]", item.Number)))
	} else if l.showNumbers && index < 9 {
		s.WriteString(NumberStyle.Render(fmt.Sprintf("[%d]", index+1)))
	} else {
		s.WriteString(NumberStyle.Render("   "))
	}

	// Selection indicator
	if selected {
		s.WriteString(" ▸ ")
	} else {
		s.WriteString("   ")
	}

	// Icon
	if item.Icon != "" {
		s.WriteString(item.Icon + " ")
	}

	// Title and description
	title := item.Title
	if item.Description != "" {
		title += " - " + lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(item.Description)
	}

	if selected {
		return SelectedItemStyle.Render(s.String() + title)
	}
	return ItemStyle.Render(s.String() + title)
}

// renderListItem renders a single list item
func (l *List) renderListItem(item ListItem, index int, selected bool) string {
	var s strings.Builder

	// Number prefix for quick navigation
	if l.showNumbers && index < 9 {
		s.WriteString(NumberStyle.Render(fmt.Sprintf("[%d]", index+1)))
	} else {
		s.WriteString(NumberStyle.Render("   "))
	}

	// Multi-select checkbox
	if l.multiSelect {
		if l.selected[index] {
			s.WriteString(" ☑ ")
		} else {
			s.WriteString(" ☐ ")
		}
	}

	// Selection indicator
	if selected {
		s.WriteString(" ▸ ")
	} else {
		s.WriteString("   ")
	}

	// Icon
	if item.Icon != "" {
		s.WriteString(item.Icon + " ")
	}

	// Title
	s.WriteString(item.Title)

	// Status with color
	if item.Status != "" {
		statusStr := " [" + item.Status + "]"
		switch strings.ToLower(item.Status) {
		case "running", "ready", "success":
			statusStr = StatusRunningStyle.Render(statusStr)
		case "pending", "waiting":
			statusStr = StatusPendingStyle.Render(statusStr)
		case "error", "failed", "crashloopbackoff":
			statusStr = StatusErrorStyle.Render(statusStr)
		default:
			statusStr = StatusInfoStyle.Render(statusStr)
		}
		s.WriteString(statusStr)
	}

	// Description
	if item.Description != "" {
		s.WriteString("\n    " + lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(item.Description))
	}

	// Details
	if len(item.Details) > 0 {
		for key, value := range item.Details {
			s.WriteString(fmt.Sprintf("\n    %s: %s", key, value))
		}
	}

	if selected {
		return SelectedItemStyle.Width(l.width - 4).Render(s.String())
	}
	return ItemStyle.Render(s.String())
}

// GetSelected returns selected items for multi-select lists
func (l *List) GetSelected() []int {
	var selected []int
	for i, sel := range l.selected {
		if sel {
			selected = append(selected, i)
		}
	}
	return selected
}

// GetCursor returns the current cursor position
func (l *List) GetCursor() int {
	return l.cursor
}

// SetSize sets the display size of the list
func (l *List) SetSize(width, height int) {
	l.width = width
	l.height = height
}

// LoadingSpinner creates a loading spinner
func LoadingSpinner(message string) spinner.Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = SpinnerStyle
	return s
}

// RenderHelp renders help text based on key bindings
func RenderHelp(keys NavigationKeys, additional ...string) string {
	helps := []string{
		keys.Up.Help().Key + " " + keys.Up.Help().Desc,
		keys.Down.Help().Key + " " + keys.Down.Help().Desc,
		keys.Enter.Help().Key + " " + keys.Enter.Help().Desc,
		keys.Back.Help().Key + " " + keys.Back.Help().Desc,
		keys.Search.Help().Key + " " + keys.Search.Help().Desc,
		keys.Number.Help().Key + " " + keys.Number.Help().Desc,
		keys.Quit.Help().Key + " " + keys.Quit.Help().Desc,
	}

	helps = append(helps, additional...)

	return HelpStyle.Render(strings.Join(helps, " • "))
}

// RenderTitle renders a styled title
func RenderTitle(title string, subtitle ...string) string {
	var s strings.Builder
	s.WriteString(TitleStyle.Width(80).Align(lipgloss.Center).Render(title))

	if len(subtitle) > 0 && subtitle[0] != "" {
		s.WriteString("\n")
		s.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Width(80).
			Align(lipgloss.Center).
			Render(subtitle[0]))
	}

	return s.String()
}

// RenderMessage renders a message with appropriate styling
func RenderMessage(messageType, message string) string {
	switch messageType {
	case "success":
		return SuccessMessageStyle.Render("✅ " + message)
	case "error":
		return ErrorMessageStyle.Render("❌ " + message)
	case "info":
		return InfoMessageStyle.Render("ℹ️  " + message)
	default:
		return InfoMessageStyle.Render(message)
	}
}