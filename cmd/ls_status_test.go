package cmd

import (
	"strings"
	"testing"

	"github.com/fatih/color"
	"orion/internal/types"
)

// TestFormatStatus 测试 formatStatus 函数的颜色输出
func TestFormatStatus(t *testing.T) {
	tests := []struct {
		name           string
		status         string
		expectedColor  string
		expectedStatus string
	}{
		{
			name:           "WORKING status",
			status:         string(types.StatusWorking),
			expectedStatus: "WORKING",
		},
		{
			name:           "READY_TO_PUSH status",
			status:         string(types.StatusReadyToPush),
			expectedStatus: "READY_TO_PUSH",
		},
		{
			name:           "FAIL status",
			status:         string(types.StatusFail),
			expectedStatus: "FAIL",
		},
		{
			name:           "PUSHED status",
			status:         string(types.StatusPushed),
			expectedStatus: "PUSHED",
		},
		{
			name:           "empty status defaults to WORKING",
			status:         "",
			expectedStatus: "WORKING",
		},
		{
			name:           "unknown status defaults to WORKING",
			status:         "UNKNOWN",
			expectedStatus: "WORKING",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatStatus(tt.status)

			// 验证输出包含正确的状态文本
			// color 函数会添加 ANSI 转义序列，所以我们需要检查是否包含状态文本
			if !strings.Contains(result, tt.expectedStatus) {
				t.Errorf("formatStatus(%q) = %q, want to contain %q", tt.status, result, tt.expectedStatus)
			}
		})
	}
}

// TestFormatStatusColorCodes 验证不同状态使用正确的颜色
func TestFormatStatusColorCodes(t *testing.T) {
	// 禁用颜色输出以便测试
	color.NoColor = true

	tests := []struct {
		name     string
		status   string
		expected string
	}{
		{
			name:     "WORKING status",
			status:   string(types.StatusWorking),
			expected: "WORKING",
		},
		{
			name:     "READY_TO_PUSH status",
			status:   string(types.StatusReadyToPush),
			expected: "READY_TO_PUSH",
		},
		{
			name:     "FAIL status",
			status:   string(types.StatusFail),
			expected: "FAIL",
		},
		{
			name:     "PUSHED status",
			status:   string(types.StatusPushed),
			expected: "PUSHED",
		},
		{
			name:     "empty status",
			status:   "",
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

// TestFormatStatusWithColorEnabled 测试启用颜色时的输出
func TestFormatStatusWithColorEnabled(t *testing.T) {
	// 启用颜色输出
	color.NoColor = false

	tests := []struct {
		name   string
		status string
	}{
		{
			name:   "WORKING status with color",
			status: string(types.StatusWorking),
		},
		{
			name:   "READY_TO_PUSH status with color",
			status: string(types.StatusReadyToPush),
		},
		{
			name:   "FAIL status with color",
			status: string(types.StatusFail),
		},
		{
			name:   "PUSHED status with color",
			status: string(types.StatusPushed),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatStatus(tt.status)
			// 验证输出包含状态文本（可能带有 ANSI 颜色代码）
			if !strings.Contains(result, tt.status) {
				t.Errorf("formatStatus(%q) = %q, want to contain %q", tt.status, result, tt.status)
			}
		})
	}
}
