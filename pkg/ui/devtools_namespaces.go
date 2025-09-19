package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/karthickk/k8s-manager/pkg/k8s"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DevToolsNamespaceModel represents the namespace selector
type DevToolsNamespaceModel struct {
	namespaces []string
	selected   int
	loading    bool
	err        error
	client     *k8s.Client
}

// NewDevToolsNamespaceModel creates a new namespace selector
func NewDevToolsNamespaceModel() *DevToolsNamespaceModel {
	return &DevToolsNamespaceModel{
		loading:  true,
		selected: -1,
	}
}

func (m *DevToolsNamespaceModel) Init() tea.Cmd {
	return m.loadNamespaces
}

func (m *DevToolsNamespaceModel) loadNamespaces() tea.Msg {
	client, err := k8s.NewClient()
	if err != nil {
		return namespaceErrorMsg{err}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	namespaceList, err := client.Clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return namespaceErrorMsg{err}
	}

	namespaces := make([]string, 0, len(namespaceList.Items))
	for _, ns := range namespaceList.Items {
		namespaces = append(namespaces, ns.Name)
	}

	return namespacesLoadedMsg{namespaces: namespaces, client: client}
}

type namespaceErrorMsg struct {
	err error
}

type namespacesLoadedMsg struct {
	namespaces []string
	client     *k8s.Client
}

func (m *DevToolsNamespaceModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m, nil

	case namespacesLoadedMsg:
		m.loading = false
		m.namespaces = msg.namespaces
		m.client = msg.client
		return m, nil

	case namespaceErrorMsg:
		m.loading = false
		m.err = msg.err
		return m, nil

	case tea.KeyMsg:
		keyStr := msg.String()

		// Number keys for quick selection
		if len(keyStr) == 1 && keyStr[0] >= '1' && keyStr[0] <= '9' {
			num := int(keyStr[0] - '0')
			if num <= len(m.namespaces) {
				m.selected = num - 1
				return m, tea.Quit
			}
		}

		switch keyStr {
		case "0", "b": // Back
			m.selected = -1
			return m, tea.Quit

		case "q", "ctrl+c", "esc":
			m.selected = -1
			return m, tea.Quit

		case "up", "k":
			if m.selected > 0 {
				m.selected--
			} else if m.selected == -1 && len(m.namespaces) > 0 {
				m.selected = len(m.namespaces) - 1
			}

		case "down", "j":
			if m.selected < len(m.namespaces)-1 {
				m.selected++
			} else if m.selected == -1 && len(m.namespaces) > 0 {
				m.selected = 0
			}

		case "enter", " ":
			if m.selected >= 0 && m.selected < len(m.namespaces) {
				return m, tea.Quit
			}
		}
	}

	return m, nil
}

func (m *DevToolsNamespaceModel) View() string {
	var s strings.Builder

	// Title
	s.WriteString(devToolsTitleStyle.Render("🏷️ Select Namespace"))
	s.WriteString("\n\n")

	if m.loading {
		s.WriteString(devToolsItemStyle.Render("Loading namespaces..."))
		return devToolsContainerStyle.Render(s.String())
	}

	if m.err != nil {
		s.WriteString(devToolsErrorStyle.Render("Error: " + m.err.Error()))
		return devToolsContainerStyle.Render(s.String())
	}

	// Show first 9 namespaces with numbers
	maxItems := 9
	if len(m.namespaces) < maxItems {
		maxItems = len(m.namespaces)
	}

	for i := 0; i < maxItems; i++ {
		ns := m.namespaces[i]

		// Number
		numberStr := devToolsNumberStyle.Render(fmt.Sprintf("%d.", i+1))

		// Namespace name
		nsStr := ns

		// Add selection indicator
		if i == m.selected {
			nsStr = devToolsSelectedStyle.Render("▸ " + nsStr)
		} else {
			nsStr = "  " + devToolsItemStyle.Render(nsStr)
		}

		s.WriteString(numberStr + nsStr)
		s.WriteString("\n")

		// Add description for special namespaces
		var desc string
		switch ns {
		case "default":
			desc = "The default namespace"
		case "kube-system":
			desc = "Kubernetes system components"
		case "kube-public":
			desc = "Public resources"
		case "kube-node-lease":
			desc = "Node heartbeat data"
		default:
			if strings.HasPrefix(ns, "kube-") {
				desc = "System namespace"
			} else {
				desc = "User namespace"
			}
		}
		s.WriteString(devToolsDescriptionStyle.Render("   " + desc))
		s.WriteString("\n")
	}

	if len(m.namespaces) > maxItems {
		s.WriteString("\n")
		s.WriteString(devToolsDescriptionStyle.Render(fmt.Sprintf("   ... and %d more namespaces (use arrows to navigate)", len(m.namespaces)-maxItems)))
	}

	// Back option
	s.WriteString("\n")
	s.WriteString(devToolsNumberStyle.Render("0.") + "  " + devToolsItemStyle.Render("Cancel"))
	s.WriteString("\n")
	s.WriteString(devToolsDescriptionStyle.Render("   Return without selecting"))

	// Help
	s.WriteString("\n\n")
	helpText := "↑/k up • ↓/j down • 1-9 quick select • enter select • 0 cancel • q quit"
	s.WriteString(devToolsHelpStyle.Render(helpText))

	return devToolsContainerStyle.Render(s.String())
}

// GetSelectedNamespace returns the selected namespace
func (m *DevToolsNamespaceModel) GetSelectedNamespace() string {
	if m.selected >= 0 && m.selected < len(m.namespaces) {
		return m.namespaces[m.selected]
	}
	return ""
}