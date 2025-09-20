package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TableColumn defines a table column
type TableColumn struct {
	Title string
	Width int
}

// TableRow represents a row of data
type TableRow []string

// Table is a reusable table component
type Table struct {
	Title       string
	Columns     []TableColumn
	Rows        []TableRow
	selected    int
	showNumbers bool
	selectable  bool
	height      int
	offset      int
	onSelect    func(row TableRow) tea.Cmd
}

// NewTable creates a new table
func NewTable(title string, columns []TableColumn) *Table {
	return &Table{
		Title:       title,
		Columns:     columns,
		Rows:        []TableRow{},
		selected:    0,
		showNumbers: true,
		selectable:  true,
		height:      20,
		offset:      0,
	}
}

// SetRows sets the table rows
func (t *Table) SetRows(rows []TableRow) {
	t.Rows = rows
	if t.selected >= len(rows) {
		t.selected = len(rows) - 1
	}
	if t.selected < 0 {
		t.selected = 0
	}
}

// SetOnSelect sets the selection handler
func (t *Table) SetOnSelect(handler func(row TableRow) tea.Cmd) {
	t.onSelect = handler
}

// Init initializes the table (required for tea.Model)
func (t *Table) Init() tea.Cmd {
	return nil
}

// Update handles table navigation
func (t *Table) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		t.height = msg.Height - 10 // Leave room for title and help

	case tea.KeyMsg:
		if !t.selectable {
			return t, nil
		}

		switch msg.String() {
		case "up", "k":
			if t.selected > 0 {
				t.selected--
				t.ensureVisible()
			}

		case "down", "j":
			if t.selected < len(t.Rows)-1 {
				t.selected++
				t.ensureVisible()
			}

		case "pgup":
			t.selected -= t.height / 2
			if t.selected < 0 {
				t.selected = 0
			}
			t.ensureVisible()

		case "pgdown":
			t.selected += t.height / 2
			if t.selected >= len(t.Rows) {
				t.selected = len(t.Rows) - 1
			}
			t.ensureVisible()

		case "home", "g":
			t.selected = 0
			t.offset = 0

		case "end", "G":
			t.selected = len(t.Rows) - 1
			t.ensureVisible()

		case "enter", " ":
			if t.onSelect != nil && t.selected < len(t.Rows) {
				return t, t.onSelect(t.Rows[t.selected])
			}

		// Number shortcuts
		default:
			if len(msg.String()) == 1 && msg.String() >= "1" && msg.String() <= "9" {
				index := int(msg.String()[0] - '1')
				if index < len(t.Rows) {
					t.selected = index
					t.ensureVisible()
					if t.onSelect != nil {
						return t, t.onSelect(t.Rows[t.selected])
					}
				}
			}
		}
	}

	return t, nil
}

// View renders the table
func (t Table) View() string {
	var b strings.Builder

	// Title
	if t.Title != "" {
		b.WriteString(TitleStyle.Render(t.Title))
		b.WriteString("\n\n")
	}

	// If no rows, show empty message
	if len(t.Rows) == 0 {
		b.WriteString(DescriptionStyle.Render("No items found"))
		b.WriteString("\n\n")
		b.WriteString(HelpStyle.Render("Press 'r' to refresh, 'q' to go back"))
		return BoxStyle.Render(b.String())
	}

	// Build header row
	headers := []string{}
	if t.showNumbers {
		headers = append(headers, "#")
	}
	for _, col := range t.Columns {
		headers = append(headers, col.Title)
	}

	// Calculate column widths
	colWidths := []int{}
	if t.showNumbers {
		colWidths = append(colWidths, 4)
	}
	for _, col := range t.Columns {
		colWidths = append(colWidths, col.Width)
	}

	// Render header
	headerLine := t.renderRow(headers, colWidths, HeaderStyle)
	b.WriteString(headerLine)
	b.WriteString("\n")

	// Separator
	b.WriteString(t.renderSeparator(colWidths))
	b.WriteString("\n")

	// Rows
	visibleRows := t.getVisibleRows()
	for i, row := range visibleRows {
		actualIndex := t.offset + i
		
		// Build row data
		rowData := []string{}
		if t.showNumbers {
			rowData = append(rowData, fmt.Sprintf("%d", actualIndex+1))
		}
		for j, cell := range row {
			if j < len(t.Columns) {
				rowData = append(rowData, cell)
			}
		}

		// Render row with selection
		var rowLine string
		if t.selectable && actualIndex == t.selected {
			rowLine = SelectedStyle.Render("▸ ")
			rowLine += t.renderRow(rowData, colWidths, SelectedStyle)
		} else {
			rowLine = "  "
			rowLine += t.renderRow(rowData, colWidths, ItemStyle)
		}

		b.WriteString(rowLine)
		if i < len(visibleRows)-1 {
			b.WriteString("\n")
		}
	}

	// Scroll indicator
	if len(t.Rows) > t.height {
		b.WriteString("\n\n")
		scrollInfo := fmt.Sprintf("Showing %d-%d of %d", 
			t.offset+1, 
			min(t.offset+t.height, len(t.Rows)), 
			len(t.Rows))
		b.WriteString(DescriptionStyle.Render(scrollInfo))
	}

	// Help
	b.WriteString("\n\n")
	help := []string{
		RenderKeyBinding("↑/k", "up"),
		RenderKeyBinding("↓/j", "down"),
		RenderKeyBinding("enter", "select"),
		RenderKeyBinding("g/G", "top/bottom"),
	}
	b.WriteString(HelpStyle.Render(strings.Join(help, " • ")))

	return BoxStyle.Render(b.String())
}

// ensureVisible ensures the selected row is visible
func (t *Table) ensureVisible() {
	if t.selected < t.offset {
		t.offset = t.selected
	} else if t.selected >= t.offset+t.height {
		t.offset = t.selected - t.height + 1
	}

	if t.offset < 0 {
		t.offset = 0
	}
}

// getVisibleRows returns the currently visible rows
func (t Table) getVisibleRows() []TableRow {
	start := t.offset
	end := min(t.offset+t.height, len(t.Rows))
	if start >= len(t.Rows) {
		return []TableRow{}
	}
	return t.Rows[start:end]
}

// GetSelected returns the currently selected row
func (t Table) GetSelected() TableRow {
	if t.selected >= 0 && t.selected < len(t.Rows) {
		return t.Rows[t.selected]
	}
	return nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// renderRow renders a row with proper column alignment
func (t Table) renderRow(cells []string, widths []int, style lipgloss.Style) string {
	var row strings.Builder
	for i, cell := range cells {
		if i < len(widths) {
			cellStr := t.padOrTruncate(cell, widths[i])
			row.WriteString(style.Width(widths[i]).Render(cellStr))
			if i < len(cells)-1 {
				row.WriteString(" ")
			}
		}
	}
	return row.String()
}

// renderSeparator renders the separator line
func (t Table) renderSeparator(widths []int) string {
	totalWidth := 0
	for i, w := range widths {
		totalWidth += w
		if i < len(widths)-1 {
			totalWidth += 1 // space between columns
		}
	}
	return strings.Repeat("─", totalWidth)
}

// padOrTruncate pads or truncates a string to fit the given width
func (t Table) padOrTruncate(s string, width int) string {
	if len(s) > width {
		if width > 3 {
			return s[:width-3] + "..."
		}
		return s[:width]
	}
	return s
}