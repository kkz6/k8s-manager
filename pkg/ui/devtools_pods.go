package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/karthickk/k8s-manager/pkg/k8s"
	"github.com/karthickk/k8s-manager/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DevToolsPodsModel represents the pods view in DevTools style
type DevToolsPodsModel struct {
	pods          []PodInfo
	filteredPods  []PodInfo
	selected      int
	filterInput   textinput.Model
	filtering     bool
	loading       bool
	loadingAction bool // Track if we're loading an action
	spinner       AnimatedSpinner // Animated spinner
	width         int
	height        int
	client        *k8s.Client
	namespace     string
	allNamespaces bool
	message       string
	err           error
	podSelected   bool // Track if a pod was selected
}

// NewDevToolsPodsModel creates a new DevTools-style pods model
func NewDevToolsPodsModel(namespace string, allNamespaces bool) *DevToolsPodsModel {
	ti := textinput.New()
	ti.Placeholder = "Type to filter..."
	ti.CharLimit = 50

	return &DevToolsPodsModel{
		filterInput:   ti,
		loading:       true,
		spinner:       NewAnimatedSpinner("spinner", "Loading pods"),
		namespace:     namespace,
		allNamespaces: allNamespaces,
		selected:      -1,
	}
}

func (m *DevToolsPodsModel) Init() tea.Cmd {
	return tea.Batch(
		m.loadPods,
		m.spinner.Init(),
	)
}

func (m *DevToolsPodsModel) loadPods() tea.Msg {
	client, err := k8s.NewClient()
	if err != nil {
		return errMsg{err}
	}

	namespace := m.namespace
	if namespace == "" && !m.allNamespaces {
		namespace = client.GetNamespace()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var pods *corev1.PodList
	if m.allNamespaces {
		pods, err = client.Clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	} else {
		pods, err = client.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	}

	if err != nil {
		return errMsg{err}
	}

	podInfos := make([]PodInfo, 0, len(pods.Items))
	for _, pod := range pods.Items {
		info := PodInfo{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			Ready:     getPodReadyStatus(&pod),
			Status:    string(pod.Status.Phase),
			Restarts:  getPodRestartCount(&pod),
			Age:       utils.FormatAge(pod.CreationTimestamp.Time),
			Node:      pod.Spec.NodeName,
			Pod:       &pod,
		}
		podInfos = append(podInfos, info)
	}

	return podsLoadedMsg{pods: podInfos, client: client}
}

func (m *DevToolsPodsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Update spinner animation
	if m.loading || m.loadingAction {
		spinner, cmd := m.spinner.Update(msg)
		m.spinner = spinner
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case podsLoadedMsg:
		m.loading = false
		m.pods = msg.pods
		m.filteredPods = msg.pods
		m.client = msg.client
		return m, nil

	case errMsg:
		m.loading = false
		m.err = msg.err
		return m, nil

	case tea.KeyMsg:
		if m.filtering {
			switch msg.String() {
			case "esc":
				m.filtering = false
				m.filterInput.Blur()
				m.filterInput.SetValue("")
				m.filteredPods = m.pods
				return m, nil

			case "enter":
				m.filtering = false
				m.filterInput.Blur()
				m.applyFilter()
				return m, nil

			default:
				var cmd tea.Cmd
				m.filterInput, cmd = m.filterInput.Update(msg)
				m.applyFilter()
				return m, cmd
			}
		}

		keyStr := msg.String()

		// Number keys for quick selection
		if len(keyStr) == 1 && keyStr[0] >= '1' && keyStr[0] <= '8' {
			num := int(keyStr[0] - '0')
			if num <= len(m.filteredPods) {
				m.selected = num - 1
				m.podSelected = true
				m.loadingAction = true
				m.spinner = NewAnimatedSpinner("spinner", fmt.Sprintf("Loading actions for pod %s", m.filteredPods[m.selected].Name))
				return m, tea.Batch(
					m.spinner.Init(),
					tea.Tick(time.Millisecond*300, func(t time.Time) tea.Msg {
						return tea.Quit()
					}),
				)
			}
		}

		// Special keys
		switch keyStr {
		case "9": // Refresh
			m.loading = true
			m.spinner = NewAnimatedSpinner("spinner", "Refreshing pods")
			return m, tea.Batch(
				m.loadPods,
				m.spinner.Init(),
			)

		case "0", "b": // Back to main menu
			return m, tea.Quit

		case "q", "ctrl+c":
			return m, tea.Quit

		case "esc":
			return m, tea.Quit

		case "/":
			m.filtering = true
			m.filterInput.Focus()
			return m, textinput.Blink

		case "up", "k":
			if m.selected > 0 {
				m.selected--
			} else if m.selected == -1 && len(m.filteredPods) > 0 {
				m.selected = len(m.filteredPods) - 1
			}

		case "down", "j":
			if m.selected < len(m.filteredPods)-1 {
				m.selected++
			} else if m.selected == -1 && len(m.filteredPods) > 0 {
				m.selected = 0
			}

		case "enter", " ":
			if m.selected >= 0 && m.selected < len(m.filteredPods) {
				m.podSelected = true
				m.loadingAction = true
				m.spinner = NewAnimatedSpinner("spinner", fmt.Sprintf("Loading actions for pod %s", m.filteredPods[m.selected].Name))
				return m, tea.Batch(
					m.spinner.Init(),
					tea.Tick(time.Millisecond*300, func(t time.Time) tea.Msg {
						return tea.Quit()
					}),
				)
			}

		case "r":
			m.loading = true
			m.spinner = NewAnimatedSpinner("spinner", "Refreshing pods")
			return m, tea.Batch(
				m.loadPods,
				m.spinner.Init(),
			)

		case "d":
			if m.selected >= 0 && m.selected < len(m.filteredPods) {
				return m, m.deletePod()
			}

		case "l":
			if m.selected >= 0 && m.selected < len(m.filteredPods) {
				return m, m.viewLogs()
			}
		}
	}

	if len(cmds) > 0 {
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

func (m *DevToolsPodsModel) View() string {
	var s strings.Builder

	if m.err != nil {
		s.WriteString(devToolsContainerStyle.Render(
			devToolsErrorStyle.Render("Error: " + m.err.Error()),
		))
		return s.String()
	}

	// Title - same style as main menu
	title := "📦 Kubernetes Pods"
	if m.namespace != "" && !m.allNamespaces {
		title += fmt.Sprintf(" - %s namespace", m.namespace)
	} else if m.allNamespaces {
		title += " - all namespaces"
	}
	s.WriteString(devToolsTitleStyle.Render(title))
	s.WriteString("\n\n")

	if m.loading || m.loadingAction {
		s.WriteString("\n")
		s.WriteString(m.spinner.View())
		s.WriteString("\n")
		return devToolsContainerStyle.Render(s.String())
	}

	// Filter
	if m.filtering {
		s.WriteString("Filter: ")
		s.WriteString(m.filterInput.View())
		s.WriteString("\n\n")
	}

	// Pods list with numbers - same format as main menu
	if len(m.filteredPods) == 0 {
		s.WriteString(devToolsDescriptionStyle.Render("No pods found"))
		if m.filterInput.Value() != "" {
			s.WriteString(devToolsDescriptionStyle.Render(fmt.Sprintf(" matching '%s'", m.filterInput.Value())))
		}
		s.WriteString("\n\n")

		// Add option to go back
		s.WriteString(devToolsNumberStyle.Render("0.") + "  " + devToolsItemStyle.Render("Back to Main Menu"))
		s.WriteString("\n")
		s.WriteString(devToolsDescriptionStyle.Render("   Return to the main menu"))
	} else {
		// Show first 8 pods with numbers (leaving 9 for refresh and 0 for back)
		maxItems := 8
		if len(m.filteredPods) < maxItems {
			maxItems = len(m.filteredPods)
		}

		for i := 0; i < maxItems; i++ {
			pod := m.filteredPods[i]

			// Number
			numberStr := devToolsNumberStyle.Render(fmt.Sprintf("%d.", i+1))

			// Pod name - same style as menu items
			podStr := pod.Name

			// Add selection indicator
			if i == m.selected {
				podStr = devToolsSelectedStyle.Render("▸ " + podStr)
			} else {
				podStr = "  " + devToolsItemStyle.Render(podStr)
			}

			// Status
			statusStr := m.getStatusString(pod.Status)

			s.WriteString(numberStr + podStr + " " + statusStr)
			s.WriteString("\n")

			// Pod details (indented) - same style as menu descriptions
			details := fmt.Sprintf("Ready: %s, Restarts: %d, Age: %s", pod.Ready, pod.Restarts, pod.Age)
			if m.allNamespaces {
				details = fmt.Sprintf("Namespace: %s, %s", pod.Namespace, details)
			}
			s.WriteString(devToolsDescriptionStyle.Render("   " + details))
			s.WriteString("\n")
		}

		if len(m.filteredPods) > maxItems {
			s.WriteString("\n")
			s.WriteString(devToolsDescriptionStyle.Render(fmt.Sprintf("   ... and %d more pods (use arrows to navigate)", len(m.filteredPods)-maxItems)))
			s.WriteString("\n")
		}

		// Add refresh option
		s.WriteString("\n")
		s.WriteString(devToolsNumberStyle.Render("9.") + "  " + devToolsItemStyle.Render("Refresh"))
		s.WriteString("\n")
		s.WriteString(devToolsDescriptionStyle.Render("   Reload the pods list"))
		s.WriteString("\n")

		// Add back option
		s.WriteString(devToolsNumberStyle.Render("0.") + "  " + devToolsItemStyle.Render("Back to Main Menu"))
		s.WriteString("\n")
		s.WriteString(devToolsDescriptionStyle.Render("   Return to the main menu"))
	}

	// Message
	if m.message != "" {
		s.WriteString("\n")
		s.WriteString(devToolsInfoStyle.Render(m.message))
	}

	// Help - same style as main menu
	s.WriteString("\n\n")
	helpText := "↑/k up • ↓/j down • 1-8 select pod • 9 refresh • 0 back • / filter • q quit"
	s.WriteString(devToolsHelpStyle.Render(helpText))

	return devToolsContainerStyle.Render(s.String())
}

func (m *DevToolsPodsModel) getStatusString(status string) string {
	switch strings.ToLower(status) {
	case "running":
		return devToolsSuccessStyle.Render("[Running]")
	case "pending":
		return devToolsWarningStyle.Render("[Pending]")
	case "failed", "error", "crashloopbackoff":
		return devToolsErrorStyle.Render("[" + status + "]")
	case "terminating":
		return devToolsWarningStyle.Render("[Terminating]")
	default:
		return devToolsDescriptionStyle.Render("[" + status + "]")
	}
}

func (m *DevToolsPodsModel) applyFilter() {
	filter := strings.ToLower(m.filterInput.Value())
	if filter == "" {
		m.filteredPods = m.pods
	} else {
		filtered := []PodInfo{}
		for _, pod := range m.pods {
			if strings.Contains(strings.ToLower(pod.Name), filter) ||
				strings.Contains(strings.ToLower(pod.Namespace), filter) ||
				strings.Contains(strings.ToLower(pod.Status), filter) {
				filtered = append(filtered, pod)
			}
		}
		m.filteredPods = filtered
	}
	m.selected = -1 // Reset selection
}


func (m *DevToolsPodsModel) deletePod() tea.Cmd {
	if m.selected < 0 || m.selected >= len(m.filteredPods) {
		return nil
	}

	pod := m.filteredPods[m.selected]
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		gracePeriod := int64(30)
		err := m.client.Clientset.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{
			GracePeriodSeconds: &gracePeriod,
		})
		if err != nil {
			return errMsg{err}
		}

		return m.loadPods()
	}
}

func (m *DevToolsPodsModel) viewLogs() tea.Cmd {
	return func() tea.Msg {
		return tea.Quit
	}
}

// GetSelectedPod returns the currently selected pod
func (m *DevToolsPodsModel) GetSelectedPod() *PodInfo {
	if m.podSelected && m.selected >= 0 && m.selected < len(m.filteredPods) {
		return &m.filteredPods[m.selected]
	}
	return nil
}

// Additional DevTools styles
var (
	devToolsErrorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196"))

	devToolsSuccessStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("42"))

	devToolsWarningStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("214"))

	devToolsInfoStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("86"))
)