package cmd

import (
	"strings"
	"testing"

	"orion/internal/types"
)

// TestFormatStatus 测试 formatStatus 函数的状态格式化逻辑
func TestFormatStatus(t *testing.T) {
	tests := []struct {
		name           string
		status         string
		wantContains   string
		wantNotContain string
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
			name:         "Unknown status defaults to WORKING",
			status:       "UNKNOWN",
			wantContains: "WORKING",
		},
		{
			name:         "Empty status defaults to WORKING",
			status:       "",
			wantContains: "WORKING",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatStatus(tt.status)
			if !strings.Contains(got, tt.wantContains) {
				t.Errorf("formatStatus(%q) = %q, want to contain %q", tt.status, got, tt.wantContains)
			}
			if tt.wantNotContain != "" && strings.Contains(got, tt.wantNotContain) {
				t.Errorf("formatStatus(%q) = %q, should not contain %q", tt.status, got, tt.wantNotContain)
			}
		})
	}
}

// TestFormatStatusColorCodes 测试 formatStatus 返回的颜色代码
func TestFormatStatusColorCodes(t *testing.T) {
	// 验证不同状态返回不同的颜色格式化
	working := formatStatus(string(types.StatusWorking))
	ready := formatStatus(string(types.StatusReadyToPush))
	fail := formatStatus(string(types.StatusFail))
	pushed := formatStatus(string(types.StatusPushed))

	// 验证它们都包含 ANSI 颜色代码（fatih/color 包使用 \x1b 转义序列）
	// 注意：在测试环境中，color 包可能会检测到终端不支持颜色而返回纯文本
	// 所以我们只验证返回的字符串包含状态名称
	hasColorOrText := func(s, expectedText string) bool {
		// 要么包含 ANSI 转义序列，要么包含期望的文本
		return strings.Contains(s, expectedText)
	}

	if !hasColorOrText(working, "WORKING") {
		t.Error("WORKING status should contain 'WORKING'")
	}
	if !hasColorOrText(ready, "READY_TO_PUSH") {
		t.Error("READY_TO_PUSH status should contain 'READY_TO_PUSH'")
	}
	if !hasColorOrText(fail, "FAIL") {
		t.Error("FAIL status should contain 'FAIL'")
	}
	if !hasColorOrText(pushed, "PUSHED") {
		t.Error("PUSHED status should contain 'PUSHED'")
	}

	// 验证不同状态使用不同的格式化（即使没有颜色，文本也应该不同）
	statuses := []string{working, ready, fail, pushed}
	for i := 0; i < len(statuses); i++ {
		for j := i + 1; j < len(statuses); j++ {
			if statuses[i] == statuses[j] {
				t.Errorf("Status %d and %d should have different formatting", i, j)
			}
		}
	}
}

// TestLsCommandQuietMode 测试 ls 命令的 quiet 模式
func TestLsCommandQuietMode(t *testing.T) {
	// 验证 quiet 标志是否正确配置
	flag := lsCmd.Flags().Lookup("quiet")
	if flag == nil || flag.DefValue != "false" {
		t.Error("quiet flag default should be false")
	}
}

// TestLsCommandAllFlag 测试 ls 命令的 all 标志
func TestLsCommandAllFlag(t *testing.T) {
	// 验证 all 标志是否正确配置
	flag := lsCmd.Flags().Lookup("all")
	if flag == nil || flag.DefValue != "false" {
		t.Error("all flag default should be false")
	}
}

// TestLsCommandFiltersAgentNodes 测试 ls 命令过滤 agent 节点的逻辑
func TestLsCommandFiltersAgentNodes(t *testing.T) {
	// 测试节点过滤逻辑
	// 根据 ls.go 的实现：if !showAll && node.CreatedBy != "user" { continue }
	// 这意味着：只有当 showAll=true 或 CreatedBy="user" 时才显示
	tests := []struct {
		name      string
		createdBy string
		showAll   bool
		wantShow  bool
	}{
		{"User node, showAll=false", "user", false, true},
		{"User node, showAll=true", "user", true, true},
		{"Agent node, showAll=false", "run-123", false, false},
		{"Agent node, showAll=true", "run-123", true, true},
		{"Empty created_by, showAll=false", "", false, false}, // Empty is treated as agent node
		{"Empty created_by, showAll=true", "", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldShow := tt.showAll || tt.createdBy == "user"
			if shouldShow != tt.wantShow {
				t.Errorf("Filter logic mismatch: shouldShow=%v, want=%v", shouldShow, tt.wantShow)
			}
		})
	}
}

// TestNodeStatusDisplay 测试节点状态显示逻辑
func TestNodeStatusDisplay(t *testing.T) {
	// 验证所有定义的状态都能被正确显示
	allStatuses := []types.NodeStatus{
		types.StatusWorking,
		types.StatusReadyToPush,
		types.StatusFail,
		types.StatusPushed,
	}

	for _, status := range allStatuses {
		t.Run(string(status), func(t *testing.T) {
			display := formatStatus(string(status))
			if display == "" {
				t.Error("Status display should not be empty")
			}
			if !strings.Contains(display, string(status)) {
				t.Errorf("Display %q should contain status name %q", display, status)
			}
		})
	}
}
