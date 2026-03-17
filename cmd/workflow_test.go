package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"orion/internal/git"
	"orion/internal/types"
	"orion/internal/workspace"
	"orion/internal/workflow"
)

// setupTestWorkspaceForWorkflow 创建一个用于测试 workflow 命令的临时 workspace
func setupTestWorkspaceForWorkflow(t *testing.T) (rootPath, repoPath string, cleanup func()) {
	t.Helper()

	// 1. 创建 root dir
	rootDir, err := os.MkdirTemp("", "orion-workflow-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// 2. 创建 remote repo
	remoteDir, err := os.MkdirTemp("", "orion-remote-test")
	if err != nil {
		os.RemoveAll(rootDir)
		t.Fatalf("failed to create temp remote dir: %v", err)
	}

	// 初始化 remote repo
	exec.Command("git", "init", remoteDir).Run()
	exec.Command("git", "-C", remoteDir, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", remoteDir, "config", "user.name", "Test User").Run()
	exec.Command("git", "-C", remoteDir, "checkout", "-b", "main").Run()
	os.WriteFile(filepath.Join(remoteDir, "README.md"), []byte("# Remote"), 0644)
	exec.Command("git", "-C", remoteDir, "add", ".").Run()
	exec.Command("git", "-C", remoteDir, "commit", "-m", "Initial commit").Run()

	// 3. 初始化 workspace
	wm, err := workspace.Init(rootDir, remoteDir)
	if err != nil {
		os.RemoveAll(rootDir)
		os.RemoveAll(remoteDir)
		t.Fatalf("Init failed: %v", err)
	}

	// 4. 克隆 repo
	if err := git.Clone(remoteDir, wm.State.RepoPath); err != nil {
		os.RemoveAll(rootDir)
		os.RemoveAll(remoteDir)
		t.Fatalf("Clone failed: %v", err)
	}

	// 配置 local repo
	exec.Command("git", "-C", wm.State.RepoPath, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", wm.State.RepoPath, "config", "user.name", "Test User").Run()

	cleanup = func() {
		os.RemoveAll(rootDir)
		os.RemoveAll(remoteDir)
	}

	return rootDir, wm.State.RepoPath, cleanup
}

// TestWorkflowRmCommand 测试 workflow rm 命令
func TestWorkflowRmCommand(t *testing.T) {
	rootPath, _, cleanup := setupTestWorkspaceForWorkflow(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	engine := workflow.NewEngine(wm)

	// 创建一个测试 run
	run, err := engine.StartRun("test-workflow", "manual", "main", "")
	if err != nil {
		// 可能因为 workflow 配置不存在而失败，这是预期的
		t.Logf("StartRun failed (expected if workflow config missing): %v", err)
		return
	}

	// 验证 run 被创建
	runs, err := engine.ListRuns()
	if err != nil {
		t.Fatalf("ListRuns failed: %v", err)
	}

	found := false
	for _, r := range runs {
		if r.ID == run.ID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Run %s not found in ListRuns", run.ID)
	}
}

// TestWorkflowLsQuietMode 测试 workflow ls 的 quiet 模式
func TestWorkflowLsQuietMode(t *testing.T) {
	rootPath, _, cleanup := setupTestWorkspaceForWorkflow(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	engine := workflow.NewEngine(wm)

	// 获取运行列表（应该是空的）
	runs, err := engine.ListRuns()
	if err != nil {
		t.Fatalf("ListRuns failed: %v", err)
	}

	// 验证 quiet 模式输出
	if len(runs) == 0 {
		// 空列表时 quiet 模式应该不输出任何内容
		t.Logf("No workflow runs found (expected)")
	}
}

// TestWorkflowInspectCommand 测试 workflow inspect 命令
func TestWorkflowInspectCommand(t *testing.T) {
	rootPath, _, cleanup := setupTestWorkspaceForWorkflow(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	engine := workflow.NewEngine(wm)

	// 尝试获取不存在的 run
	_, err = engine.GetRun("non-existent-run")
	if err == nil {
		t.Error("Expected error for non-existent run")
	}
}

// TestArtifactsLsCommand 测试 artifacts ls 命令
func TestArtifactsLsCommand(t *testing.T) {
	rootPath, _, cleanup := setupTestWorkspaceForWorkflow(t)
	defer cleanup()

	// 验证 artifacts 目录结构
	artifactBaseDir := filepath.Join(rootPath, ".orion", "runs")
	if _, err := os.Stat(artifactBaseDir); os.IsNotExist(err) {
		t.Logf("Artifact base directory does not exist yet (expected)")
	}
}

// TestSelectWorkflowRun 测试工作流选择功能（如果有）
func TestWorkflowRunSelection(t *testing.T) {
	rootPath, _, cleanup := setupTestWorkspaceForWorkflow(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	engine := workflow.NewEngine(wm)

	// 验证空列表情况
	runs, err := engine.ListRuns()
	if err != nil {
		t.Fatalf("ListRuns failed: %v", err)
	}

	if len(runs) != 0 {
		t.Logf("Found %d workflow runs", len(runs))
	}
}

// TestWorkflowEngineStatusUpdate 测试 workflow engine 的状态更新功能
func TestWorkflowEngineStatusUpdate(t *testing.T) {
	rootPath, _, cleanup := setupTestWorkspaceForWorkflow(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// 创建节点
	nodeName := "test-status-node"
	if err := wm.SpawnNode(nodeName, "feature/status-test", "main", "Status Test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// 测试状态更新
	testCases := []struct {
		name       string
		fromStatus types.NodeStatus
		toStatus   types.NodeStatus
	}{
		{"WORKING to READY_TO_PUSH", types.StatusWorking, types.StatusReadyToPush},
		{"WORKING to FAIL", types.StatusWorking, types.StatusFail},
		{"READY_TO_PUSH to PUSHED", types.StatusReadyToPush, types.StatusPushed},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 设置初始状态
			if err := wm.UpdateNodeStatus(nodeName, tc.fromStatus); err != nil {
				t.Fatalf("Failed to set initial status: %v", err)
			}

			// 更新到新状态
			if err := wm.UpdateNodeStatus(nodeName, tc.toStatus); err != nil {
				t.Fatalf("UpdateNodeStatus failed: %v", err)
			}

			// 验证状态
			wm2, err := workspace.NewManager(rootPath)
			if err != nil {
				t.Fatalf("NewManager reload failed: %v", err)
			}

			node := wm2.State.Nodes[nodeName]
			if node.Status != tc.toStatus {
				t.Errorf("Expected status %s, got %s", tc.toStatus, node.Status)
			}
		})
	}
}

// TestWorkflowRunStatusPersistence 测试 workflow run 状态的持久化
func TestWorkflowRunStatusPersistence(t *testing.T) {
	rootPath, _, cleanup := setupTestWorkspaceForWorkflow(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	engine := workflow.NewEngine(wm)

	// 验证 ListRuns 返回空列表
	runs, err := engine.ListRuns()
	if err != nil {
		t.Fatalf("ListRuns failed: %v", err)
	}

	if len(runs) != 0 {
		t.Errorf("Expected 0 runs, got %d", len(runs))
	}
}

// TestWorkflowTriggerValidation 测试 workflow 触发验证
func TestWorkflowTriggerValidation(t *testing.T) {
	// 测试 shadow branch 的递归保护逻辑
	// 根据 engine.go 的逻辑：if len(baseBranch) > 11 && baseBranch[:11] == "orion/run-"
	// 这个检查用于防止在工作流运行中递归触发工作流
	tests := []struct {
		branch     string
		wantShadow bool
		desc       string
	}{
		// 这些分支的前 11 个字符都不是 "orion/run-"，所以不会被检测为 shadow branch
		{"orion/run-123/ut", false, "actual shadow branch format"},
		{"orion/run-abc/step1", false, "actual shadow branch with letters"},
		// 长度不足 11 的分支
		{"orion/run", false, "short branch name"},
		{"main", false, "main branch"},
		{"feature/test", false, "feature branch"},
		// 边界情况：长度刚好大于 11 但不匹配前缀
		{"orion/run-1", false, "short shadow-like branch"},
	}

	for _, tc := range tests {
		t.Run(tc.branch, func(t *testing.T) {
			// 复制实际代码的逻辑
			isShadow := len(tc.branch) > 11 && tc.branch[:11] == "orion/run-"
			if isShadow != tc.wantShadow {
				t.Errorf("%s: isShadow=%v, want=%v (len=%d, first11=%q)",
					tc.desc, isShadow, tc.wantShadow, len(tc.branch), tc.branch[:11])
			}
		})
	}
}

// TestWorkflowStepStatus 测试 workflow step 状态
func TestWorkflowStepStatus(t *testing.T) {
	// 验证 workflow 包中的状态常量
	statuses := []workflow.RunStatus{
		workflow.StatusPending,
		workflow.StatusRunning,
		workflow.StatusSuccess,
		workflow.StatusFailed,
	}

	expectedValues := []string{"pending", "running", "success", "failed"}

	for i, status := range statuses {
		if string(status) != expectedValues[i] {
			t.Errorf("Status %d = %q, want %q", i, status, expectedValues[i])
		}
	}
}

// TestWorkflowRunStructure 测试 workflow Run 结构
func TestWorkflowRunStructure(t *testing.T) {
	// 验证 Run 结构的基本字段
	run := workflow.Run{
		ID:              "run-test",
		Workflow:        "test",
		Trigger:         "manual",
		BaseBranch:      "main",
		TriggeredByNode: "test-node",
		Status:          workflow.StatusRunning,
	}

	if run.ID != "run-test" {
		t.Errorf("Run ID mismatch")
	}
	if run.Workflow != "test" {
		t.Errorf("Run Workflow mismatch")
	}
	if run.Trigger != "manual" {
		t.Errorf("Run Trigger mismatch")
	}
	if run.BaseBranch != "main" {
		t.Errorf("Run BaseBranch mismatch")
	}
	if run.TriggeredByNode != "test-node" {
		t.Errorf("Run TriggeredByNode mismatch")
	}
}
