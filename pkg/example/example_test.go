package example

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdd(t *testing.T) {
	testCases := []struct {
		name     string
		a        int
		b        int
		expected int
	}{
		{
			name:     "positive numbers",
			a:        1,
			b:        2,
			expected: 3,
		},
		{
			name:     "equal numbers",
			a:        2,
			b:        2,
			expected: 4,
		},
		{
			name:     "negative numbers",
			a:        -1,
			b:        -2,
			expected: -3,
		},
		{
			name:     "zero addition",
			a:        5,
			b:        0,
			expected: 5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, Add(tc.a, tc.b))
		})
	}
}

func TestMultiply(t *testing.T) {
	testCases := []struct {
		name     string
		a        int
		b        int
		expected int
	}{
		{
			name:     "positive multiplication",
			a:        1,
			b:        2,
			expected: 2,
		},
		{
			name:     "equal numbers multiplication",
			a:        2,
			b:        2,
			expected: 4,
		},
		{
			name:     "zero multiplication",
			a:        5,
			b:        0,
			expected: 0,
		},
		{
			name:     "negative multiplication",
			a:        -2,
			b:        3,
			expected: -6,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, Multiply(tc.a, tc.b))
		})
	}
}
