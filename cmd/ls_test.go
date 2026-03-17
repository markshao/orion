package cmd

import (
	"testing"

	"orion/internal/types"

	"github.com/fatih/color"
)

func TestFormatStatus(t *testing.T) {
	// Disable color output for testing
	color.NoColor = true

	tests := []struct {
		name     string
		status   string
		expected string
	}{
		{
			name:     "StatusWorking",
			status:   string(types.StatusWorking),
			expected: "WORKING",
		},
		{
			name:     "StatusReadyToPush",
			status:   string(types.StatusReadyToPush),
			expected: "READY_TO_PUSH",
		},
		{
			name:     "StatusFail",
			status:   string(types.StatusFail),
			expected: "FAIL",
		},
		{
			name:     "StatusPushed",
			status:   string(types.StatusPushed),
			expected: "PUSHED",
		},
		{
			name:     "Empty status (legacy)",
			status:   "",
			expected: "WORKING",
		},
		{
			name:     "Unknown status",
			status:   "UNKNOWN",
			expected: "WORKING",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatStatus(tt.status)
			if result != tt.expected {
				t.Errorf("formatStatus(%q) = %q, want %q", tt.status, result, tt.expected)
			}
		})
	}
}
