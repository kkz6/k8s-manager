package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/karthickk/k8s-manager/pkg/k8s"
	"github.com/karthickk/k8s-manager/pkg/utils"
	"github.com/pterm/pterm"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	podHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")).
			Background(lipgloss.Color("235")).
			Padding(0, 1)

	podStatusRunning = lipgloss.NewStyle().
				Foreground(lipgloss.Color("42")).
				Bold(true)

	podStatusPending = lipgloss.NewStyle().
				Foreground(lipgloss.Color("214")).
				Bold(true)

	podStatusFailed = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Bold(true)

	podStatusUnknown = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244"))

	podSelectedStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("236")).
				Foreground(lipgloss.Color("170"))

	podActionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true)
)

// PodInfo represents pod information for display
type PodInfo struct {
	Name      string
	Namespace string
	Ready     string
	Status    string
	Restarts  int32
	Age       string
	Node      string
	Pod       *corev1.Pod
}

// PodsModel represents the pods view state
type PodsModel struct {
	pods         []PodInfo
	filteredPods []PodInfo
	table        table.Model
	filterInput  textinput.Model
	spinner      spinner.Model
	loading      bool
	filtering    bool
	actionMenu   bool
	selectedPod  int
	width        int
	height       int
	err          error
	message      string
	client       *k8s.Client
	namespace    string
	allNamespaces bool
}

// PodActionModel represents the action menu for a pod
type PodActionModel struct {
	pod     PodInfo
	actions []string
	cursor  int
	width   int
	height  int
}

func NewPodsModel(namespace string, allNamespaces bool) PodsModel {
	// Create filter input
	ti := textinput.New()
	ti.Placeholder = "Type to filter pods..."
	ti.CharLimit = 100

	// Create spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	// Create table
	columns := []table.Column{
		{Title: "Name", Width: 30},
		{Title: "Ready", Width: 8},
		{Title: "Status", Width: 12},
		{Title: "Restarts", Width: 10},
		{Title: "Age", Width: 10},
		{Title: "Node", Width: 20},
	}

	if allNamespaces {
		// Add namespace column at the beginning
		columns = append([]table.Column{
			{Title: "Namespace", Width: 15},
		}, columns...)
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	// Style the table
	s1 := table.DefaultStyles()
	s1.Header = s1.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s1.Selected = s1.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s1)

	return PodsModel{
		table:         t,
		filterInput:   ti,
		spinner:       s,
		loading:       true,
		namespace:     namespace,
		allNamespaces: allNamespaces,
	}
}

func (m PodsModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.loadPods,
	)
}

func (m PodsModel) loadPods() tea.Msg {
	// Create client
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

type errMsg struct{ err error }
type podsLoadedMsg struct {
	pods   []PodInfo
	client *k8s.Client
}
type refreshPodsMsg struct{}

func (m PodsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.table.SetHeight(msg.Height - 10)
		m.table.SetWidth(msg.Width - 4)
		return m, nil

	case podsLoadedMsg:
		m.loading = false
		m.pods = msg.pods
		m.filteredPods = msg.pods
		m.client = msg.client
		m.updateTableData()
		return m, nil

	case errMsg:
		m.loading = false
		m.err = msg.err
		return m, nil

	case refreshPodsMsg:
		m.loading = true
		return m, m.loadPods

	case tea.KeyMsg:
		if m.actionMenu {
			// Handle action menu
			return m.handleActionMenu(msg)
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "/":
			m.filtering = true
			m.filterInput.Focus()
			return m, textinput.Blink

		case "esc":
			if m.filtering {
				m.filtering = false
				m.filterInput.Blur()
				m.filterInput.SetValue("")
				m.filteredPods = m.pods
				m.updateTableData()
			}
			return m, nil

		case "enter":
			if m.filtering {
				m.filtering = false
				m.filterInput.Blur()
				m.applyFilter()
			} else if len(m.filteredPods) > 0 {
				// Get selected pod and quit to show action menu
				m.selectedPod = m.table.Cursor()
				return m, tea.Quit
			}
			return m, nil

		case "r", "R":
			if !m.filtering {
				m.message = "Refreshing pods..."
				return m, m.loadPods
			}

		case "d":
			if !m.filtering && len(m.filteredPods) > 0 {
				// Quick delete
				idx := m.table.Cursor()
				if idx < len(m.filteredPods) {
					pod := m.filteredPods[idx]
					return m, m.deletePod(pod)
				}
			}

		case "l":
			if !m.filtering && len(m.filteredPods) > 0 {
				// Quick logs
				idx := m.table.Cursor()
				if idx < len(m.filteredPods) {
					pod := m.filteredPods[idx]
					return m, m.viewLogs(pod)
				}
			}
		}

		if m.filtering {
			var cmd tea.Cmd
			m.filterInput, cmd = m.filterInput.Update(msg)
			m.applyFilter()
			cmds = append(cmds, cmd)
		}
	}

	// Update table
	if !m.filtering {
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update spinner
	if m.loading {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *PodsModel) handleActionMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// This would show a submenu for pod actions
	// For now, we'll just handle escape to go back
	if msg.String() == "esc" {
		m.actionMenu = false
	}
	return m, nil
}

func (m *PodsModel) applyFilter() {
	filter := strings.ToLower(m.filterInput.Value())
	if filter == "" {
		m.filteredPods = m.pods
	} else {
		filtered := []PodInfo{}
		for _, pod := range m.pods {
			if strings.Contains(strings.ToLower(pod.Name), filter) ||
				strings.Contains(strings.ToLower(pod.Namespace), filter) ||
				strings.Contains(strings.ToLower(pod.Status), filter) ||
				strings.Contains(strings.ToLower(pod.Node), filter) {
				filtered = append(filtered, pod)
			}
		}
		m.filteredPods = filtered
	}
	m.updateTableData()
}

func (m *PodsModel) updateTableData() {
	rows := []table.Row{}
	for _, pod := range m.filteredPods {
		var row table.Row
		if m.allNamespaces {
			row = table.Row{
				pod.Namespace,
				pod.Name,
				pod.Ready,
				m.colorStatus(pod.Status),
				fmt.Sprint(pod.Restarts),
				pod.Age,
				pod.Node,
			}
		} else {
			row = table.Row{
				pod.Name,
				pod.Ready,
				m.colorStatus(pod.Status),
				fmt.Sprint(pod.Restarts),
				pod.Age,
				pod.Node,
			}
		}
		rows = append(rows, row)
	}
	m.table.SetRows(rows)
}

func (m PodsModel) colorStatus(status string) string {
	switch strings.ToLower(status) {
	case "running":
		return podStatusRunning.Render(status)
	case "pending":
		return podStatusPending.Render(status)
	case "failed", "error", "crashloopbackoff":
		return podStatusFailed.Render(status)
	default:
		return podStatusUnknown.Render(status)
	}
}

func (m PodsModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("\n❌ Error: %v\n", m.err)
	}

	var s strings.Builder

	// Header
	header := "📦 Kubernetes Pods"
	if m.namespace != "" && !m.allNamespaces {
		header += fmt.Sprintf(" (namespace: %s)", m.namespace)
	} else if m.allNamespaces {
		header += " (all namespaces)"
	}
	s.WriteString(podHeaderStyle.Render(header))
	s.WriteString("\n\n")

	// Loading
	if m.loading {
		s.WriteString(fmt.Sprintf("%s Loading pods...\n", m.spinner.View()))
		return s.String()
	}

	// Filter
	if m.filtering {
		s.WriteString("🔍 Filter: ")
		s.WriteString(m.filterInput.View())
		s.WriteString("\n\n")
	}

	// Message
	if m.message != "" {
		s.WriteString(podActionStyle.Render(m.message))
		s.WriteString("\n\n")
	}

	// Table
	if len(m.filteredPods) == 0 {
		s.WriteString("No pods found")
		if m.filterInput.Value() != "" {
			s.WriteString(fmt.Sprintf(" matching '%s'", m.filterInput.Value()))
		}
		s.WriteString("\n")
	} else {
		s.WriteString(m.table.View())
		s.WriteString("\n")
		s.WriteString(fmt.Sprintf("\nShowing %d of %d pods\n", len(m.filteredPods), len(m.pods)))
	}

	// Help
	s.WriteString("\n")
	helpItems := []string{
		"↑/↓: navigate",
		"enter: actions",
		"/: filter",
		"r: refresh",
		"d: delete",
		"l: logs",
		"q: quit",
	}
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(strings.Join(helpItems, " • ")))

	return s.String()
}

func (m PodsModel) deletePod(pod PodInfo) tea.Cmd {
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

		// Refresh the pod list
		return refreshPodsMsg{}
	}
}

func (m PodsModel) viewLogs(pod PodInfo) tea.Cmd {
	return func() tea.Msg {
		// For now, just print that we would view logs
		// In a real implementation, this would open a log viewer
		return tea.Quit
	}
}

// Helper functions (same as in pods.go)
func getPodReadyStatus(pod *corev1.Pod) string {
	readyContainers := 0
	totalContainers := len(pod.Spec.Containers)

	for _, status := range pod.Status.ContainerStatuses {
		if status.Ready {
			readyContainers++
		}
	}

	return fmt.Sprintf("%d/%d", readyContainers, totalContainers)
}

func getPodRestartCount(pod *corev1.Pod) int32 {
	var totalRestarts int32
	for _, status := range pod.Status.ContainerStatuses {
		totalRestarts += status.RestartCount
	}
	return totalRestarts
}

// ShowPodsInteractive displays the interactive pods view
func ShowPodsInteractive(namespace string, allNamespaces bool) error {
	for {
		// Show loading spinner first
		spinner, _ := pterm.DefaultSpinner.Start("Loading pods...")

		m := NewPodsModel(namespace, allNamespaces)
		p := tea.NewProgram(m, tea.WithAltScreen())

		spinner.Stop()

		result, err := p.Run()
		if err != nil {
			return err
		}

		// Check if a pod was selected
		if model, ok := result.(PodsModel); ok {
			if model.selectedPod >= 0 && model.selectedPod < len(model.filteredPods) {
				selectedPod := model.filteredPods[model.selectedPod]

				// Clear screen and show pod actions
				fmt.Print("\033[H\033[2J")

				// Show the pod actions menu
				err := ShowPodActions(selectedPod)
				if err != nil {
					// If there's an error or user backs out, return to pod list
					continue
				}

				// After action completes, ask if they want to continue
				fmt.Println("\nPress Enter to return to pod list, or 'q' to quit...")
				var input string
				fmt.Scanln(&input)
				if input == "q" {
					return nil
				}
				// Continue to show pod list again
			} else {
				// No pod selected, exit
				return nil
			}
		} else {
			return nil
		}
	}
}