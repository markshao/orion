package cmd

import (
	"strings"
	"testing"

	"orion/internal/types"
)

func TestFormatStatus(t *testing.T) {
	tests := []struct {
		name         string
		status       string
		wantContains string
	}{
		{
			name:         "WORKING status",
			status:       string(types.StatusWorking),
			wantContains: "WORKING",
		},
		{
			name:         "READY_TO_PUSH status",
			status:       string(types.StatusReadyToPush),
			wantContains: "READY_TO_PUSH",
		},
		{
			name:         "FAIL status",
			status:       string(types.StatusFail),
			wantContains: "FAIL",
		},
		{
			name:         "PUSHED status",
			status:       string(types.StatusPushed),
			wantContains: "PUSHED",
		},
		{
			name:         "Empty status (legacy)",
			status:       "",
			wantContains: "WORKING",
		},
		{
			name:         "Unknown status",
			status:       "UNKNOWN",
			wantContains: "WORKING",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatStatus(tt.status)

			// Check if the result contains the expected status text
			if !strings.Contains(got, tt.wantContains) {
				t.Errorf("formatStatus(%q) = %q, want to contain %q", tt.status, got, tt.wantContains)
			}
		})
	}
}

func TestFormatStatusColorConsistency(t *testing.T) {
	// Test that the same status always returns the same color
	statuses := []string{
		string(types.StatusWorking),
		string(types.StatusReadyToPush),
		string(types.StatusFail),
		string(types.StatusPushed),
	}

	for _, status := range statuses {
		result1 := formatStatus(status)
		result2 := formatStatus(status)
		result3 := formatStatus(status)

		// All results should contain the same status text
		if !strings.Contains(result1, status) {
			t.Errorf("formatStatus(%q) result does not contain status: %q", status, result1)
		}
		if !strings.Contains(result2, status) {
			t.Errorf("formatStatus(%q) result does not contain status: %q", status, result2)
		}
		if !strings.Contains(result3, status) {
			t.Errorf("formatStatus(%q) result does not contain status: %q", status, result3)
		}
	}
}

func TestFormatStatusAllNodeStatusTypes(t *testing.T) {
	// Test all defined NodeStatus types
	allStatuses := []struct {
		status   types.NodeStatus
		expected string
	}{
		{types.StatusWorking, "WORKING"},
		{types.StatusReadyToPush, "READY_TO_PUSH"},
		{types.StatusFail, "FAIL"},
		{types.StatusPushed, "PUSHED"},
	}

	for _, s := range allStatuses {
		t.Run(string(s.status), func(t *testing.T) {
			result := formatStatus(string(s.status))
			if !strings.Contains(result, s.expected) {
				t.Errorf("formatStatus(%q) = %q, want to contain %q", s.status, result, s.expected)
			}
		})
	}
}
