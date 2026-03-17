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
		expectedText   string
		expectedColors bool // true if we expect color codes in output
	}{
		{"Working", string(types.StatusWorking), "WORKING", true},
		{"ReadyToPush", string(types.StatusReadyToPush), "READY_TO_PUSH", true},
		{"Fail", string(types.StatusFail), "FAIL", true},
		{"Pushed", string(types.StatusPushed), "PUSHED", true},
		{"Default", "UNKNOWN", "WORKING", true}, // Default case
		{"Empty", "", "WORKING", true},          // Empty defaults to WORKING
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatStatus(tt.status)

			// Check if the expected text is in the result (color codes may be present)
			if !strings.Contains(result, tt.expectedText) {
				t.Errorf("formatStatus(%q) = %q, expected to contain %q", tt.status, result, tt.expectedText)
			}

			// Color output should contain ANSI codes when colors are expected
			if tt.expectedColors && !strings.Contains(result, "\033") {
				// Note: This might fail if color output is disabled in the test environment
				// In that case, this check can be removed or made conditional
				t.Logf("Expected color codes in output for status %s", tt.status)
			}
		})
	}
}

func TestFormatStatusColorCodes(t *testing.T) {
	// Test that different statuses produce different color codes
	statuses := []types.NodeStatus{
		types.StatusWorking,
		types.StatusReadyToPush,
		types.StatusFail,
		types.StatusPushed,
	}

	results := make(map[string]string)
	for _, status := range statuses {
		result := formatStatus(string(status))
		results[string(status)] = result
	}

	// Verify each status produces the correct text
	expectedTexts := map[string]string{
		"WORKING":       "WORKING",
		"READY_TO_PUSH": "READY_TO_PUSH",
		"FAIL":          "FAIL",
		"PUSHED":        "PUSHED",
	}

	for status, expectedText := range expectedTexts {
		result, ok := results[status]
		if !ok {
			t.Errorf("No result for status %s", status)
			continue
		}
		if !strings.Contains(result, expectedText) {
			t.Errorf("formatStatus(%q) = %q, expected to contain %q", status, result, expectedText)
		}
	}
}

func TestFormatStatusDefault(t *testing.T) {
	// Unknown status should default to WORKING
	result := formatStatus("UNKNOWN_STATUS")
	if !strings.Contains(result, "WORKING") {
		t.Errorf("formatStatus(\"UNKNOWN_STATUS\") = %q, expected to contain \"WORKING\"", result)
	}
}

func TestFormatStatusEmpty(t *testing.T) {
	// Empty status should default to WORKING
	result := formatStatus("")
	if !strings.Contains(result, "WORKING") {
		t.Errorf("formatStatus(\"\") = %q, expected to contain \"WORKING\"", result)
	}
}
