package services

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// SQLProxyConnection represents an active Cloud SQL proxy connection
type SQLProxyConnection struct {
	InstanceName   string
	LocalPort      int
	Status         string
	StartedAt      time.Time
	ErrorMessage   string
	ConnectionName string
	LogFilePath    string
	cmd            *exec.Cmd
	cancel         context.CancelFunc
	logFile        *os.File
}

// SQLProxyManager manages Cloud SQL proxy connections
type SQLProxyManager struct {
	connections map[string]*SQLProxyConnection
	mu          sync.RWMutex
}

var (
	proxyManager     *SQLProxyManager
	proxyManagerOnce sync.Once
)

// GetSQLProxyManager returns the singleton SQL proxy manager
func GetSQLProxyManager() *SQLProxyManager {
	proxyManagerOnce.Do(func() {
		proxyManager = &SQLProxyManager{
			connections: make(map[string]*SQLProxyConnection),
		}
	})
	return proxyManager
}

// isPortAvailable checks if a port is available for use
func isPortAvailable(port int) bool {
	address := fmt.Sprintf("127.0.0.1:%d", port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

// ListInstances lists available Cloud SQL instances
func (m *SQLProxyManager) ListInstances(projectID string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "gcloud", "sql", "instances", "list",
		"--project", projectID,
		"--format", "value(name)")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list SQL instances: %w", err)
	}

	instances := strings.Split(strings.TrimSpace(string(output)), "\n")
	var result []string
	for _, inst := range instances {
		if inst != "" {
			result = append(result, inst)
		}
	}

	return result, nil
}

// GetInstanceConnectionName returns the full connection name for an instance
func (m *SQLProxyManager) GetInstanceConnectionName(projectID, region, instanceName string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "gcloud", "sql", "instances", "describe", instanceName,
		"--project", projectID,
		"--format", "value(connectionName)")

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get instance connection name: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// StartProxy starts a Cloud SQL proxy connection
func (m *SQLProxyManager) StartProxy(projectID, region, instanceName string, localPort int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already connected
	if conn, exists := m.connections[instanceName]; exists {
		if conn.Status == "running" {
			return fmt.Errorf("proxy already running for instance %s on port %d", instanceName, conn.LocalPort)
		}
	}

	// Check if port is available
	if !isPortAvailable(localPort) {
		return fmt.Errorf("port %d is already in use. Please stop any services using this port or the proxy will use the next available port", localPort)
	}

	// Get full connection name
	connectionName, err := m.GetInstanceConnectionName(projectID, region, instanceName)
	if err != nil {
		return err
	}

	// Find cloud-sql-proxy binary (try multiple names and locations)
	var proxyPath string
	possiblePaths := []string{
		"cloud-sql-proxy",     // In PATH with hyphens
		"cloud_sql_proxy",     // In PATH with underscores (old name)
		"/opt/homebrew/share/google-cloud-sdk/bin/cloud-sql-proxy",  // Homebrew macOS
		"/usr/local/google-cloud-sdk/bin/cloud-sql-proxy",           // Standard macOS
		"/usr/bin/cloud-sql-proxy",                                   // Linux system
		"/usr/local/bin/cloud-sql-proxy",                             // Linux local
	}

	for _, path := range possiblePaths {
		if resolvedPath, err := exec.LookPath(path); err == nil {
			proxyPath = resolvedPath
			break
		}
	}

	if proxyPath == "" {
		return fmt.Errorf("cloud-sql-proxy not found. Install it with: gcloud components install cloud-sql-proxy\nThen add it to PATH or restart your terminal")
	}

	// Create context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())

	// Build cloud-sql-proxy command with proper format
	// Using global --port flag and --private-ip for VPC-only instances
	// Format: cloud-sql-proxy CONNECTION_NAME --port PORT --private-ip --debug-logs
	cmd := exec.CommandContext(ctx, proxyPath,
		connectionName,
		"--port", fmt.Sprintf("%d", localPort),
		"--private-ip",
		"--debug-logs")

	// Create log file for this instance
	logFile, err := os.CreateTemp("", fmt.Sprintf("cloud-sql-proxy-%s-*.log", instanceName))
	if err != nil {
		cancel()
		return fmt.Errorf("failed to create log file: %w", err)
	}

	// Capture stdout and stderr both to log file and to strings
	var stdout, stderr strings.Builder
	cmd.Stdout = io.MultiWriter(&stdout, logFile)
	cmd.Stderr = io.MultiWriter(&stderr, logFile)

	fmt.Fprintf(logFile, "Starting cloud-sql-proxy for %s on port %d\n", connectionName, localPort)
	fmt.Fprintf(logFile, "Command: %s %s --port %d --private-ip --debug-logs\n", proxyPath, connectionName, localPort)
	fmt.Fprintf(logFile, "----------------------------------------\n")

	// Start the proxy in the background
	if err := cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("failed to start cloud-sql-proxy: %w", err)
	}

	// Give the proxy a moment to start and check for immediate errors
	time.Sleep(2 * time.Second)

	// Check if the process is still running
	if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
		cancel()
		stderrOutput := stderr.String()
		stdoutOutput := stdout.String()
		errorMsg := "cloud-sql-proxy failed to start"
		if stderrOutput != "" {
			errorMsg += fmt.Sprintf("\nStderr: %s", stderrOutput)
		}
		if stdoutOutput != "" {
			errorMsg += fmt.Sprintf("\nStdout: %s", stdoutOutput)
		}
		return fmt.Errorf("%s", errorMsg)
	}

	// Store connection info
	m.connections[instanceName] = &SQLProxyConnection{
		InstanceName:   instanceName,
		LocalPort:      localPort,
		Status:         "running",
		StartedAt:      time.Now(),
		ConnectionName: connectionName,
		LogFilePath:    logFile.Name(),
		cmd:            cmd,
		cancel:         cancel,
		logFile:        logFile,
	}

	// Monitor the process
	go func() {
		err := cmd.Wait()
		m.mu.Lock()
		defer m.mu.Unlock()
		if conn, exists := m.connections[instanceName]; exists {
			// Close log file
			if conn.logFile != nil {
				conn.logFile.Close()
			}

			if err != nil {
				stderrOutput := stderr.String()
				stdoutOutput := stdout.String()
				errorMsg := fmt.Sprintf("Process exited with error: %v", err)
				if stderrOutput != "" {
					errorMsg += fmt.Sprintf("\nStderr: %s", stderrOutput)
				}
				if stdoutOutput != "" {
					errorMsg += fmt.Sprintf("\nStdout: %s", stdoutOutput)
				}
				errorMsg += fmt.Sprintf("\nFull logs: %s", conn.LogFilePath)
				conn.Status = "failed"
				conn.ErrorMessage = errorMsg
			} else {
				conn.Status = "stopped"
			}
		}
	}()

	return nil
}

// StopProxy stops a Cloud SQL proxy connection
func (m *SQLProxyManager) StopProxy(instanceName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	conn, exists := m.connections[instanceName]
	if !exists {
		return fmt.Errorf("no proxy running for instance %s", instanceName)
	}

	if conn.cancel != nil {
		conn.cancel()
	}

	if conn.cmd != nil && conn.cmd.Process != nil {
		if err := conn.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill proxy process: %w", err)
		}
	}

	// Close log file
	if conn.logFile != nil {
		conn.logFile.Close()
	}

	conn.Status = "stopped"
	return nil
}

// GetConnections returns all active connections
func (m *SQLProxyManager) GetConnections() []*SQLProxyConnection {
	m.mu.RLock()
	defer m.mu.RUnlock()

	connections := make([]*SQLProxyConnection, 0, len(m.connections))
	for _, conn := range m.connections {
		connections = append(connections, conn)
	}
	return connections
}

// GetConnection returns a specific connection
func (m *SQLProxyManager) GetConnection(instanceName string) (*SQLProxyConnection, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	conn, exists := m.connections[instanceName]
	return conn, exists
}

// StopAll stops all proxy connections
func (m *SQLProxyManager) StopAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errors []string
	for instanceName, conn := range m.connections {
		if conn.Status == "running" {
			if conn.cancel != nil {
				conn.cancel()
			}
			if conn.cmd != nil && conn.cmd.Process != nil {
				if err := conn.cmd.Process.Kill(); err != nil {
					errors = append(errors, fmt.Sprintf("%s: %v", instanceName, err))
				}
			}
			// Close log file
			if conn.logFile != nil {
				conn.logFile.Close()
			}
			conn.Status = "stopped"
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors stopping proxies: %s", strings.Join(errors, "; "))
	}

	return nil
}

// OpenInTablePlus opens a TablePlus connection URL
func (m *SQLProxyManager) OpenInTablePlus(instanceName string, port int) error {
	conn, exists := m.GetConnection(instanceName)
	if !exists {
		return fmt.Errorf("no connection found for instance %s", instanceName)
	}

	if conn.Status != "running" {
		return fmt.Errorf("proxy is not running for instance %s", instanceName)
	}

	// TablePlus URL format: mysql://host:port/database?name=connectionName
	// Using root as default user - users can modify credentials in TablePlus
	// Database is optional, leaving it empty
	tablePlusURL := fmt.Sprintf("mysql://root@127.0.0.1:%d/?name=%s", port, instanceName)

	// Try to open with TablePlus
	cmd := exec.Command("open", tablePlusURL)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to open TablePlus: %w. Make sure TablePlus is installed", err)
	}

	return nil
}
