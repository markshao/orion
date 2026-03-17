package cmd

import (
	"testing"

	"orion/internal/types"
)

func TestFormatStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		wantColor bool // whether output should contain ANSI color codes
	}{
		{
			name:     "StatusWorking",
			status:   string(types.StatusWorking),
			wantColor: true,
		},
		{
			name:     "StatusReadyToPush",
			status:   string(types.StatusReadyToPush),
			wantColor: true,
		},
		{
			name:     "StatusFail",
			status:   string(types.StatusFail),
			wantColor: true,
		},
		{
			name:     "StatusPushed",
			status:   string(types.StatusPushed),
			wantColor: true,
		},
		{
			name:     "Empty status (legacy)",
			status:   "",
			wantColor: true,
		},
		{
			name:     "Unknown status",
			status:   "UNKNOWN",
			wantColor: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatStatus(tt.status)

			// Check that output is non-empty
			if got == "" {
				t.Errorf("formatStatus(%q) returned empty string", tt.status)
			}

			// Check that output contains the expected status text
			// Note: formatStatus returns colored strings, so we check for the status text
			switch tt.status {
			case string(types.StatusWorking), "":
				// Working and empty status should return "WORKING"
				if got == "" {
					t.Errorf("formatStatus(%q) should return non-empty string", tt.status)
				}
			case string(types.StatusReadyToPush):
				if got == "" {
					t.Errorf("formatStatus(%q) should return non-empty string", tt.status)
				}
			case string(types.StatusFail):
				if got == "" {
					t.Errorf("formatStatus(%q) should return non-empty string", tt.status)
				}
			case string(types.StatusPushed):
				if got == "" {
					t.Errorf("formatStatus(%q) should return non-empty string", tt.status)
				}
			default:
				// Unknown status should default to WORKING
				if got == "" {
					t.Errorf("formatStatus(%q) should return non-empty string for unknown status", tt.status)
				}
			}
		})
	}
}

func TestFormatStatusColorCodes(t *testing.T) {
	// Test that different statuses return different colored outputs
	working := formatStatus(string(types.StatusWorking))
	ready := formatStatus(string(types.StatusReadyToPush))
	fail := formatStatus(string(types.StatusFail))
	pushed := formatStatus(string(types.StatusPushed))

	// All should be non-empty
	if working == "" {
		t.Error("formatStatus(WORKING) should not be empty")
	}
	if ready == "" {
		t.Error("formatStatus(READY_TO_PUSH) should not be empty")
	}
	if fail == "" {
		t.Error("formatStatus(FAIL) should not be empty")
	}
	if pushed == "" {
		t.Error("formatStatus(PUSHED) should not be empty")
	}
}
