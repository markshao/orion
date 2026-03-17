package types

import (
	"testing"
)

func TestNodeStatusConstants(t *testing.T) {
	tests := []struct {
		name   string
		status NodeStatus
		want   string
	}{
		{
			name:   "StatusWorking",
			status: StatusWorking,
			want:   "WORKING",
		},
		{
			name:   "StatusReadyToPush",
			status: StatusReadyToPush,
			want:   "READY_TO_PUSH",
		},
		{
			name:   "StatusFail",
			status: StatusFail,
			want:   "FAIL",
		},
		{
			name:   "StatusPushed",
			status: StatusPushed,
			want:   "PUSHED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.want {
				t.Errorf("NodeStatus string = %q, want %q", tt.status, tt.want)
			}
		})
	}
}

func TestNodeStatusEmpty(t *testing.T) {
	// Test that empty status can be used for legacy nodes
	var status NodeStatus
	if status != "" {
		t.Errorf("empty NodeStatus should be empty string, got %q", status)
	}
}
