package components

import (
	"fmt"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TableColumn defines a table column
type TableColumn struct {
	Title string
	Width int
	Align string // "left", "right", "center"
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
	// Set default alignment if not specified
	for i := range columns {
		if columns[i].Align == "" {
			columns[i].Align = "left"
		}
	}

	return &Table{
		Title:       title,
		Columns:     columns,
		Rows:        []TableRow{},
		selected:    0,
		showNumbers: false, // Disabled by default for cleaner look
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
		}
	}

	return t, nil
}

// View renders the table
func (t Table) View() string {
	var b strings.Builder

	// Title
	if t.Title != "" {
		titleStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			Padding(0, 1)
		b.WriteString(titleStyle.Render("📊 " + t.Title))
		b.WriteString("\n\n")
	}

	// If no rows, show empty message
	if len(t.Rows) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(ColorMuted).
			Italic(true).
			Padding(2, 4)
		b.WriteString(emptyStyle.Render("No items found"))
		return b.String()
	}

	// Calculate actual column widths needed
	colWidths := t.calculateColumnWidths()

	// Render header
	b.WriteString(t.renderHeader(colWidths))
	b.WriteString("\n")

	// Separator line
	b.WriteString(t.renderSeparator(colWidths))
	b.WriteString("\n")

	// Rows
	visibleRows := t.getVisibleRows()
	for i, row := range visibleRows {
		actualIndex := t.offset + i
		b.WriteString(t.renderRow(row, colWidths, actualIndex))
		if i < len(visibleRows)-1 {
			b.WriteString("\n")
		}
	}

	// Scroll indicator
	if len(t.Rows) > t.height {
		b.WriteString("\n")
		scrollStyle := lipgloss.NewStyle().
			Foreground(ColorMuted).
			Italic(true).
			Padding(1, 0)
		scrollInfo := fmt.Sprintf("Rows %d-%d of %d",
			t.offset+1,
			min(t.offset+len(visibleRows), len(t.Rows)),
			len(t.Rows))
		b.WriteString(scrollStyle.Render(scrollInfo))
	}

	return b.String()
}

// calculateColumnWidths calculates the actual widths needed for each column
func (t Table) calculateColumnWidths() []int {
	widths := make([]int, len(t.Columns))

	// Start with specified widths
	for i, col := range t.Columns {
		widths[i] = col.Width
		// Make sure at least as wide as header
		headerLen := utf8.RuneCountInString(col.Title)
		if headerLen > widths[i] {
			widths[i] = headerLen
		}
	}

	// Check actual data widths (sample first 100 rows for performance)
	sampleSize := min(100, len(t.Rows))
	for _, row := range t.Rows[:sampleSize] {
		for i, cell := range row {
			if i < len(widths) {
				cellLen := utf8.RuneCountInString(cell)
				// Only increase if specified width is too small, but respect max width
				if cellLen > widths[i] && widths[i] < t.Columns[i].Width {
					widths[i] = min(cellLen, t.Columns[i].Width)
				}
			}
		}
	}

	return widths
}

// renderHeader renders the table header
func (t Table) renderHeader(widths []int) string {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorInfo).
		Background(lipgloss.Color("235"))

	var parts []string
	for i, col := range t.Columns {
		if i < len(widths) {
			cellContent := t.alignText(col.Title, widths[i], col.Align)
			parts = append(parts, headerStyle.Render(cellContent))
		}
	}

	return "  " + strings.Join(parts, "  ")
}

// renderSeparator renders the separator line
func (t Table) renderSeparator(widths []int) string {
	sepStyle := lipgloss.NewStyle().
		Foreground(ColorBorder)

	var parts []string
	for _, width := range widths {
		parts = append(parts, strings.Repeat("─", width))
	}

	return sepStyle.Render("  " + strings.Join(parts, "──"))
}

// renderRow renders a single data row
func (t Table) renderRow(row TableRow, widths []int, rowIndex int) string {
	isSelected := t.selectable && rowIndex == t.selected

	// Row style
	var rowStyle lipgloss.Style
	if isSelected {
		rowStyle = lipgloss.NewStyle().
			Foreground(ColorHighlight).
			Background(lipgloss.Color("237")).
			Bold(true)
	} else {
		rowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))
	}

	// Selection indicator
	indicator := "  "
	if isSelected {
		indicatorStyle := lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)
		indicator = indicatorStyle.Render("▶ ")
	}

	// Render cells
	var parts []string
	for i, cell := range row {
		if i < len(t.Columns) && i < len(widths) {
			align := t.Columns[i].Align
			cellContent := t.alignText(cell, widths[i], align)
			parts = append(parts, rowStyle.Render(cellContent))
		}
	}

	return indicator + strings.Join(parts, "  ")
}

// alignText aligns text within a given width
func (t Table) alignText(text string, width int, align string) string {
	// Truncate if too long
	textLen := utf8.RuneCountInString(text)
	if textLen > width {
		if width > 3 {
			// Convert to runes for proper truncation
			runes := []rune(text)
			return string(runes[:width-3]) + "..."
		}
		runes := []rune(text)
		return string(runes[:width])
	}

	// Pad based on alignment
	padding := width - textLen
	switch align {
	case "right":
		return strings.Repeat(" ", padding) + text
	case "center":
		leftPad := padding / 2
		rightPad := padding - leftPad
		return strings.Repeat(" ", leftPad) + text + strings.Repeat(" ", rightPad)
	default: // left
		return text + strings.Repeat(" ", padding)
	}
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
