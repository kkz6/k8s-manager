package views

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/karthickk/k8s-manager/internal/services"
	"github.com/karthickk/k8s-manager/internal/ui/components"
	corev1 "k8s.io/api/core/v1"
)

// LogsViewModel shows pod logs with better formatting
type LogsViewModel struct {
	namespace   string
	podName     string
	container   string
	follow      bool
	viewport    viewport.Model
	ready       bool
	quitting    bool
	lines       []string
	mu          sync.Mutex
	ctx         context.Context
	cancel      context.CancelFunc
	errorMsg    string
	logStream   io.ReadCloser
	logReader   *bufio.Reader
}

// NewLogsViewModel creates a new logs view
func NewLogsViewModel(namespace, podName, container string, follow bool) *LogsViewModel {
	ctx, cancel := context.WithCancel(context.Background())
	return &LogsViewModel{
		namespace: namespace,
		podName:   podName,
		container: container,
		follow:    follow,
		lines:     []string{},
		ctx:       ctx,
		cancel:    cancel,
	}
}

// logsUpdateMsg is sent when new log lines are available
type logsUpdateMsg struct {
	lines []string
}

// logsErrorMsg is sent when there's an error
type logsErrorMsg struct {
	err error
}

// Init starts fetching logs
func (m *LogsViewModel) Init() tea.Cmd {
	return tea.Batch(
		m.initLogStream(),
		tea.WindowSize(),
	)
}

// Update handles messages
func (m *LogsViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		headerHeight := 5
		footerHeight := 2
		verticalMargin := headerHeight + footerHeight

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMargin)
			m.viewport.Style = lipgloss.NewStyle()
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMargin
		}
		m.updateViewport()

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.quitting = true
			m.cancel() // Cancel log streaming
			if m.logStream != nil {
				m.logStream.Close()
			}
			return m, tea.Quit
		case "g", "home":
			m.viewport.GotoTop()
		case "G", "end":
			m.viewport.GotoBottom()
		case "ctrl+l":
			// Clear logs
			m.mu.Lock()
			m.lines = []string{}
			m.mu.Unlock()
			m.updateViewport()
		}

	case logsUpdateMsg:
		m.mu.Lock()
		m.lines = append(m.lines, msg.lines...)
		// Keep only last 10000 lines for performance
		if len(m.lines) > 10000 {
			m.lines = m.lines[len(m.lines)-10000:]
		}
		m.mu.Unlock()
		m.updateViewport()
		// Auto-scroll to bottom if following
		if m.follow {
			m.viewport.GotoBottom()
		}
		// Continue fetching if following
		if m.follow {
			return m, m.readMoreLogs()
		}

	case logsErrorMsg:
		m.errorMsg = msg.err.Error()
		return m, nil
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View renders the logs
func (m *LogsViewModel) View() string {
	if m.quitting {
		return ""
	}

	if !m.ready {
		return "\n  Initializing logs viewer..."
	}

	// Title
	title := fmt.Sprintf("📝 Pod Logs: %s", m.podName)
	if m.follow {
		title += " [FOLLOWING]"
	}
	if m.container != "" {
		title += fmt.Sprintf(" (container: %s)", m.container)
	}

	header := components.TitleStyle.Render(title)

	// Error message
	if m.errorMsg != "" {
		errorBox := components.ErrorMessageStyle.Render(m.errorMsg)
		return fmt.Sprintf("%s\n\n%s", header, errorBox)
	}

	// Footer with controls
	var footerText string
	if m.follow {
		footerText = "q/esc/ctrl+c: stop following • g/G: top/bottom • ctrl+l: clear"
	} else {
		footerText = "↑/↓: scroll • g/G: top/bottom • ctrl+l: clear • q/esc: back"
	}
	
	scrollPos := ""
	if m.viewport.TotalLineCount() > m.viewport.Height {
		scrollPos = fmt.Sprintf(" • Line %d/%d", m.viewport.YOffset+1, m.viewport.TotalLineCount())
	}
	
	footer := components.HelpStyle.Render(footerText + scrollPos)

	// Logs count
	m.mu.Lock()
	logCount := len(m.lines)
	m.mu.Unlock()
	
	countText := fmt.Sprintf("Showing %d lines", logCount)
	if m.follow {
		countText += " • 🔴 Live"
	}
	countLine := components.DescriptionStyle.Render(countText)

	// Combine all parts with proper spacing
	return strings.Join([]string{
		header,
		countLine,
		m.viewport.View(),
		footer,
	}, "\n")
}

// updateViewport updates the viewport content
func (m *LogsViewModel) updateViewport() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Format logs with minimal spacing
	var formattedLines []string
	for _, line := range m.lines {
		// Trim excessive whitespace but keep the line structure
		trimmed := strings.TrimRight(line, " \t")
		if trimmed != "" {
			formattedLines = append(formattedLines, trimmed)
		}
	}
	
	content := strings.Join(formattedLines, "\n")
	m.viewport.SetContent(content)
}

// initLogStream initializes the log stream
func (m *LogsViewModel) initLogStream() tea.Cmd {
	return func() tea.Msg {
		client, err := services.GetK8sClient()
		if err != nil {
			return logsErrorMsg{err: err}
		}

		// Build log options
		opts := &corev1.PodLogOptions{
			Follow: m.follow,
		}
		
		if m.container != "" {
			opts.Container = m.container
		}
		
		if !m.follow {
			// For static logs, get last 1000 lines
			tailLines := int64(1000)
			opts.TailLines = &tailLines
		}

		// Get log stream
		req := client.Clientset.CoreV1().Pods(m.namespace).GetLogs(m.podName, opts)
		stream, err := req.Stream(m.ctx)
		if err != nil {
			return logsErrorMsg{err: err}
		}

		// Store stream and reader for later use
		m.logStream = stream
		m.logReader = bufio.NewReader(stream)

		// Start reading logs
		if m.follow {
			return m.readMoreLogs()()
		} else {
			return m.readAllLogs()()
		}
	}
}

// readAllLogs reads all logs for non-follow mode
func (m *LogsViewModel) readAllLogs() tea.Cmd {
	return func() tea.Msg {
		if m.logReader == nil {
			return logsErrorMsg{err: fmt.Errorf("log reader not initialized")}
		}

		var allLines []string
		for {
			line, err := m.logReader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
				return logsErrorMsg{err: err}
			}
			// Remove newline
			line = strings.TrimSuffix(line, "\n")
			allLines = append(allLines, line)
		}
		
		// Close the stream for non-follow mode
		if m.logStream != nil {
			m.logStream.Close()
		}
		
		return logsUpdateMsg{lines: allLines}
	}
}

// readMoreLogs reads more logs for follow mode
func (m *LogsViewModel) readMoreLogs() tea.Cmd {
	return func() tea.Msg {
		if m.logReader == nil {
			return logsErrorMsg{err: fmt.Errorf("log reader not initialized")}
		}

		var newLines []string
		for {
			select {
			case <-m.ctx.Done():
				return nil
			default:
				// Set a read deadline to avoid blocking forever
				line, err := m.logReader.ReadString('\n')
				if err != nil {
					if err == io.EOF {
						// If we have some lines, send them
						if len(newLines) > 0 {
							return logsUpdateMsg{lines: newLines}
						}
						// Otherwise, wait a bit and try again
						time.Sleep(100 * time.Millisecond)
						return m.readMoreLogs()()
					}
					return logsErrorMsg{err: err}
				}
				
				// Remove newline
				line = strings.TrimSuffix(line, "\n")
				if line != "" {
					newLines = append(newLines, line)
				}
				
				// Send batch updates for better performance
				if len(newLines) >= 10 {
					return logsUpdateMsg{lines: newLines}
				}
			}
		}
	}
}

// ShowLogsView shows the logs viewer
func ShowLogsView(namespace, podName, container string, follow bool) tea.Cmd {
	return func() tea.Msg {
		model := NewLogsViewModel(namespace, podName, container, follow)
		p := tea.NewProgram(model, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return components.ErrorMsg{Error: err}
		}
		return nil
	}
}