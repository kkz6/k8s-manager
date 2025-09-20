# K8s Manager - New Features

## Secret Management
- **View Secrets**: Browse all secrets with filtering and instant number selection
- **Create Secrets**: Step-by-step wizard to create new secrets with custom key-value pairs
- **Edit Secrets**: Full editor to add, modify, or delete key-value pairs in existing secrets
- **Export Secrets**: Export as environment variables or YAML format
- **Secret Types**: Support for Opaque, Docker Config, and TLS secret types

## Pod Environment Management
- **View Environment Variables**: Display all environment variables in a pod
- **Assign Environment Variables**:
  - Select from existing Secrets or ConfigMaps
  - Choose specific keys to expose as environment variables
  - Automatic environment variable naming (converts keys to uppercase)
  - Preview changes before applying
- **Environment Sources**:
  - Kubernetes Secrets
  - ConfigMaps
  - Direct values (coming soon)

## UI Features
- **DevTools-style Interface**: Clean, minimalist design inspired by browser DevTools
- **Instant Number Selection**: Press 1-9 to instantly select options (no Enter key needed)
- **Consistent Navigation**:
  - Number keys (1-9) for quick selection
  - Arrow keys or j/k for navigation
  - 0 or b to go back
  - q or Esc to quit
- **Visual Feedback**: Success/error/warning messages with color coding
- **Filtering**: Type `/` to filter lists in real-time

## How to Use

### Managing Secrets
1. From main menu, press `4` for "ConfigMaps & Secrets"
2. Select namespace or view all
3. Press number to select a secret or `9` to create new
4. For existing secrets:
   - Press `1` to view data
   - Press `2` to edit (add/modify/delete keys)
   - Press `5` to export as environment variables

### Assigning Environment to Pods
1. From main menu, press `1` for "Pods Manager"
2. Select namespace and pod
3. Press `8` for "Environment Variables"
4. Press `2` for "Assign Environment Variables"
5. Choose source (Secret/ConfigMap)
6. Select the resource
7. Choose which keys to expose
8. Review and apply changes

## Technical Implementation
- Built with Bubble Tea TUI framework
- Uses Kubernetes client-go for API operations
- Modular component architecture
- Real-time data updates
- Context-aware help text