package types

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNodeStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		status   NodeStatus
		expected string
	}{
		{"Working", StatusWorking, "WORKING"},
		{"ReadyToPush", StatusReadyToPush, "READY_TO_PUSH"},
		{"Fail", StatusFail, "FAIL"},
		{"Pushed", StatusPushed, "PUSHED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.status)
			}
		})
	}
}

func TestNodeStatusJSONSerialization(t *testing.T) {
	tests := []struct {
		name         string
		status       NodeStatus
		expectedJSON string
	}{
		{"Working", StatusWorking, `"WORKING"`},
		{"ReadyToPush", StatusReadyToPush, `"READY_TO_PUSH"`},
		{"Fail", StatusFail, `"FAIL"`},
		{"Pushed", StatusPushed, `"PUSHED"`},
		{"Empty", "", `""`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal
			data, err := json.Marshal(tt.status)
			if err != nil {
				t.Fatalf("json.Marshal failed: %v", err)
			}
			if string(data) != tt.expectedJSON {
				t.Errorf("Expected JSON %s, got %s", tt.expectedJSON, string(data))
			}

			// Unmarshal
			var unmarshaled NodeStatus
			if err := json.Unmarshal([]byte(tt.expectedJSON), &unmarshaled); err != nil {
				t.Fatalf("json.Unmarshal failed: %v", err)
			}
			if unmarshaled != tt.status {
				t.Errorf("Expected status %s, got %s", tt.status, unmarshaled)
			}
		})
	}
}

func TestNodeWithStatusJSONSerialization(t *testing.T) {
	node := Node{
		Name:          "test-node",
		LogicalBranch: "feature/test",
		BaseBranch:    "main",
		ShadowBranch:  "orion/run-123/test",
		WorktreePath:  "/tmp/orion/nodes/test-node",
		Label:         "test",
		CreatedBy:     "user",
		AppliedRuns:   []string{"run-1", "run-2"},
		Status:        StatusReadyToPush,
		CreatedAt:     time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	// Marshal
	data, err := json.Marshal(node)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	// Verify status is in JSON
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	statusVal, ok := result["status"]
	if !ok {
		t.Error("Expected 'status' field in JSON")
	}
	if statusVal != "READY_TO_PUSH" {
		t.Errorf("Expected status 'READY_TO_PUSH', got %v", statusVal)
	}

	// Unmarshal
	var unmarshaled Node
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if unmarshaled.Name != node.Name {
		t.Errorf("Expected name %s, got %s", node.Name, unmarshaled.Name)
	}
	if unmarshaled.Status != StatusReadyToPush {
		t.Errorf("Expected status READY_TO_PUSH, got %s", unmarshaled.Status)
	}
	if len(unmarshaled.AppliedRuns) != 2 {
		t.Errorf("Expected 2 applied runs, got %d", len(unmarshaled.AppliedRuns))
	}
}

func TestNodeWithEmptyStatus(t *testing.T) {
	node := Node{
		Name:          "legacy-node",
		LogicalBranch: "feature/legacy",
		Status:        "", // Legacy node without status
		CreatedAt:     time.Now(),
	}

	// Marshal should work with empty status
	data, err := json.Marshal(node)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	// Unmarshal
	var unmarshaled Node
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if unmarshaled.Status != "" {
		t.Errorf("Expected empty status, got %s", unmarshaled.Status)
	}
}

func TestNodeStatusComparison(t *testing.T) {
	tests := []struct {
		name     string
		s1       NodeStatus
		s2       NodeStatus
		expected bool
	}{
		{"Same Working", StatusWorking, StatusWorking, true},
		{"Same ReadyToPush", StatusReadyToPush, StatusReadyToPush, true},
		{"Different", StatusWorking, StatusReadyToPush, false},
		{"Empty vs Working", "", StatusWorking, false},
		{"Empty vs Empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if (tt.s1 == tt.s2) != tt.expected {
				t.Errorf("Expected comparison result %v, got %v", tt.expected, tt.s1 == tt.s2)
			}
		})
	}
}

func TestNodeStatusStringConversion(t *testing.T) {
	tests := []struct {
		name     string
		status   NodeStatus
		expected string
	}{
		{"Working", StatusWorking, "WORKING"},
		{"ReadyToPush", StatusReadyToPush, "READY_TO_PUSH"},
		{"Fail", StatusFail, "FAIL"},
		{"Pushed", StatusPushed, "PUSHED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := string(tt.status)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestNodeStatusFromValidString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected NodeStatus
	}{
		{"From WORKING", "WORKING", StatusWorking},
		{"From READY_TO_PUSH", "READY_TO_PUSH", StatusReadyToPush},
		{"From FAIL", "FAIL", StatusFail},
		{"From PUSHED", "PUSHED", StatusPushed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := NodeStatus(tt.input)
			if status != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, status)
			}
		})
	}
}

func TestNodeStatusFromInvalidString(t *testing.T) {
	// Invalid status should still be accepted (no validation in type system)
	status := NodeStatus("INVALID_STATUS")
	if status != "INVALID_STATUS" {
		t.Errorf("Expected INVALID_STATUS, got %s", status)
	}
}
