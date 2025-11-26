package views

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/karthickk/k8s-manager/internal/ui/components"
	"github.com/karthickk/k8s-manager/pkg/config"
)

// ConfigViewModel displays and manages configuration
type ConfigViewModel struct {
	cfg        *config.Config
	configPath string
	rawYAML    string
	loaded     bool
	err        error
}

// NewConfigViewModel creates a new configuration view model
func NewConfigViewModel() tea.Model {
	return &ConfigViewModel{}
}

func (m *ConfigViewModel) Init() tea.Cmd {
	return m.loadConfig
}

func (m *ConfigViewModel) loadConfig() tea.Msg {
	cfg, err := config.Load()
	if err != nil {
		return configLoadErrorMsg{err: err}
	}

	// Determine config file path
	configPath := "k8s-manager.yaml" // Current directory
	homeDir := os.Getenv("HOME")
	if homeDir != "" {
		userConfigPath := filepath.Join(homeDir, ".config", "k8s-manager", "k8s-manager.yaml")
		if _, err := os.Stat(userConfigPath); err == nil {
			configPath = userConfigPath
		}
	}

	// Read raw YAML
	rawYAML := ""
	if data, err := os.ReadFile(configPath); err == nil {
		rawYAML = string(data)
	}

	return configLoadedMsg{
		cfg:        cfg,
		configPath: configPath,
		rawYAML:    rawYAML,
	}
}

type configLoadedMsg struct {
	cfg        *config.Config
	configPath string
	rawYAML    string
}

type configLoadErrorMsg struct {
	err error
}

func (m *ConfigViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case configLoadedMsg:
		m.cfg = msg.cfg
		m.configPath = msg.configPath
		m.rawYAML = msg.rawYAML
		m.loaded = true
		return m, nil

	case configLoadErrorMsg:
		m.err = msg.err
		m.loaded = true
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q", "esc", "b":
			return m, Navigate(ViewMainMenu, nil)
		case "r":
			m.loaded = false
			m.err = nil
			return m, m.loadConfig
		}
	}

	return m, nil
}

func (m *ConfigViewModel) View() string {
	title := components.RenderTitle("⚙️  Configuration", "K8s Manager Settings")

	if m.err != nil {
		errorMsg := fmt.Sprintf("\n  ❌ Error: %s\n\n", m.err.Error())
		help := "  " + components.HelpStyle.Render("r: Retry • q/b/esc: Back • ctrl+c: Quit")
		return title + errorMsg + help
	}

	// If config not loaded yet, show loading
	if !m.loaded || m.cfg == nil {
		return title + "\n\n  Loading configuration...\n\n  " +
			components.HelpStyle.Render("q/b/esc: Back • ctrl+c: Quit")
	}

	// Render the config content
	content := m.renderConfigContent()

	help := "\n  " + components.HelpStyle.Render("r: Reload • q/b/esc: Back • ctrl+c: Quit")

	return title + content + help
}

func (m *ConfigViewModel) renderConfigContent() string {
	var b strings.Builder

	b.WriteString("\n")

	// Config file location
	b.WriteString("  📁 Configuration File\n")
	b.WriteString("     " + components.DescriptionStyle.Render(m.configPath) + "\n\n")

	// GCP Settings
	b.WriteString("  🌐 GCP Settings\n")
	b.WriteString(fmt.Sprintf("     Project ID: %s\n", m.formatValue(m.cfg.GCP.ProjectID)))
	b.WriteString(fmt.Sprintf("     Region:     %s\n", m.formatValue(m.cfg.GCP.Region)))
	b.WriteString(fmt.Sprintf("     Zone:       %s\n", m.formatValue(m.cfg.GCP.Zone)))
	b.WriteString("\n")

	// Kubernetes Settings
	b.WriteString("  ☸️  Kubernetes Settings\n")
	b.WriteString(fmt.Sprintf("     Cluster:    %s\n", m.formatValue(m.cfg.K8s.ClusterName)))
	b.WriteString(fmt.Sprintf("     Namespace:  %s\n", m.formatValue(m.cfg.K8s.Namespace)))
	if m.cfg.K8s.Context != "" {
		b.WriteString(fmt.Sprintf("     Context:    %s\n", m.formatValue(m.cfg.K8s.Context)))
	}
	b.WriteString("\n")

	// SSH Settings
	b.WriteString("  🔐 SSH Settings\n")
	b.WriteString(fmt.Sprintf("     Username:   %s\n", m.formatValue(m.cfg.SSH.Username)))
	b.WriteString(fmt.Sprintf("     Port:       %d\n", m.cfg.SSH.Port))
	if m.cfg.SSH.KeyPath != "" {
		b.WriteString(fmt.Sprintf("     Key Path:   %s\n", m.formatValue(m.cfg.SSH.KeyPath)))
	}
	b.WriteString("\n")

	// Other Settings
	b.WriteString("  📊 Other Settings\n")
	b.WriteString(fmt.Sprintf("     Log Level:  %s\n", m.formatValue(m.cfg.LogLevel)))
	b.WriteString("\n")

	// Instructions
	b.WriteString("  💡 How to Edit Configuration\n")
	b.WriteString("     • Edit YAML file: " + m.configPath + "\n")
	b.WriteString("     • Or use CLI: k8s-manager config set <key> <value>\n")
	b.WriteString("     • Press 'r' to reload after making changes\n")

	return b.String()
}

func (m *ConfigViewModel) formatValue(value string) string {
	if value == "" {
		return components.DescriptionStyle.Render("(not set)")
	}
	return components.StatusRunningStyle.Render(value)
}
