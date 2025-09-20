package components

import (
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
)

// InputField represents a text input field
type InputField struct {
	Label       string
	Value       string
	Placeholder string
	Width       int
	CharLimit   int
	Focused     bool
	Validator   func(string) error
	Transform   func(string) string
	cursorPos   int
}

// NewInputField creates a new input field
func NewInputField(label string) *InputField {
	return &InputField{
		Label:     label,
		Width:     40,
		CharLimit: 100,
		cursorPos: 0,
	}
}

// Focus sets focus on the input field
func (i *InputField) Focus() {
	i.Focused = true
	i.cursorPos = len(i.Value)
}

// Blur removes focus from the input field
func (i *InputField) Blur() {
	i.Focused = false
}

// SetValue sets the input value
func (i *InputField) SetValue(value string) {
	i.Value = value
	i.cursorPos = len(value)
}

// Update handles input field updates
func (i *InputField) Update(msg tea.Msg) tea.Cmd {
	if !i.Focused {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyBackspace:
			if i.cursorPos > 0 && len(i.Value) > 0 {
				i.Value = i.Value[:i.cursorPos-1] + i.Value[i.cursorPos:]
				i.cursorPos--
			}

		case tea.KeyDelete:
			if i.cursorPos < len(i.Value) {
				i.Value = i.Value[:i.cursorPos] + i.Value[i.cursorPos+1:]
			}

		case tea.KeyLeft:
			if i.cursorPos > 0 {
				i.cursorPos--
			}

		case tea.KeyRight:
			if i.cursorPos < len(i.Value) {
				i.cursorPos++
			}

		case tea.KeyHome:
			i.cursorPos = 0

		case tea.KeyEnd:
			i.cursorPos = len(i.Value)

		default:
			// Handle character input
			if msg.String() == "space" {
				msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
			}

			if msg.Type == tea.KeyRunes {
				for _, r := range msg.Runes {
					if unicode.IsPrint(r) && len(i.Value) < i.CharLimit {
						// Apply transform if set
						char := string(r)
						if i.Transform != nil {
							char = i.Transform(char)
						}
						i.Value = i.Value[:i.cursorPos] + char + i.Value[i.cursorPos:]
						i.cursorPos += len(char)
					}
				}
			}
		}
	}

	return nil
}

// View renders the input field
func (i InputField) View() string {
	var b strings.Builder

	// Label
	labelStyle := ItemStyle
	if i.Focused {
		labelStyle = SelectedStyle
	}
	b.WriteString(labelStyle.Render(i.Label + ":"))
	b.WriteString(" ")

	// Input value or placeholder
	content := i.Value
	if content == "" && i.Placeholder != "" && !i.Focused {
		content = i.Placeholder
		contentStyle := DescriptionStyle
		b.WriteString(contentStyle.Render(content))
	} else {
		// Show cursor if focused
		if i.Focused {
			before := i.Value[:i.cursorPos]
			after := i.Value[i.cursorPos:]
			cursor := "▌"
			
			b.WriteString(ItemStyle.Render(before))
			b.WriteString(SelectedStyle.Render(cursor))
			b.WriteString(ItemStyle.Render(after))
		} else {
			b.WriteString(ItemStyle.Render(content))
		}
	}

	// Validation error
	if i.Validator != nil {
		if err := i.Validator(i.Value); err != nil {
			b.WriteString("\n")
			b.WriteString(ErrorMessageStyle.Render("  " + err.Error()))
		}
	}

	return b.String()
}

// FormModel represents a form with multiple input fields
type FormModel struct {
	Title       string
	Fields      []*InputField
	currentField int
	submitted   bool
}

// NewForm creates a new form
func NewForm(title string, fields []*InputField) *FormModel {
	if len(fields) > 0 {
		fields[0].Focus()
	}
	return &FormModel{
		Title:  title,
		Fields: fields,
	}
}

// Init initializes the form
func (f FormModel) Init() tea.Cmd {
	return nil
}

// Update handles form updates
func (f FormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			f.nextField()
		case "shift+tab", "up":
			f.prevField()
		case "enter":
			// If on last field, submit form
			if f.currentField == len(f.Fields)-1 {
				f.submitted = true
				return f, nil
			}
			f.nextField()
		case "esc":
			// Don't quit on esc, let parent handle it
			return f, nil
		}
	}

	// Update current field
	if f.currentField < len(f.Fields) {
		cmd := f.Fields[f.currentField].Update(msg)
		return f, cmd
	}

	return f, nil
}

// View renders the form
func (f FormModel) View() string {
	var b strings.Builder

	// Title
	if f.Title != "" {
		b.WriteString(TitleStyle.Render(f.Title))
		b.WriteString("\n\n")
	}

	// Fields
	for i, field := range f.Fields {
		b.WriteString(field.View())
		if i < len(f.Fields)-1 {
			b.WriteString("\n\n")
		}
	}

	// Help
	b.WriteString("\n\n")
	help := []string{
		RenderKeyBinding("tab", "next field"),
		RenderKeyBinding("shift+tab", "previous field"),
		RenderKeyBinding("enter", "submit"),
		RenderKeyBinding("esc", "cancel"),
	}
	b.WriteString(HelpStyle.Render(strings.Join(help, " • ")))

	return BoxStyle.Render(b.String())
}

// nextField moves to the next field
func (f *FormModel) nextField() {
	if f.currentField < len(f.Fields) {
		f.Fields[f.currentField].Blur()
	}
	f.currentField++
	if f.currentField >= len(f.Fields) {
		f.currentField = 0
	}
	if f.currentField < len(f.Fields) {
		f.Fields[f.currentField].Focus()
	}
}

// prevField moves to the previous field
func (f *FormModel) prevField() {
	if f.currentField < len(f.Fields) {
		f.Fields[f.currentField].Blur()
	}
	f.currentField--
	if f.currentField < 0 {
		f.currentField = len(f.Fields) - 1
	}
	if f.currentField < len(f.Fields) {
		f.Fields[f.currentField].Focus()
	}
}

// GetValues returns all field values
func (f FormModel) GetValues() map[string]string {
	values := make(map[string]string)
	for _, field := range f.Fields {
		values[field.Label] = field.Value
	}
	return values
}

// IsSubmitted returns true if the form was submitted
func (f FormModel) IsSubmitted() bool {
	return f.submitted
}