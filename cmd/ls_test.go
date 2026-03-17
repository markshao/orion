package cmd

import (
	"testing"

	"orion/internal/types"
)

func TestFormatStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected string
	}{
		{
			name:     "StatusWorking",
			status:   string(types.StatusWorking),
			expected: "WORKING", // Yellow colored
		},
		{
			name:     "StatusReadyToPush",
			status:   string(types.StatusReadyToPush),
			expected: "READY_TO_PUSH", // Green colored
		},
		{
			name:     "StatusFail",
			status:   string(types.StatusFail),
			expected: "FAIL", // Red colored
		},
		{
			name:     "StatusPushed",
			status:   string(types.StatusPushed),
			expected: "PUSHED", // HiBlack colored
		},
		{
			name:     "Empty status defaults to WORKING",
			status:   "",
			expected: "WORKING",
		},
		{
			name:     "Unknown status defaults to WORKING",
			status:   "UNKNOWN",
			expected: "WORKING",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatStatus(tt.status)
			
			// The result includes ANSI color codes, so we need to check if it contains the expected text
			// ANSI codes format: \x1b[<color>m<text>\x1b[0m
			if !containsString(result, tt.expected) {
				t.Errorf("formatStatus(%q) = %q, expected to contain %q", tt.status, result, tt.expected)
			}
		})
	}
}

// containsString checks if s contains substr (ignoring ANSI color codes)
func containsString(s, substr string) bool {
	// Simple check: the substring should appear in s
	// Note: This will match even if the text is wrapped in ANSI codes
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
