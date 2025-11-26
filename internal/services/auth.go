package services

import (
	"os/exec"
	"strings"
)

// AuthProvider represents different cloud providers
type AuthProvider string

const (
	GCP   AuthProvider = "gcp"
	AWS   AuthProvider = "aws"
	Azure AuthProvider = "azure"
)

// CheckGCPAuth checks if GCP credentials are valid
func CheckGCPAuth() (bool, error) {
	cmd := exec.Command("gcloud", "auth", "list", "--format=json")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	return len(output) > 2 && !strings.Contains(string(output), "[]"), nil
}

// LoginGCP initiates GCP login flow
func LoginGCP() error {
	cmd := exec.Command("gcloud", "auth", "login")
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Start()
}

// GetAuthProvider detects which auth provider is in use
func GetAuthProvider(errorMsg string) AuthProvider {
	errorLower := strings.ToLower(errorMsg)

	if strings.Contains(errorLower, "gcloud") ||
	   strings.Contains(errorLower, "reauthentication failed") ||
	   strings.Contains(errorLower, "refreshing your current auth tokens") {
		return GCP
	}

	if strings.Contains(errorLower, "aws") || strings.Contains(errorLower, "eks") {
		return AWS
	}

	if strings.Contains(errorLower, "azure") || strings.Contains(errorLower, "aks") {
		return Azure
	}

	return ""
}

// GetAuthHelp returns helpful auth instructions based on the error
func GetAuthHelp(errorMsg string) string {
	provider := GetAuthProvider(errorMsg)

	switch provider {
	case GCP:
		return `Your GCP authentication has expired.

To fix this, run one of these commands:

  1. Login to GCP:
     $ gcloud auth login

  2. Or use application-default credentials:
     $ gcloud auth application-default login

  3. Or switch to a different account:
     $ gcloud config set account ACCOUNT

After authenticating, restart k8s-manager.

Press 'l' to run 'gcloud auth login' now
Press 'r' to retry
Press 'q' to quit`

	case AWS:
		return `Your AWS authentication may have expired.

To fix this, run:
  $ aws configure

Or ensure your AWS credentials are set:
  $ export AWS_ACCESS_KEY_ID=your-key
  $ export AWS_SECRET_ACCESS_KEY=your-secret

Press 'r' to retry
Press 'q' to quit`

	case Azure:
		return `Your Azure authentication may have expired.

To fix this, run:
  $ az login

Press 'r' to retry
Press 'q' to quit`

	default:
		return `Authentication error detected.

Please ensure you are authenticated to your Kubernetes cluster.

Press 'r' to retry
Press 'q' to quit`
	}
}

// IsAuthError checks if an error is authentication-related
func IsAuthError(err error) bool {
	if err == nil {
		return false
	}

	errorMsg := strings.ToLower(err.Error())
	authKeywords := []string{
		"auth",
		"authentication",
		"credential",
		"unauthorized",
		"forbidden",
		"reauthentication",
		"token",
		"login",
	}

	for _, keyword := range authKeywords {
		if strings.Contains(errorMsg, keyword) {
			return true
		}
	}

	return false
}
