package cmd

import (
	"testing"

	"orion/internal/types"

	"github.com/fatih/color"
)

// TestFormatStatus 测试 formatStatus 函数的各种状态
func TestFormatStatus(t *testing.T) {
	tests := []struct {
		name         string
		status       string
		expectedText string
	}{
		{
			name:         "StatusWorking returns yellow WORKING",
			status:       string(types.StatusWorking),
			expectedText: "WORKING",
		},
		{
			name:         "StatusReadyToPush returns green READY_TO_PUSH",
			status:       string(types.StatusReadyToPush),
			expectedText: "READY_TO_PUSH",
		},
		{
			name:         "StatusFail returns red FAIL",
			status:       string(types.StatusFail),
			expectedText: "FAIL",
		},
		{
			name:         "StatusPushed returns hi-black PUSHED",
			status:       string(types.StatusPushed),
			expectedText: "PUSHED",
		},
		{
			name:         "Empty status defaults to yellow WORKING",
			status:       "",
			expectedText: "WORKING",
		},
		{
			name:         "Unknown status defaults to yellow WORKING",
			status:       "UNKNOWN_STATUS",
			expectedText: "WORKING",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatStatus(tt.status)

			// Check if result contains expected text
			if got == "" {
				t.Errorf("formatStatus(%q) returned empty string, want non-empty", tt.status)
			}

			// Generate expected output using the correct color function based on status
			var want string
			switch tt.status {
			case string(types.StatusWorking):
				want = color.YellowString("WORKING")
			case string(types.StatusReadyToPush):
				want = color.GreenString("READY_TO_PUSH")
			case string(types.StatusFail):
				want = color.RedString("FAIL")
			case string(types.StatusPushed):
				want = color.HiBlackString("PUSHED")
			default:
				want = color.YellowString("WORKING")
			}

			if got != want {
				t.Errorf("formatStatus(%q) = %q, want %q", tt.status, got, want)
			}
		})
	}
}

// TestFormatStatusANSICodes 验证 formatStatus 返回包含 ANSI 颜色码
func TestFormatStatusANSICodes(t *testing.T) {
	// Enable color output for testing
	color.NoColor = false

	tests := []struct {
		name   string
		status string
	}{
		{"Working status", string(types.StatusWorking)},
		{"Ready to push status", string(types.StatusReadyToPush)},
		{"Fail status", string(types.StatusFail)},
		{"Pushed status", string(types.StatusPushed)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatStatus(tt.status)

			// Check that result contains ANSI escape sequence (ESC[)
			// ANSI codes start with \x1b[ or \033[
			if len(got) == 0 {
				t.Error("formatStatus should return non-empty string")
			}

			// The colored output should be different from plain text
			plainText := tt.status
			if tt.status == "" {
				plainText = "WORKING"
			}
			if got == plainText {
				t.Errorf("formatStatus(%q) should include ANSI color codes, got plain text", tt.status)
			}
		})
	}
}

// TestFormatStatusAllNodeStatuses 测试所有 NodeStatus 类型的格式化
func TestFormatStatusAllNodeStatuses(t *testing.T) {
	allStatuses := []types.NodeStatus{
		types.StatusWorking,
		types.StatusReadyToPush,
		types.StatusFail,
		types.StatusPushed,
	}

	expectedTexts := map[types.NodeStatus]string{
		types.StatusWorking:      "WORKING",
		types.StatusReadyToPush:  "READY_TO_PUSH",
		types.StatusFail:         "FAIL",
		types.StatusPushed:       "PUSHED",
	}

	for _, status := range allStatuses {
		t.Run(string(status), func(t *testing.T) {
			got := formatStatus(string(status))
			wantText := expectedTexts[status]
			want := color.YellowString(wantText)

			// Verify correct color function is used based on status
			switch status {
			case types.StatusWorking:
				want = color.YellowString("WORKING")
			case types.StatusReadyToPush:
				want = color.GreenString("READY_TO_PUSH")
			case types.StatusFail:
				want = color.RedString("FAIL")
			case types.StatusPushed:
				want = color.HiBlackString("PUSHED")
			}

			if got != want {
				t.Errorf("formatStatus(%q) = %q, want %q", status, got, want)
			}
		})
	}
}
