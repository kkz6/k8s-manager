# K8s Manager

<div align="center">
A comprehensive CLI tool for managing Kubernetes clusters on Google Cloud Platform
<br>
<br>
K8s Manager simplifies Kubernetes operations by providing an intuitive command-line interface for common tasks like configuration management, secrets handling, pod operations, log viewing, and remote command execution.
<br>
<br>
<img src="https://img.shields.io/badge/Go-1.19+-blue.svg" alt="Go Version"/>
<img src="https://img.shields.io/badge/Platform-GCP-orange.svg" alt="GCP Platform"/>
<img src="https://img.shields.io/badge/License-MIT-green.svg" alt="License"/>
</div>

# Table of Contents

<!--ts-->

- [K8s Manager](#k8s-manager)
- [Features](#features)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Commands](#commands)
- [Configuration](#configuration)
- [Examples](#examples)
- [Contributing](#contributing)

<!--te-->

# Features

🚀 **Easy Setup**: Interactive configuration with automatic GCP and Kubernetes integration  
🔐 **Secrets Management**: Create, view, update, and delete Kubernetes secrets with ease  
☸️ **Pod Operations**: List, describe, restart, and delete pods with comprehensive filtering  
🔗 **Pod Access**: Direct SSH access to pods with automatic container selection  
📊 **Log Viewing**: Stream and follow pod logs with flexible filtering options  
⚡ **Command Execution**: Run commands on pods interactively or non-interactively  
🌐 **GCP Integration**: Seamless authentication using gcloud CLI  
🎯 **Multi-Namespace**: Support for operations across multiple namespaces

# Prerequisites

Before using K8s Manager, ensure you have:

- **Go 1.19+** installed
- **gcloud CLI** installed and authenticated (`gcloud auth login`)
- **kubectl** installed and configured
- Access to a **Google Kubernetes Engine (GKE)** cluster
- Appropriate **IAM permissions** for your GCP project

# Installation

## From Source

```bash
git clone https://github.com/karthickk/k8s-manager.git
cd k8s-manager
go build -o k8s-manager .
sudo mv k8s-manager /usr/local/bin/
```

## Using Go Install

```bash
go install github.com/karthickk/k8s-manager@latest
```

# Quick Start

1. **Initialize Configuration**

   ```bash
   k8s-manager config init
   ```

2. **Validate Setup**

   ```bash
   k8s-manager config validate
   ```

3. **List Pods**

   ```bash
   k8s-manager pods list
   ```

4. **View Pod Logs**

   ```bash
   k8s-manager logs <pod-name> -f
   ```

5. **SSH into Pod**
   ```bash
   k8s-manager pods ssh <pod-name>
   ```

# Commands

## Configuration Management

```bash
k8s-manager config init        # Interactive configuration setup
k8s-manager config show        # Display current configuration
k8s-manager config set <key> <value>  # Set configuration values
k8s-manager config validate    # Validate configuration and connectivity
```

## Secrets Management

```bash
k8s-manager secrets list                    # List all secrets
k8s-manager secrets get <secret-name>       # Get secret details
k8s-manager secrets create <name> -l key=value  # Create secret
k8s-manager secrets update <name> -l key=value  # Update secret
k8s-manager secrets delete <name>           # Delete secret
k8s-manager secrets decode <name> <key>     # Decode secret value
```

## Pod Management

```bash
k8s-manager pods list                 # List pods
k8s-manager pods list -A              # List pods in all namespaces
k8s-manager pods get <pod-name>       # Get pod details
k8s-manager pods restart <pod-name>   # Restart pod
k8s-manager pods delete <pod-name>    # Delete pod
k8s-manager pods ssh <pod-name>       # SSH into pod
```

## Log Viewing

```bash
k8s-manager logs <pod-name>           # View pod logs
k8s-manager logs <pod-name> -f        # Follow pod logs
k8s-manager logs <pod-name> --tail 100  # Last 100 lines
k8s-manager logs <pod-name> --since 1h  # Logs from last hour
```

## Command Execution

```bash
k8s-manager exec shell <pod-name>          # Interactive shell
k8s-manager exec run <pod-name> -- ls -la  # Run specific command
k8s-manager exec run <pod-name> -it -- bash  # Interactive command
```

# Configuration

K8s Manager stores configuration in `~/.config/k8s-manager/k8s-manager.yaml`:

```yaml
gcp:
  project_id: "my-gcp-project"
  zone: "us-central1-a"
  region: "us-central1"

k8s:
  cluster_name: "my-cluster"
  namespace: "default"
  config_path: "~/.kube/config"

ssh:
  username: "root"
  port: 22

log_level: "info"
```

## Environment Variables

You can also configure using environment variables with the `K8S_MANAGER_` prefix:

```bash
export K8S_MANAGER_GCP_PROJECT_ID="my-project"
export K8S_MANAGER_K8S_CLUSTER_NAME="my-cluster"
export K8S_MANAGER_K8S_NAMESPACE="production"
```

# Examples

## Complete Workflow Example

```bash
# 1. Initialize configuration
k8s-manager config init

# 2. List all pods in current namespace
k8s-manager pods list

# 3. Get detailed information about a specific pod
k8s-manager pods get my-app-pod-123

# 4. View live logs from the pod
k8s-manager logs my-app-pod-123 -f

# 5. SSH into the pod for debugging
k8s-manager pods ssh my-app-pod-123

# 6. Run a specific command on the pod
k8s-manager exec run my-app-pod-123 -- cat /etc/hostname

# 7. Create a new secret
k8s-manager secrets create db-credentials \
  -l username=admin \
  -l password=secret123

# 8. View the secret (encoded)
k8s-manager secrets get db-credentials

# 9. Decode a specific key from the secret
k8s-manager secrets decode db-credentials username
```

## Advanced Usage

```bash
# List pods across all namespaces with labels
k8s-manager pods list -A --show-labels

# Filter pods by label selector
k8s-manager pods list -l app=nginx

# Restart a deployment instead of individual pod
k8s-manager pods restart my-deployment --deployment

# Follow logs with timestamps since 30 minutes ago
k8s-manager logs my-pod --timestamps --since 30m -f

# Create secret from file
k8s-manager secrets create tls-cert \
  -f tls.crt=./cert.pem \
  -f tls.key=./key.pem \
  --type kubernetes.io/tls

# Execute command on specific container in multi-container pod
k8s-manager exec run my-pod -c sidecar -- ps aux
```

# Authentication

K8s Manager uses **gcloud CLI** for authentication, which provides:

- ✅ Seamless integration with GCP services
- ✅ Automatic token refresh
- ✅ Support for service accounts
- ✅ Multi-project support
- ✅ Security best practices

Make sure you're authenticated with gcloud:

```bash
gcloud auth login
gcloud config set project YOUR_PROJECT_ID
```

# Troubleshooting

## Common Issues

**"gcloud not found"**

```bash
# Install Google Cloud SDK
curl https://sdk.cloud.google.com | bash
exec -l $SHELL
```

**"No active gcloud authentication"**

```bash
gcloud auth login
gcloud auth application-default login
```

**"kubectl not found"**

```bash
# Install kubectl
gcloud components install kubectl
```

**"Failed to connect to cluster"**

```bash
# Get cluster credentials
gcloud container clusters get-credentials CLUSTER_NAME --zone=ZONE
```

## Debug Mode

Enable debug logging:

```bash
k8s-manager config set log_level debug
```

# Contributing

We welcome contributions! Please see our contributing guidelines:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Development Setup

```bash
git clone https://github.com/karthickk/k8s-manager.git
cd k8s-manager
go mod tidy
go build -o k8s-manager .
```

## Running Tests

```bash
go test ./...
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

<div align="center">
Made with ❤️ for Kubernetes developers working with GCP
</div>
