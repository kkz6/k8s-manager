package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/karthickk/k8s-manager/pkg/k8s"
	"github.com/karthickk/k8s-manager/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EnhancedPodsModel represents the enhanced pods view with common UI
type EnhancedPodsModel struct {
	list          *List
	pods          []PodInfo
	filteredPods  []PodInfo
	filterInput   textinput.Model
	loading       bool
	filtering     bool
	width         int
	height        int
	err           error
	message       string
	messageType   string
	client        *k8s.Client
	namespace     string
	allNamespaces bool
	showHelp      bool
	keys          NavigationKeys
}

// NewEnhancedPodsModel creates a new enhanced pods model
func NewEnhancedPodsModel(namespace string, allNamespaces bool) EnhancedPodsModel {
	// Create filter input
	ti := textinput.New()
	ti.Placeholder = "Type to filter pods..."
	ti.CharLimit = 100

	// Create navigation keys
	keys := DefaultNavigationKeys()

	// Add custom key bindings for pod operations
	keys.Help = key.NewBinding(
		key.WithKeys("?", "h"),
		key.WithHelp("?/h", "help"),
	)

	return EnhancedPodsModel{
		filterInput:   ti,
		loading:       true,
		namespace:     namespace,
		allNamespaces: allNamespaces,
		keys:          keys,
	}
}

func (m EnhancedPodsModel) Init() tea.Cmd {
	return m.loadPods
}

func (m EnhancedPodsModel) loadPods() tea.Msg {
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

func (m EnhancedPodsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.list != nil {
			m.list.SetSize(msg.Width-4, msg.Height-15)
		}
		return m, nil

	case podsLoadedMsg:
		m.loading = false
		m.pods = msg.pods
		m.filteredPods = msg.pods
		m.client = msg.client
		m.updateList()
		m.message = fmt.Sprintf("Loaded %d pods", len(m.pods))
		m.messageType = "success"
		return m, nil

	case errMsg:
		m.loading = false
		m.err = msg.err
		m.message = fmt.Sprintf("Error: %v", msg.err)
		m.messageType = "error"
		return m, nil

	case tea.KeyMsg:
		// Handle filtering mode first
		if m.filtering {
			switch msg.String() {
			case "esc":
				m.filtering = false
				m.filterInput.Blur()
				m.filterInput.SetValue("")
				m.filteredPods = m.pods
				m.updateList()
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

		// Handle number keys for quick navigation
		if msg.String() >= "1" && msg.String() <= "9" && m.list != nil {
			num := int(msg.String()[0] - '0')
			if num <= len(m.filteredPods) {
				// Select pod by number and show actions
				m.list.cursor = num - 1
				return m, m.showPodActions()
			}
		}

		// Handle other keys
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Search):
			m.filtering = true
			m.filterInput.Focus()
			m.message = "Search mode: Type to filter, Enter to apply, Esc to cancel"
			m.messageType = "info"
			return m, textinput.Blink

		case key.Matches(msg, m.keys.Refresh):
			if !m.filtering {
				m.loading = true
				m.message = "Refreshing pods..."
				m.messageType = "info"
				return m, m.loadPods
			}

		case key.Matches(msg, m.keys.Enter):
			if m.list != nil && len(m.filteredPods) > 0 {
				return m, m.showPodActions()
			}

		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp
			return m, nil

		case msg.String() == "d":
			// Quick delete
			if m.list != nil && len(m.filteredPods) > 0 {
				idx := m.list.GetCursor()
				if idx < len(m.filteredPods) {
					pod := m.filteredPods[idx]
					m.message = fmt.Sprintf("Deleting pod %s...", pod.Name)
					m.messageType = "info"
					return m, m.deletePod(pod)
				}
			}

		case msg.String() == "l":
			// Quick logs view
			if m.list != nil && len(m.filteredPods) > 0 {
				idx := m.list.GetCursor()
				if idx < len(m.filteredPods) {
					return m, tea.Quit // Exit to show logs
				}
			}

		case msg.String() == "x":
			// Quick exec
			if m.list != nil && len(m.filteredPods) > 0 {
				idx := m.list.GetCursor()
				if idx < len(m.filteredPods) {
					return m, tea.Quit // Exit to exec into pod
				}
			}

		case msg.String() == "r":
			// Quick restart (delete and recreate)
			if m.list != nil && len(m.filteredPods) > 0 {
				idx := m.list.GetCursor()
				if idx < len(m.filteredPods) {
					pod := m.filteredPods[idx]
					m.message = fmt.Sprintf("Restarting pod %s...", pod.Name)
					m.messageType = "info"
					return m, m.restartPod(pod)
				}
			}
		}

		// Update the list with navigation keys
		if m.list != nil && !m.filtering {
			updatedList, cmd := m.list.Update(msg)
			m.list = updatedList
			if cmd != nil {
				return m, cmd
			}
		}
	}

	return m, nil
}

func (m *EnhancedPodsModel) showPodActions() tea.Cmd {
	return func() tea.Msg {
		return tea.Quit
	}
}

func (m *EnhancedPodsModel) applyFilter() {
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
	m.updateList()
}

func (m *EnhancedPodsModel) updateList() {
	items := make([]ListItem, 0, len(m.filteredPods))
	for _, pod := range m.filteredPods {
		// Choose icon based on status
		icon := "📦"
		switch strings.ToLower(pod.Status) {
		case "running":
			icon = "✅"
		case "pending":
			icon = "⏳"
		case "failed", "error":
			icon = "❌"
		case "terminating":
			icon = "🔄"
		}

		details := make(map[string]string)
		details["Ready"] = pod.Ready
		details["Restarts"] = fmt.Sprint(pod.Restarts)
		details["Age"] = pod.Age
		if pod.Node != "" {
			details["Node"] = pod.Node
		}
		if m.allNamespaces {
			details["Namespace"] = pod.Namespace
		}

		items = append(items, ListItem{
			ID:          pod.Name,
			Title:       pod.Name,
			Status:      pod.Status,
			Description: fmt.Sprintf("Ready: %s, Restarts: %d, Age: %s", pod.Ready, pod.Restarts, pod.Age),
			Details:     details,
			Icon:        icon,
		})
	}

	m.list = NewList(items, true)
	if m.width > 0 && m.height > 0 {
		m.list.SetSize(m.width-4, m.height-15)
	}
}

func (m EnhancedPodsModel) deletePod(pod PodInfo) tea.Cmd {
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

		// Reload pods after deletion
		return m.loadPods()
	}
}

func (m EnhancedPodsModel) restartPod(pod PodInfo) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		gracePeriod := int64(0) // Force delete for quick restart
		err := m.client.Clientset.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{
			GracePeriodSeconds: &gracePeriod,
		})
		if err != nil {
			return errMsg{err}
		}

		// Reload pods after restart
		time.Sleep(2 * time.Second) // Give it a moment to restart
		return m.loadPods()
	}
}

func (m EnhancedPodsModel) View() string {
	if m.err != nil {
		return RenderMessage("error", fmt.Sprintf("Error: %v", m.err))
	}

	var s strings.Builder

	// Title
	title := "🚀 Kubernetes Pods Manager"
	subtitle := ""
	if m.namespace != "" && !m.allNamespaces {
		subtitle = fmt.Sprintf("Namespace: %s", m.namespace)
	} else if m.allNamespaces {
		subtitle = "All Namespaces"
	}
	s.WriteString(RenderTitle(title, subtitle))
	s.WriteString("\n\n")

	// Loading state
	if m.loading {
		spinner := LoadingSpinner("Loading pods...")
		s.WriteString(ContentBoxStyle.Render(spinner.View() + " Loading pods..."))
		return AppStyle.Render(s.String())
	}

	// Filter input
	if m.filtering {
		s.WriteString("🔍 Filter: ")
		s.WriteString(m.filterInput.View())
		s.WriteString("\n\n")
	}

	// Message
	if m.message != "" {
		s.WriteString(RenderMessage(m.messageType, m.message))
		s.WriteString("\n")
	}

	// Pod list or empty state
	if len(m.filteredPods) == 0 {
		emptyMsg := "No pods found"
		if m.filterInput.Value() != "" {
			emptyMsg = fmt.Sprintf("No pods found matching '%s'", m.filterInput.Value())
		}
		s.WriteString(ContentBoxStyle.Render(emptyMsg))
	} else {
		if m.list != nil {
			s.WriteString(m.list.View())
		}
		s.WriteString(fmt.Sprintf("\n\n📊 Showing %d of %d pods", len(m.filteredPods), len(m.pods)))
	}

	// Help section
	if m.showHelp {
		helpText := `
🎮 Navigation:
  ↑/k, ↓/j     Navigate up/down
  PgUp/b, PgDn/f   Page up/down
  Home/g, End/G    First/last item
  1-9             Quick select by number

🔧 Actions:
  Enter/Space     Show pod actions menu
  d               Quick delete pod
  l               View pod logs
  x               Exec into pod
  r               Restart pod

🔍 Features:
  /               Search/filter pods
  R/F5            Refresh pod list
  ?/h             Toggle this help
  q/Ctrl+C        Quit
`
		s.WriteString("\n")
		s.WriteString(HelpStyle.Render(helpText))
	} else {
		// Compact help
		additionalHelp := []string{
			"d: delete",
			"l: logs",
			"x: exec",
			"r: restart",
		}
		s.WriteString("\n")
		s.WriteString(RenderHelp(m.keys, additionalHelp...))
	}

	return AppStyle.Render(s.String())
}

// GetSelectedPod returns the currently selected pod
func (m EnhancedPodsModel) GetSelectedPod() *PodInfo {
	if m.list != nil && len(m.filteredPods) > 0 {
		idx := m.list.GetCursor()
		if idx >= 0 && idx < len(m.filteredPods) {
			return &m.filteredPods[idx]
		}
	}
	return nil
}