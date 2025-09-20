package ui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/karthickk/k8s-manager/pkg/k8s"
	"github.com/pterm/pterm"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EnhancedPodActionsModel represents the enhanced pod actions view
type EnhancedPodActionsModel struct {
	pod            PodInfo
	menu           *List
	client         *k8s.Client
	width          int
	height         int
	loading        bool
	executing      bool
	message        string
	messageType    string
	showHelp       bool
	keys           NavigationKeys
	spinner        spinner.Model
	currentAction  string
}

// NewEnhancedPodActionsModel creates a new enhanced pod actions model
func NewEnhancedPodActionsModel(pod PodInfo, client *k8s.Client) EnhancedPodActionsModel {
	// Create menu items with handlers
	menuItems := []MenuItem{
		{
			Title:       "Describe Pod",
			Description: "Show detailed pod information and events",
			Icon:        "📋",
			Number:      1,
			Handler:     nil, // Will be set after model creation
		},
		{
			Title:       "View Logs",
			Description: "View recent pod logs (last 100 lines)",
			Icon:        "📝",
			Number:      2,
			Handler:     nil,
		},
		{
			Title:       "Follow Logs",
			Description: "Follow pod logs in real-time",
			Icon:        "📊",
			Number:      3,
			Handler:     nil,
		},
		{
			Title:       "Execute Shell",
			Description: "Open interactive shell session in pod",
			Icon:        "⚡",
			Number:      4,
			Handler:     nil,
		},
		{
			Title:       "Port Forward",
			Description: "Forward local port to pod port",
			Icon:        "🌐",
			Number:      5,
			Handler:     nil,
		},
		{
			Title:       "Resource Usage",
			Description: "Show CPU and memory usage metrics",
			Icon:        "📈",
			Number:      6,
			Handler:     nil,
		},
		{
			Title:       "Edit Pod",
			Description: "Edit pod configuration (advanced)",
			Icon:        "✏️",
			Number:      7,
			Handler:     nil,
		},
		{
			Title:       "Restart Pod",
			Description: "Delete pod to trigger restart",
			Icon:        "🔄",
			Number:      8,
			Handler:     nil,
		},
		{
			Title:       "Delete Pod",
			Description: "Permanently delete the pod",
			Icon:        "🗑️",
			Number:      9,
			Handler:     nil,
		},
	}

	menu := NewMenu(menuItems)
	keys := DefaultNavigationKeys()

	// Create spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return EnhancedPodActionsModel{
		pod:     pod,
		menu:    menu,
		client:  client,
		loading: false,
		keys:    keys,
		spinner: s,
	}
}

func (m EnhancedPodActionsModel) Init() tea.Cmd {
	// Set up handlers that need access to model
	for i := range m.menu.MenuItems {
		idx := i
		m.menu.MenuItems[idx].Handler = func() tea.Cmd {
			return m.executeAction(idx)
		}
	}
	return m.spinner.Tick
}

func (m EnhancedPodActionsModel) executeAction(actionIndex int) tea.Cmd {
	return func() tea.Msg {
		if actionIndex >= len(m.menu.MenuItems) {
			return nil
		}

		// Set the current action
		m.currentAction = m.menu.MenuItems[actionIndex].Title
		m.executing = true

		// Handle different actions
		switch actionIndex {
		case 0: // Describe
			return m.describePod()
		case 1: // View Logs
			return m.viewLogs()
		case 2: // Follow Logs
			return m.followLogs()
		case 3: // Execute Shell
			return m.execShell()
		case 4: // Port Forward
			return m.portForward()
		case 5: // Resource Usage
			return m.resourceUsage()
		case 6: // Edit Pod
			return m.editPod()
		case 7: // Restart Pod
			return m.restartPod()
		case 8: // Delete Pod
			return m.deletePod()
		}

		return nil
	}
}

func (m EnhancedPodActionsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.menu != nil {
			m.menu.SetSize(msg.Width-4, msg.Height-15)
		}
		return m, nil

	case tea.KeyMsg:
		// Don't process keys while executing
		if m.executing {
			return m, nil
		}

		// Handle special keys first
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "?", "h":
			m.showHelp = !m.showHelp
			return m, nil

		case "esc", "backspace":
			// Go back to pod list
			return m, tea.Quit
		}

		// Let the menu handle other keys
		if m.menu != nil {
			updatedMenu, cmd := m.menu.Update(msg)
			m.menu = updatedMenu
			if cmd != nil {
				return m, cmd
			}
		}

	case actionResultMsg:
		m.executing = false
		m.currentAction = ""
		if msg.err != nil {
			m.message = fmt.Sprintf("Error: %v", msg.err)
			m.messageType = "error"
		} else if msg.message != "" {
			m.message = msg.message
			m.messageType = "success"
		}
		return m, nil

	case spinner.TickMsg:
		if m.executing {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	return m, nil
}

type actionResultMsg struct {
	message string
	err     error
}

func (m EnhancedPodActionsModel) View() string {
	var s strings.Builder

	// Title with pod info
	title := fmt.Sprintf("🔧 Pod Actions: %s", m.pod.Name)
	subtitle := fmt.Sprintf("Namespace: %s | Status: %s | Ready: %s | Age: %s",
		m.pod.Namespace, m.pod.Status, m.pod.Ready, m.pod.Age)
	s.WriteString(RenderTitle(title, subtitle))
	s.WriteString("\n\n")

	// Pod details box
	var details strings.Builder
	details.WriteString("📦 Pod Information\n")
	details.WriteString(strings.Repeat("─", 40) + "\n")

	// Status with color
	statusStr := m.pod.Status
	switch strings.ToLower(statusStr) {
	case "running":
		statusStr = StatusRunningStyle.Render(statusStr)
	case "pending":
		statusStr = StatusPendingStyle.Render(statusStr)
	case "failed", "error", "crashloopbackoff":
		statusStr = StatusErrorStyle.Render(statusStr)
	}
	details.WriteString(fmt.Sprintf("  Status:   %s\n", statusStr))

	// Ready status with color
	readyParts := strings.Split(m.pod.Ready, "/")
	readyStr := m.pod.Ready
	if len(readyParts) == 2 && readyParts[0] == readyParts[1] {
		readyStr = StatusRunningStyle.Render(readyStr)
	} else {
		readyStr = StatusPendingStyle.Render(readyStr)
	}
	details.WriteString(fmt.Sprintf("  Ready:    %s\n", readyStr))

	details.WriteString(fmt.Sprintf("  Restarts: %d\n", m.pod.Restarts))
	details.WriteString(fmt.Sprintf("  Age:      %s\n", m.pod.Age))
	if m.pod.Node != "" {
		details.WriteString(fmt.Sprintf("  Node:     %s\n", m.pod.Node))
	}

	s.WriteString(ContentBoxStyle.Render(details.String()))
	s.WriteString("\n")

	// Show loading spinner when executing
	if m.executing {
		s.WriteString("\n")
		s.WriteString(lipgloss.NewStyle().
			Padding(1, 2).
			Render(fmt.Sprintf("%s Executing %s...", m.spinner.View(), m.currentAction)))
		s.WriteString("\n")
	}

	// Message
	if m.message != "" && !m.executing {
		s.WriteString(RenderMessage(m.messageType, m.message))
		s.WriteString("\n")
	}

	// Actions menu
	s.WriteString(HeaderStyle.Render("Available Actions"))
	s.WriteString("\n")

	if m.menu != nil {
		s.WriteString(m.menu.View())
	}

	// Help section
	if m.showHelp {
		helpText := `
🎮 Quick Actions:
  1-9             Direct action selection
  ↑/k, ↓/j       Navigate menu
  Enter/Space     Execute selected action

🔧 Pod Operations:
  1 - Describe    Show pod details and events
  2 - View Logs   View recent logs (last 100 lines)
  3 - Follow Logs Stream logs in real-time
  4 - Exec Shell  Open interactive shell
  5 - Port Fwd    Forward ports to localhost
  6 - Resources   View CPU/memory usage
  7 - Edit        Edit pod configuration
  8 - Restart     Delete and recreate pod
  9 - Delete      Permanently remove pod

📝 Navigation:
  ?/h             Toggle this help
  Esc/Backspace   Return to pods list
  q/Ctrl+C        Quit application
`
		s.WriteString("\n")
		s.WriteString(HelpStyle.Render(helpText))
	} else {
		// Compact help
		s.WriteString("\n")
		s.WriteString(FooterStyle.Render("Press 1-9 for quick action • ↑/↓ to navigate • Enter to select • ?/h for help • Esc to go back"))
	}

	return AppStyle.Render(s.String())
}

// Action implementations
func (m EnhancedPodActionsModel) describePod() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pod, err := m.client.Clientset.CoreV1().Pods(m.pod.Namespace).Get(ctx, m.pod.Name, metav1.GetOptions{})
	if err != nil {
		return actionResultMsg{err: err}
	}

	// Get events for the pod
	events, _ := m.client.Clientset.CoreV1().Events(m.pod.Namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s", m.pod.Name),
	})

	// Display pod details
	fmt.Print("\033[H\033[2J") // Clear screen
	pterm.DefaultHeader.Println("Pod Details")

	data := [][]string{
		{"Name", pod.Name},
		{"Namespace", pod.Namespace},
		{"Status", string(pod.Status.Phase)},
		{"Node", pod.Spec.NodeName},
		{"IP", pod.Status.PodIP},
		{"Created", pod.CreationTimestamp.Format(time.RFC3339)},
	}

	pterm.DefaultTable.WithData(data).Render()

	// Show containers
	pterm.DefaultSection.Println("Containers")
	for _, container := range pod.Spec.Containers {
		containerData := [][]string{
			{"Name", container.Name},
			{"Image", container.Image},
			{"Ports", formatPorts(container.Ports)},
		}
		pterm.DefaultTable.WithData(containerData).Render()
	}

	// Show recent events
	if len(events.Items) > 0 {
		pterm.DefaultSection.Println("Recent Events")
		for _, event := range events.Items {
			eventTime := event.LastTimestamp.Format("15:04:05")
			fmt.Printf("[%s] %s: %s\n", eventTime, event.Reason, event.Message)
		}
	}

	fmt.Println("\nPress Enter to continue...")
	fmt.Scanln()

	return actionResultMsg{message: "Pod description viewed"}
}

func (m EnhancedPodActionsModel) viewLogs() tea.Msg {
	fmt.Print("\033[H\033[2J") // Clear screen
	pterm.DefaultHeader.Printf("Pod Logs: %s\n", m.pod.Name)

	cmd := exec.Command("kubectl", "logs", m.pod.Name, "-n", m.pod.Namespace, "--tail=100")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()

	fmt.Println("\nPress Enter to continue...")
	fmt.Scanln()

	if err != nil {
		return actionResultMsg{err: err}
	}
	return actionResultMsg{message: "Logs viewed"}
}

func (m EnhancedPodActionsModel) followLogs() tea.Msg {
	fmt.Print("\033[H\033[2J") // Clear screen
	pterm.DefaultHeader.Printf("Following Logs: %s (Press Ctrl+C to stop)\n", m.pod.Name)

	cmd := exec.Command("kubectl", "logs", "-f", m.pod.Name, "-n", m.pod.Namespace)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()

	if err != nil {
		return actionResultMsg{err: err}
	}
	return actionResultMsg{message: "Log follow completed"}
}

func (m EnhancedPodActionsModel) execShell() tea.Msg {
	fmt.Print("\033[H\033[2J") // Clear screen

	// Check if pod has multiple containers
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pod, err := m.client.Clientset.CoreV1().Pods(m.pod.Namespace).Get(ctx, m.pod.Name, metav1.GetOptions{})
	if err != nil {
		return actionResultMsg{err: err}
	}

	containerName := ""
	if len(pod.Spec.Containers) > 1 {
		// Let user select container
		pterm.DefaultHeader.Printf("Pod has %d containers:\n", len(pod.Spec.Containers))
		options := []string{}
		for _, c := range pod.Spec.Containers {
			options = append(options, c.Name)
		}

		selectedOption, _ := pterm.DefaultInteractiveSelect.
			WithOptions(options).
			WithDefaultText("Select container").
			Show()

		containerName = selectedOption
	} else if len(pod.Spec.Containers) == 1 {
		containerName = pod.Spec.Containers[0].Name
	}

	pterm.DefaultHeader.Printf("Executing shell in pod: %s\n", m.pod.Name)
	pterm.Info.Println("Type 'exit' to leave the shell")

	// Use kubectl exec
	args := []string{"exec", "-it", m.pod.Name, "-n", m.pod.Namespace}
	if containerName != "" {
		args = append(args, "-c", containerName)
	}
	args = append(args, "--", "/bin/bash")

	cmd := exec.Command("kubectl", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Try bash first, fall back to sh
	if err := cmd.Run(); err != nil {
		args[len(args)-1] = "/bin/sh"
		cmd = exec.Command("kubectl", args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			return actionResultMsg{err: err}
		}
	}

	return actionResultMsg{message: "Shell session ended"}
}

func (m EnhancedPodActionsModel) portForward() tea.Msg {
	fmt.Print("\033[H\033[2J") // Clear screen
	pterm.DefaultHeader.Printf("Port Forwarding: %s\n", m.pod.Name)

	// Get port information
	localPort := ""
	podPort := ""

	localPortInput := pterm.DefaultInteractiveTextInput.
		WithDefaultText("Enter local port (e.g., 8080)")
	localPort, _ = localPortInput.Show()

	podPortInput := pterm.DefaultInteractiveTextInput.
		WithDefaultText("Enter pod port (e.g., 80)")
	podPort, _ = podPortInput.Show()

	if localPort == "" || podPort == "" {
		return actionResultMsg{err: fmt.Errorf("invalid port configuration")}
	}

	pterm.Info.Printf("Port forwarding %s:%s -> %s:%s\n", "localhost", localPort, m.pod.Name, podPort)
	pterm.Info.Println("Press Ctrl+C to stop port forwarding")

	// Use kubectl port-forward
	cmd := exec.Command("kubectl", "port-forward",
		fmt.Sprintf("pod/%s", m.pod.Name),
		fmt.Sprintf("%s:%s", localPort, podPort),
		"-n", m.pod.Namespace)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()

	if err != nil {
		return actionResultMsg{err: err}
	}
	return actionResultMsg{message: "Port forwarding stopped"}
}

func (m EnhancedPodActionsModel) resourceUsage() tea.Msg {
	fmt.Print("\033[H\033[2J") // Clear screen
	pterm.DefaultHeader.Printf("Resource Usage: %s\n", m.pod.Name)

	cmd := exec.Command("kubectl", "top", "pod", m.pod.Name, "-n", m.pod.Namespace, "--containers")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		pterm.Warning.Println("Note: Metrics server might not be installed in the cluster")
		fmt.Println("\nPress Enter to continue...")
		fmt.Scanln()
		return actionResultMsg{err: err}
	}

	fmt.Println("\nPress Enter to continue...")
	fmt.Scanln()
	return actionResultMsg{message: "Resource usage displayed"}
}

func (m EnhancedPodActionsModel) editPod() tea.Msg {
	fmt.Print("\033[H\033[2J") // Clear screen
	pterm.DefaultHeader.Printf("Editing Pod: %s\n", m.pod.Name)

	cmd := exec.Command("kubectl", "edit", "pod", m.pod.Name, "-n", m.pod.Namespace)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return actionResultMsg{err: err}
	}
	return actionResultMsg{message: "Pod edit completed"}
}

func (m EnhancedPodActionsModel) restartPod() tea.Msg {
	// Confirm before restarting
	result, _ := pterm.DefaultInteractiveConfirm.
		WithDefaultText(fmt.Sprintf("Are you sure you want to restart pod '%s'?", m.pod.Name)).
		WithTextStyle(pterm.NewStyle(pterm.FgYellow)).
		Show()

	if !result {
		return actionResultMsg{message: "Restart cancelled"}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	gracePeriod := int64(30)
	err := m.client.Clientset.CoreV1().Pods(m.pod.Namespace).Delete(ctx, m.pod.Name, metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
	})

	if err != nil {
		return actionResultMsg{err: err}
	}
	return actionResultMsg{message: fmt.Sprintf("Pod %s restarted successfully", m.pod.Name)}
}

func (m EnhancedPodActionsModel) deletePod() tea.Msg {
	// Confirm before deleting
	result, _ := pterm.DefaultInteractiveConfirm.
		WithDefaultText(fmt.Sprintf("Are you sure you want to DELETE pod '%s'?", m.pod.Name)).
		WithTextStyle(pterm.NewStyle(pterm.FgRed)).
		Show()

	if !result {
		return actionResultMsg{message: "Deletion cancelled"}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	gracePeriod := int64(30)
	err := m.client.Clientset.CoreV1().Pods(m.pod.Namespace).Delete(ctx, m.pod.Name, metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
	})

	if err != nil {
		return actionResultMsg{err: err}
	}
	return actionResultMsg{message: fmt.Sprintf("Pod %s deleted successfully", m.pod.Name)}
}

// Helper function to format ports
func formatPorts(ports []corev1.ContainerPort) string {
	if len(ports) == 0 {
		return "None"
	}
	portStrs := []string{}
	for _, p := range ports {
		portStrs = append(portStrs, fmt.Sprintf("%d/%s", p.ContainerPort, p.Protocol))
	}
	return strings.Join(portStrs, ", ")
}