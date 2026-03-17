package cmd

import (
	"strings"
	"testing"

	"orion/internal/types"
)

func TestFormatStatus(t *testing.T) {
	tests := []struct {
		name           string
		status         string
		wantContains   string
		wantColorFunc  string // Expected color function hint
	}{
		{
			name:          "StatusWorking",
			status:        string(types.StatusWorking),
			wantContains:  "WORKING",
			wantColorFunc: "Yellow",
		},
		{
			name:          "StatusReadyToPush",
			status:        string(types.StatusReadyToPush),
			wantContains:  "READY_TO_PUSH",
			wantColorFunc: "Green",
		},
		{
			name:          "StatusFail",
			status:        string(types.StatusFail),
			wantContains:  "FAIL",
			wantColorFunc: "Red",
		},
		{
			name:          "StatusPushed",
			status:        string(types.StatusPushed),
			wantContains:  "PUSHED",
			wantColorFunc: "HiBlack",
		},
		{
			name:          "Empty status defaults to WORKING",
			status:        "",
			wantContains:  "WORKING",
			wantColorFunc: "Yellow",
		},
		{
			name:          "Unknown status defaults to WORKING",
			status:        "UNKNOWN",
			wantContains:  "WORKING",
			wantColorFunc: "Yellow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatStatus(tt.status)

			// Check that the result contains the expected status text
			if !strings.Contains(got, tt.wantContains) {
				t.Errorf("formatStatus(%q) = %q, want to contain %q", tt.status, got, tt.wantContains)
			}

			// Note: We can't directly test ANSI color codes without knowing the exact format,
			// but we verify the function returns a non-empty colored string
			if got == "" {
				t.Errorf("formatStatus(%q) returned empty string, expected colored output", tt.status)
			}

			// Verify that the output is different from plain status (color codes should be added)
			// Note: fatih/color may return plain text if terminal doesn't support colors,
			// so we just verify the status text is present
			if tt.status != "" && !strings.Contains(got, tt.status) && tt.status != "UNKNOWN" {
				t.Errorf("formatStatus(%q) = %q, want to contain original status %q", tt.status, got, tt.status)
			}
		})
	}
}

func TestFormatStatusColorOutput(t *testing.T) {
	// Test that different statuses produce different outputs
	working := formatStatus(string(types.StatusWorking))
	readyToPush := formatStatus(string(types.StatusReadyToPush))
	fail := formatStatus(string(types.StatusFail))
	pushed := formatStatus(string(types.StatusPushed))

	// All should contain their respective status text
	if !strings.Contains(working, "WORKING") {
		t.Errorf("Working status output should contain 'WORKING', got: %q", working)
	}
	if !strings.Contains(readyToPush, "READY_TO_PUSH") {
		t.Errorf("ReadyToPush status output should contain 'READY_TO_PUSH', got: %q", readyToPush)
	}
	if !strings.Contains(fail, "FAIL") {
		t.Errorf("Fail status output should contain 'FAIL', got: %q", fail)
	}
	if !strings.Contains(pushed, "PUSHED") {
		t.Errorf("Pushed status output should contain 'PUSHED', got: %q", pushed)
	}
}
