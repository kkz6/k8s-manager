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

var (
	actionMenuStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 3).
			Margin(1, 2)

	actionTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")).
			Background(lipgloss.Color("235")).
			Padding(0, 2).
			MarginBottom(1)

	podInfoStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 2).
			MarginBottom(1)

	actionSectionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true).
			MarginTop(1).
			MarginBottom(1)

	actionItemStyle = lipgloss.NewStyle().
			PaddingLeft(4).
			PaddingTop(0).
			PaddingBottom(0)

	actionSelectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("170")).
			Background(lipgloss.Color("236")).
			Bold(true).
			PaddingLeft(2).
			PaddingRight(2)

	actionDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			PaddingLeft(6).
			PaddingTop(0)

	statusGoodStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Bold(true)

	statusBadStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	statusWarningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true)
)

type PodActionsModel struct {
	pod          PodInfo
	podDetails   *corev1.Pod
	actions      []PodAction
	cursor       int
	width        int
	height       int
	quitting     bool
	loading      bool
	executing    bool
	message      string
	client       *k8s.Client
	spinner      spinner.Model
	initialized  bool
}

type PodAction struct {
	Name        string
	Description string
	Key         string
	Handler     func(pod PodInfo, client *k8s.Client) error
}

func NewPodActionsModel(pod PodInfo, client *k8s.Client) PodActionsModel {
	actions := []PodAction{
		{
			Name:        "📋  Describe",
			Description: "Show detailed pod information",
			Key:         "describe",
			Handler:     describePod,
		},
		{
			Name:        "📝  View Logs",
			Description: "View recent pod logs",
			Key:         "logs",
			Handler:     viewPodLogs,
		},
		{
			Name:        "📊  Follow Logs",
			Description: "Follow pod logs in real-time",
			Key:         "follow",
			Handler:     followPodLogs,
		},
		{
			Name:        "⚡  Execute Shell",
			Description: "Open interactive shell in pod",
			Key:         "exec",
			Handler:     execIntoPod,
		},
		{
			Name:        "🔄  Restart Pod",
			Description: "Delete pod to trigger restart",
			Key:         "restart",
			Handler:     restartPod,
		},
		{
			Name:        "🗑️   Delete Pod",
			Description: "Permanently delete the pod",
			Key:         "delete",
			Handler:     deletePod,
		},
		{
			Name:        "📦  Port Forward",
			Description: "Forward local port to pod",
			Key:         "port-forward",
			Handler:     portForwardPod,
		},
		{
			Name:        "📈  Resource Usage",
			Description: "Show CPU and memory usage",
			Key:         "resources",
			Handler:     showResourceUsage,
		},
		{
			Name:        "↩️   Back",
			Description: "Return to pods list",
			Key:         "back",
			Handler:     nil,
		},
	}

	// Create spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return PodActionsModel{
		pod:     pod,
		actions: actions,
		client:  client,
		spinner: s,
		loading: true,
	}
}

func (m PodActionsModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.loadPodDetails,
	)
}

func (m PodActionsModel) loadPodDetails() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pod, err := m.client.Clientset.CoreV1().Pods(m.pod.Namespace).Get(ctx, m.pod.Name, metav1.GetOptions{})
	if err != nil {
		return podDetailsErrorMsg{err: err}
	}

	return podDetailsLoadedMsg{pod: pod}
}

type podDetailsLoadedMsg struct {
	pod *corev1.Pod
}

type podDetailsErrorMsg struct {
	err error
}

func (m PodActionsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.initialized = true
		return m, nil

	case podDetailsLoadedMsg:
		m.loading = false
		m.podDetails = msg.pod
		return m, nil

	case podDetailsErrorMsg:
		m.loading = false
		m.message = fmt.Sprintf("Error loading pod details: %v", msg.err)
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		if m.loading {
			// Don't process keys while loading
			return m, nil
		}
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.actions)-1 {
				m.cursor++
			}

		case "enter", " ":
			action := m.actions[m.cursor]
			if action.Key == "back" {
				m.quitting = true
				return m, tea.Quit
			}

			if action.Handler != nil {
				m.executing = true
				return m, m.executeAction(action)
			}
		}
	}

	return m, nil
}

func (m PodActionsModel) executeAction(action PodAction) tea.Cmd {
	return func() tea.Msg {
		// For actions that need to exit the TUI
		if action.Key == "exec" || action.Key == "logs" || action.Key == "follow" {
			// We'll handle these specially
			return tea.Quit
		}

		err := action.Handler(m.pod, m.client)
		if err != nil {
			m.message = fmt.Sprintf("Error: %v", err)
		} else {
			m.message = fmt.Sprintf("✅ %s completed successfully", action.Name)
		}
		return nil
	}
}

func (m PodActionsModel) View() string {
	if m.quitting {
		return ""
	}

	// Show loading spinner
	if m.loading {
		var s strings.Builder
		s.WriteString("\n\n")
		s.WriteString(lipgloss.NewStyle().
			Width(50).
			Align(lipgloss.Center).
			Render(fmt.Sprintf("%s Loading pod details...", m.spinner.View())))
		s.WriteString("\n\n")
		return s.String()
	}

	var content strings.Builder

	// Header with pod name
	header := fmt.Sprintf("🔧 Pod Actions: %s", m.pod.Name)
	content.WriteString(actionTitleStyle.Width(60).Align(lipgloss.Center).Render(header))
	content.WriteString("\n\n")

	// Pod Information Box
	var podInfo strings.Builder
	podInfo.WriteString(fmt.Sprintf("📦 Pod: %s\n", m.pod.Name))
	podInfo.WriteString(fmt.Sprintf("🏷️  Namespace: %s\n", m.pod.Namespace))

	// Status with color
	status := m.pod.Status
	var styledStatus string
	switch strings.ToLower(status) {
	case "running":
		styledStatus = statusGoodStyle.Render(status)
	case "pending", "terminating":
		styledStatus = statusWarningStyle.Render(status)
	case "failed", "error", "crashloopbackoff":
		styledStatus = statusBadStyle.Render(status)
	default:
		styledStatus = status
	}
	podInfo.WriteString(fmt.Sprintf("📊 Status: %s\n", styledStatus))

	// Ready status
	readyParts := strings.Split(m.pod.Ready, "/")
	if len(readyParts) == 2 && readyParts[0] == readyParts[1] {
		podInfo.WriteString(fmt.Sprintf("✅ Ready: %s\n", statusGoodStyle.Render(m.pod.Ready)))
	} else {
		podInfo.WriteString(fmt.Sprintf("⏳ Ready: %s\n", statusWarningStyle.Render(m.pod.Ready)))
	}

	podInfo.WriteString(fmt.Sprintf("🔄 Restarts: %d\n", m.pod.Restarts))
	podInfo.WriteString(fmt.Sprintf("🕒 Age: %s\n", m.pod.Age))

	if m.pod.Node != "" {
		podInfo.WriteString(fmt.Sprintf("🖥️  Node: %s\n", m.pod.Node))
	}

	// Add container info if available
	if m.podDetails != nil && len(m.podDetails.Spec.Containers) > 0 {
		podInfo.WriteString(fmt.Sprintf("📦 Containers: %d\n", len(m.podDetails.Spec.Containers)))
		if m.podDetails.Status.PodIP != "" {
			podInfo.WriteString(fmt.Sprintf("🌐 IP: %s\n", m.podDetails.Status.PodIP))
		}
	}

	content.WriteString(podInfoStyle.Width(60).Render(podInfo.String()))
	content.WriteString("\n")

	// Actions Section
	content.WriteString(actionSectionStyle.Render("Available Actions"))
	content.WriteString("\n\n")

	// Actions menu with better spacing
	for i, action := range m.actions {
		if i == m.cursor {
			// Selected item - show with background
			line := fmt.Sprintf("  ▸ %s", action.Name)
			content.WriteString(actionSelectedStyle.Width(56).Render(line))
			content.WriteString("\n")
			content.WriteString(actionDescStyle.Render(action.Description))
			content.WriteString("\n")
		} else {
			// Normal item
			line := fmt.Sprintf("    %s", action.Name)
			content.WriteString(actionItemStyle.Render(line))
			content.WriteString("\n")
		}

		// Add spacing between items
		if i < len(m.actions)-1 {
			content.WriteString("\n")
		}
	}

	// Message (if any)
	if m.message != "" {
		content.WriteString("\n\n")
		content.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("220")).
			Bold(true).
			Render(m.message))
	}

	// Help footer
	content.WriteString("\n\n")
	helpText := "↑/k up • ↓/j down • enter select • q/esc back"
	content.WriteString(lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Width(60).
		Align(lipgloss.Center).
		Render(helpText))

	// Wrap everything in the menu style
	return actionMenuStyle.MaxWidth(70).Render(content.String())
}

// Action handlers

func describePod(pod PodInfo, client *k8s.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	p, err := client.Clientset.CoreV1().Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// Display pod details
	pterm.DefaultSection.Println("Pod Details")

	data := [][]string{
		{"Name", p.Name},
		{"Namespace", p.Namespace},
		{"Status", string(p.Status.Phase)},
		{"Node", p.Spec.NodeName},
		{"IP", p.Status.PodIP},
		{"Created", p.CreationTimestamp.Format(time.RFC3339)},
	}

	pterm.DefaultTable.WithData(data).Render()

	// Show containers
	pterm.DefaultSection.Println("Containers")
	for _, container := range p.Spec.Containers {
		containerData := [][]string{
			{"Name", container.Name},
			{"Image", container.Image},
		}
		pterm.DefaultTable.WithData(containerData).Render()
	}

	fmt.Println("\nPress Enter to continue...")
	fmt.Scanln()

	return nil
}

func viewPodLogs(pod PodInfo, client *k8s.Client) error {
	// Use kubectl for simplicity
	cmd := exec.Command("kubectl", "logs", pod.Name, "-n", pod.Namespace, "--tail=100")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func followPodLogs(pod PodInfo, client *k8s.Client) error {
	// Use kubectl for following logs
	cmd := exec.Command("kubectl", "logs", "-f", pod.Name, "-n", pod.Namespace)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func execIntoPod(pod PodInfo, client *k8s.Client) error {
	// Check if pod has multiple containers
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	p, err := client.Clientset.CoreV1().Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	containerName := ""
	if len(p.Spec.Containers) > 1 {
		// Let user select container
		pterm.DefaultSection.Printf("Pod has %d containers:\n", len(p.Spec.Containers))
		for i, c := range p.Spec.Containers {
			fmt.Printf("  %d. %s\n", i+1, c.Name)
		}
		fmt.Print("Select container (1-", len(p.Spec.Containers), "): ")

		var choice int
		fmt.Scanln(&choice)
		if choice < 1 || choice > len(p.Spec.Containers) {
			return fmt.Errorf("invalid container selection")
		}
		containerName = p.Spec.Containers[choice-1].Name
	} else if len(p.Spec.Containers) == 1 {
		containerName = p.Spec.Containers[0].Name
	}

	// Use kubectl exec
	args := []string{"exec", "-it", pod.Name, "-n", pod.Namespace}
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
		return cmd.Run()
	}

	return nil
}

func restartPod(pod PodInfo, client *k8s.Client) error {
	// Confirm before restarting
	confirm := false
	prompt := pterm.InteractiveConfirmPrinter{
		DefaultText: fmt.Sprintf("Are you sure you want to restart pod '%s'?", pod.Name),
		TextStyle:   pterm.NewStyle(pterm.FgYellow),
	}
	confirm, _ = prompt.Show()

	if !confirm {
		return fmt.Errorf("restart cancelled")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	gracePeriod := int64(30)
	return client.Clientset.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
	})
}

func deletePod(pod PodInfo, client *k8s.Client) error {
	// Confirm before deleting
	confirm := false
	prompt := pterm.InteractiveConfirmPrinter{
		DefaultText: fmt.Sprintf("Are you sure you want to delete pod '%s'?", pod.Name),
		TextStyle:   pterm.NewStyle(pterm.FgRed),
	}
	confirm, _ = prompt.Show()

	if !confirm {
		return fmt.Errorf("deletion cancelled")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	gracePeriod := int64(30)
	return client.Clientset.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
	})
}

func portForwardPod(pod PodInfo, client *k8s.Client) error {
	// Get port from user
	fmt.Print("Enter local port: ")
	var localPort int
	fmt.Scanln(&localPort)

	fmt.Print("Enter pod port: ")
	var podPort int
	fmt.Scanln(&podPort)

	fmt.Printf("Port forwarding %d:%d for pod %s...\n", localPort, podPort, pod.Name)
	fmt.Println("Press Ctrl+C to stop port forwarding")

	// Use kubectl port-forward
	cmd := exec.Command("kubectl", "port-forward",
		fmt.Sprintf("pod/%s", pod.Name),
		fmt.Sprintf("%d:%d", localPort, podPort),
		"-n", pod.Namespace)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func showResourceUsage(pod PodInfo, client *k8s.Client) error {
	// Use kubectl top
	cmd := exec.Command("kubectl", "top", "pod", pod.Name, "-n", pod.Namespace, "--containers")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Println("Note: Metrics server might not be installed in the cluster")
		return err
	}

	fmt.Println("\nPress Enter to continue...")
	fmt.Scanln()
	return nil
}

// ShowPodActions displays the action menu for a pod
func ShowPodActions(pod PodInfo) error {
	client, err := k8s.NewClient()
	if err != nil {
		return err
	}

	m := NewPodActionsModel(pod, client)
	p := tea.NewProgram(m)

	_, err = p.Run()

	// Check if we need to execute special actions after quitting
	if m.cursor < len(m.actions) {
		action := m.actions[m.cursor]
		if action.Handler != nil && (action.Key == "exec" || action.Key == "logs" || action.Key == "follow") {
			return action.Handler(pod, client)
		}
	}

	return err
}