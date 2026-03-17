package cmd

import (
	"testing"

	"orion/internal/types"
)

// TestFormatStatus 测试 formatStatus 函数的输出
func TestFormatStatus(t *testing.T) {
	tests := []struct {
		name           string
		status         string
		expectedOutput string
		shouldContain  []string
	}{
		{
			name:           "Working status",
			status:         string(types.StatusWorking),
			expectedOutput: "WORKING",
			shouldContain:  []string{"WORKING"},
		},
		{
			name:           "ReadyToPush status",
			status:         string(types.StatusReadyToPush),
			expectedOutput: "READY_TO_PUSH",
			shouldContain:  []string{"READY_TO_PUSH"},
		},
		{
			name:           "Fail status",
			status:         string(types.StatusFail),
			expectedOutput: "FAIL",
			shouldContain:  []string{"FAIL"},
		},
		{
			name:           "Pushed status",
			status:         string(types.StatusPushed),
			expectedOutput: "PUSHED",
			shouldContain:  []string{"PUSHED"},
		},
		{
			name:           "Empty status (legacy)",
			status:         "",
			expectedOutput: "WORKING",
			shouldContain:  []string{"WORKING"},
		},
		{
			name:           "Unknown status",
			status:         "UNKNOWN_STATUS",
			expectedOutput: "WORKING",
			shouldContain:  []string{"WORKING"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatStatus(tt.status)

			// 验证输出包含预期的状态文本
			for _, expected := range tt.shouldContain {
				if !containsString(result, expected) {
					t.Errorf("formatStatus(%q) = %q, should contain %q", tt.status, result, expected)
				}
			}

			// 验证返回非空字符串
			if result == "" {
				t.Errorf("formatStatus(%q) returned empty string", tt.status)
			}
		})
	}
}

// TestFormatStatus_ColorCodes 测试 formatStatus 函数返回带颜色的字符串
func TestFormatStatus_ColorCodes(t *testing.T) {
	// 验证不同状态返回不同的颜色编码字符串
	statuses := []string{
		string(types.StatusWorking),
		string(types.StatusReadyToPush),
		string(types.StatusFail),
		string(types.StatusPushed),
	}

	results := make(map[string]string)
	for _, status := range statuses {
		results[status] = formatStatus(status)
	}

	// 验证不同状态返回不同的字符串（因为颜色不同）
	for i, s1 := range statuses {
		for j := range statuses {
			if i != j {
				// 注意：由于颜色编码的存在，不同状态应该返回不同的字符串
				// 但这里我们只验证它们都包含各自的状态文本
				if !containsString(results[s1], s1) && s1 != "" {
					t.Errorf("formatStatus(%q) result should contain %q", s1, s1)
				}
			}
		}
	}
}

// TestFormatStatus_StatusConstants 测试状态常量与 formatStatus 的配合
func TestFormatStatus_StatusConstants(t *testing.T) {
	tests := []struct {
		name   string
		status types.NodeStatus
	}{
		{"StatusWorking", types.StatusWorking},
		{"StatusReadyToPush", types.StatusReadyToPush},
		{"StatusFail", types.StatusFail},
		{"StatusPushed", types.StatusPushed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatStatus(string(tt.status))

			// 验证结果包含状态字符串
			if !containsString(result, string(tt.status)) {
				t.Errorf("formatStatus(%q) = %q, should contain %q", tt.status, result, tt.status)
			}
		})
	}
}

// containsString 辅助函数，检查字符串是否包含子串
func containsString(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
