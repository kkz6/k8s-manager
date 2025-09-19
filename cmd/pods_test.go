package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestPodsCommand(t *testing.T) {
	testCases := []struct {
		name     string
		args     []string
		wantErr  bool
		contains []string
	}{
		{
			name:    "pods help",
			args:    []string{"pods", "--help"},
			wantErr: false,
			contains: []string{
				"Manage Kubernetes pods",
				"list",
				"get",
				"restart",
				"delete",
				"ssh",
			},
		},
		{
			name:    "pods list help",
			args:    []string{"pods", "list", "--help"},
			wantErr: false,
			contains: []string{
				"List pods in the namespace",
				"--namespace",
				"--all-namespaces",
				"--selector",
				"--show-labels",
			},
		},
		{
			name:    "pods get help",
			args:    []string{"pods", "get", "--help"},
			wantErr: false,
			contains: []string{
				"Get details of a specific pod",
				"--namespace",
				"--yaml",
			},
		},
		{
			name:    "pods restart help",
			args:    []string{"pods", "restart", "--help"},
			wantErr: false,
			contains: []string{
				"Restart pods",
				"--deployment",
				"--force",
			},
		},
		{
			name:    "pods delete help",
			args:    []string{"pods", "delete", "--help"},
			wantErr: false,
			contains: []string{
				"Delete a pod",
				"--force",
				"--grace-period",
			},
		},
		{
			name:    "pods ssh help",
			args:    []string{"pods", "ssh", "--help"},
			wantErr: false,
			contains: []string{
				"SSH into a pod",
				"--container",
				"--shell",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newRootCmd("test")
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(tc.args)

			err := cmd.Execute()
			output := buf.String()

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			for _, expected := range tc.contains {
				assert.Contains(t, output, expected)
			}
		})
	}
}

func TestPodsCommandArguments(t *testing.T) {
	testCases := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "pods get missing argument",
			args:    []string{"pods", "get"},
			wantErr: true,
		},
		{
			name:    "pods restart missing argument",
			args:    []string{"pods", "restart"},
			wantErr: true,
		},
		{
			name:    "pods delete missing argument",
			args:    []string{"pods", "delete"},
			wantErr: true,
		},
		{
			name:    "pods ssh missing argument",
			args:    []string{"pods", "ssh"},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newRootCmd("test")
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(tc.args)

			err := cmd.Execute()

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetPodReadyStatus(t *testing.T) {
	testCases := []struct {
		name     string
		pod      *corev1.Pod
		expected string
	}{
		{
			name: "all containers ready",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "container1"}, {Name: "container2"}},
				},
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{Name: "container1", Ready: true},
						{Name: "container2", Ready: true},
					},
				},
			},
			expected: "2/2",
		},
		{
			name: "partial containers ready",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "container1"}, {Name: "container2"}},
				},
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{Name: "container1", Ready: true},
						{Name: "container2", Ready: false},
					},
				},
			},
			expected: "1/2",
		},
		{
			name: "no containers ready",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "container1"}},
				},
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{Name: "container1", Ready: false},
					},
				},
			},
			expected: "0/1",
		},
		{
			name: "empty containers",
			pod: &corev1.Pod{
				Spec:   corev1.PodSpec{Containers: []corev1.Container{}},
				Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{}},
			},
			expected: "0/0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := getPodReadyStatus(tc.pod)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetPodRestartCount(t *testing.T) {
	testCases := []struct {
		name     string
		pod      *corev1.Pod
		expected int32
	}{
		{
			name: "no restarts",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{RestartCount: 0},
						{RestartCount: 0},
					},
				},
			},
			expected: 0,
		},
		{
			name: "some restarts",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{RestartCount: 2},
						{RestartCount: 3},
					},
				},
			},
			expected: 5,
		},
		{
			name: "single container with restarts",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{RestartCount: 10},
					},
				},
			},
			expected: 10,
		},
		{
			name: "no container statuses",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{},
				},
			},
			expected: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := getPodRestartCount(tc.pod)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestPodsCommandStructure(t *testing.T) {
	cmd := newPodsCmd()

	// Verify basic properties
	assert.Equal(t, "pods", cmd.Use)
	assert.Contains(t, cmd.Short, "Manage Kubernetes pods")
	assert.Contains(t, cmd.Long, "List, describe, restart, and perform other operations")

	// Verify subcommands are registered
	subcommands := cmd.Commands()
	expectedCommands := []string{"list", "get", "restart", "delete", "ssh"}

	for _, expected := range expectedCommands {
		found := false
		for _, subcmd := range subcommands {
			if subcmd.Name() == expected {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected subcommand %s not found", expected)
	}
}
