package workspace

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"orion/internal/git"
	"orion/internal/types"
)

// setupTestWorkspace 创建一个用于测试的临时 workspace
func setupTestWorkspace(t *testing.T) (rootPath, repoPath, remotePath string, cleanup func()) {
	t.Helper()

	// 1. 创建 root dir
	rootDir, err := os.MkdirTemp("", "orion-workspace-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// 2. 创建 remote repo（bare repository）
	remoteDir, err := os.MkdirTemp("", "orion-remote-test")
	if err != nil {
		os.RemoveAll(rootDir)
		t.Fatalf("failed to create temp remote dir: %v", err)
	}
	exec.Command("git", "init", "--bare", remoteDir).Run()

	// 3. 初始化 workspace
	wm, err := Init(rootDir, remoteDir)
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

	// 创建初始 commit
	testFile := filepath.Join(wm.State.RepoPath, "README.md")
	os.WriteFile(testFile, []byte("# Test Repo"), 0644)
	exec.Command("git", "-C", wm.State.RepoPath, "add", ".").Run()
	exec.Command("git", "-C", wm.State.RepoPath, "commit", "-m", "Initial commit").Run()
	exec.Command("git", "-C", wm.State.RepoPath, "push", "-u", "origin", "main").Run()

	cleanup = func() {
		os.RemoveAll(rootDir)
		os.RemoveAll(remoteDir)
	}

	return rootDir, wm.State.RepoPath, remoteDir, cleanup
}

// TestInit 测试初始化 workspace
func TestInit(t *testing.T) {
	rootPath, _, _, cleanup := setupTestWorkspace(t)
	defer cleanup()

	// 验证目录结构
	expectedDirs := []string{
		filepath.Join(rootPath, RepoDir),
		filepath.Join(rootPath, WorkspacesDir),
		filepath.Join(rootPath, MetaDir),
		filepath.Join(rootPath, MetaDir, WorkflowsDir),
		filepath.Join(rootPath, MetaDir, AgentsDir),
		filepath.Join(rootPath, MetaDir, PromptsDir),
		filepath.Join(rootPath, MetaDir, RunsDir),
	}

	for _, dir := range expectedDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("directory %s should exist", dir)
		}
	}

	// 验证配置文件
	configPath := filepath.Join(rootPath, MetaDir, ConfigFile)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("config file %s should exist", configPath)
	}

	// 验证 state 文件
	statePath := filepath.Join(rootPath, MetaDir, StateFile)
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Errorf("state file %s should exist", statePath)
	}

	// 验证 workflow 文件
	workflowPath := filepath.Join(rootPath, MetaDir, WorkflowsDir, "default.yaml")
	if _, err := os.Stat(workflowPath); os.IsNotExist(err) {
		t.Errorf("workflow file %s should exist", workflowPath)
	}

	// 验证 agent 文件
	agentPaths := []string{
		filepath.Join(rootPath, MetaDir, AgentsDir, "ut-agent.yaml"),
		filepath.Join(rootPath, MetaDir, AgentsDir, "cr-agent.yaml"),
	}
	for _, path := range agentPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("agent file %s should exist", path)
		}
	}

	// 验证 prompt 文件
	promptPaths := []string{
		filepath.Join(rootPath, MetaDir, PromptsDir, "ut.md"),
		filepath.Join(rootPath, MetaDir, PromptsDir, "cr.md"),
		filepath.Join(rootPath, MetaDir, PromptsDir, "base.md"),
	}
	for _, path := range promptPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("prompt file %s should exist", path)
		}
	}
}

// TestNewManager 测试创建 manager
func TestNewManager(t *testing.T) {
	rootPath, _, _, cleanup := setupTestWorkspace(t)
	defer cleanup()

	// 测试成功创建
	wm, err := NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	if wm.RootPath != rootPath {
		t.Errorf("RootPath = %q, want %q", wm.RootPath, rootPath)
	}

	if wm.State == nil {
		t.Error("State should not be nil")
	}

	// 测试无效路径
	_, err = NewManager("/non/existent/path")
	if err == nil {
		t.Error("NewManager should fail for non-existent path")
	}
}

// TestFindWorkspaceRoot 测试查找 workspace root
func TestFindWorkspaceRoot(t *testing.T) {
	rootPath, _, _, cleanup := setupTestWorkspace(t)
	defer cleanup()

	// 测试在 root 目录
	found, err := FindWorkspaceRoot(rootPath)
	if err != nil {
		t.Fatalf("FindWorkspaceRoot failed: %v", err)
	}
	if found != rootPath {
		t.Errorf("FindWorkspaceRoot = %q, want %q", found, rootPath)
	}

	// 测试在子目录
	subDir := filepath.Join(rootPath, "some", "nested", "dir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	found, err = FindWorkspaceRoot(subDir)
	if err != nil {
		t.Fatalf("FindWorkspaceRoot failed: %v", err)
	}
	if found != rootPath {
		t.Errorf("FindWorkspaceRoot from subdir = %q, want %q", found, rootPath)
	}

	// 测试在非 workspace 目录
	tempDir, err := os.MkdirTemp("", "orion-non-workspace")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	_, err = FindWorkspaceRoot(tempDir)
	if err == nil {
		t.Error("FindWorkspaceRoot should fail for non-workspace directory")
	}
}

// TestSaveAndLoadState 测试保存和加载状态
func TestSaveAndLoadState(t *testing.T) {
	rootPath, _, _, cleanup := setupTestWorkspace(t)
	defer cleanup()

	wm, err := NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// 添加节点
	wm.State.Nodes["test-node"] = types.Node{
		Name:          "test-node",
		LogicalBranch: "feature/test",
		ShadowBranch:  "orion-shadow/test-node/feature/test",
		WorktreePath:  "/path/to/worktree",
		CreatedBy:     "user",
		Status:        types.StatusWorking,
	}

	// 保存状态
	if err := wm.SaveState(); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// 创建新的 manager 并加载状态
	wm2, err := NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// 验证加载的节点
	node, exists := wm2.State.Nodes["test-node"]
	if !exists {
		t.Error("test-node should exist in loaded state")
	}
	if node.Name != "test-node" {
		t.Errorf("node.Name = %q, want %q", node.Name, "test-node")
	}
	if node.LogicalBranch != "feature/test" {
		t.Errorf("node.LogicalBranch = %q, want %q", node.LogicalBranch, "feature/test")
	}
	if node.Status != types.StatusWorking {
		t.Errorf("node.Status = %q, want %q", node.Status, types.StatusWorking)
	}
}

// TestSpawnNode 测试创建节点
func TestSpawnNode(t *testing.T) {
	rootPath, _, _, cleanup := setupTestWorkspace(t)
	defer cleanup()

	wm, err := NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// 测试创建 shadow 模式节点
	nodeName := "test-spawn-node"
	logicalBranch := "feature/spawn-test"
	baseBranch := "main"
	label := "test-label"

	err = wm.SpawnNode(nodeName, logicalBranch, baseBranch, label, true)
	if err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// 验证节点已创建
	node, exists := wm.State.Nodes[nodeName]
	if !exists {
		t.Fatal("node should exist after SpawnNode")
	}

	if node.Name != nodeName {
		t.Errorf("node.Name = %q, want %q", node.Name, nodeName)
	}
	if node.LogicalBranch != logicalBranch {
		t.Errorf("node.LogicalBranch = %q, want %q", node.LogicalBranch, logicalBranch)
	}
	if node.BaseBranch != baseBranch {
		t.Errorf("node.BaseBranch = %q, want %q", node.BaseBranch, baseBranch)
	}
	if node.Label != label {
		t.Errorf("node.Label = %q, want %q", node.Label, label)
	}
	if node.CreatedBy != "user" {
		t.Errorf("node.CreatedBy = %q, want %q", node.CreatedBy, "user")
	}
	if node.Status != types.StatusWorking {
		t.Errorf("node.Status = %q, want %q", node.Status, types.StatusWorking)
	}

	// 验证 worktree 路径存在
	if _, err := os.Stat(node.WorktreePath); os.IsNotExist(err) {
		t.Errorf("worktree path %s should exist", node.WorktreePath)
	}

	// 验证 shadow 分支命名
	expectedShadowBranch := "orion-shadow/" + nodeName + "/" + logicalBranch
	if node.ShadowBranch != expectedShadowBranch {
		t.Errorf("node.ShadowBranch = %q, want %q", node.ShadowBranch, expectedShadowBranch)
	}
}

// TestSpawnNodeDuplicate 测试创建重复节点
func TestSpawnNodeDuplicate(t *testing.T) {
	rootPath, _, _, cleanup := setupTestWorkspace(t)
	defer cleanup()

	wm, err := NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	nodeName := "test-dup-node"

	// 创建第一个节点
	err = wm.SpawnNode(nodeName, "feature/dup", "main", "", true)
	if err != nil {
		t.Fatalf("first SpawnNode failed: %v", err)
	}

	// 尝试创建同名节点
	err = wm.SpawnNode(nodeName, "feature/dup2", "main", "", true)
	if err == nil {
		t.Error("SpawnNode should fail for duplicate node name")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error should mention 'already exists': %v", err)
	}
}

// TestUpdateNodeStatus 测试更新节点状态
func TestUpdateNodeStatus(t *testing.T) {
	rootPath, _, _, cleanup := setupTestWorkspace(t)
	defer cleanup()

	wm, err := NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	nodeName := "test-status-node"
	err = wm.SpawnNode(nodeName, "feature/status", "main", "", true)
	if err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// 测试更新状态
	err = wm.UpdateNodeStatus(nodeName, types.StatusReadyToPush)
	if err != nil {
		t.Fatalf("UpdateNodeStatus failed: %v", err)
	}

	// 验证状态已更新
	node, exists := wm.State.Nodes[nodeName]
	if !exists {
		t.Fatal("node should exist")
	}
	if node.Status != types.StatusReadyToPush {
		t.Errorf("node.Status = %q, want %q", node.Status, types.StatusReadyToPush)
	}

	// 测试更新不存在的节点
	err = wm.UpdateNodeStatus("non-existent", types.StatusPushed)
	if err == nil {
		t.Error("UpdateNodeStatus should fail for non-existent node")
	}
}

// TestRemoveNode 测试删除节点
func TestRemoveNode(t *testing.T) {
	rootPath, _, _, cleanup := setupTestWorkspace(t)
	defer cleanup()

	wm, err := NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	nodeName := "test-remove-node"
	err = wm.SpawnNode(nodeName, "feature/remove", "main", "", true)
	if err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// 验证节点存在
	_, exists := wm.State.Nodes[nodeName]
	if !exists {
		t.Fatal("node should exist before removal")
	}

	// 删除节点
	err = wm.RemoveNode(nodeName)
	if err != nil {
		t.Fatalf("RemoveNode failed: %v", err)
	}

	// 验证节点已删除
	_, exists = wm.State.Nodes[nodeName]
	if exists {
		t.Error("node should not exist after removal")
	}
}

// TestFindNodeByPath 测试通过路径查找节点
func TestFindNodeByPath(t *testing.T) {
	rootPath, _, _, cleanup := setupTestWorkspace(t)
	defer cleanup()

	wm, err := NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	nodeName := "test-path-node"
	err = wm.SpawnNode(nodeName, "feature/path", "main", "", true)
	if err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	node := wm.State.Nodes[nodeName]

	// 测试在节点根目录
	foundName, foundNode, err := wm.FindNodeByPath(node.WorktreePath)
	if err != nil {
		t.Fatalf("FindNodeByPath failed: %v", err)
	}
	if foundName != nodeName {
		t.Errorf("FindNodeByPath = %q, want %q", foundName, nodeName)
	}
	if foundNode == nil {
		t.Error("FindNodeByPath should return non-nil node")
	}

	// 测试在节点子目录
	subDir := filepath.Join(node.WorktreePath, "sub", "dir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	foundName, foundNode, err = wm.FindNodeByPath(subDir)
	if err != nil {
		t.Fatalf("FindNodeByPath failed: %v", err)
	}
	if foundName != nodeName {
		t.Errorf("FindNodeByPath from subdir = %q, want %q", foundName, nodeName)
	}

	// 测试不在任何节点的路径
	outsidePath := filepath.Join(rootPath, "outside", "path")
	if err := os.MkdirAll(outsidePath, 0755); err != nil {
		t.Fatalf("failed to create outside path: %v", err)
	}

	foundName, foundNode, err = wm.FindNodeByPath(outsidePath)
	if err != nil {
		t.Fatalf("FindNodeByPath failed: %v", err)
	}
	if foundName != "" {
		t.Errorf("FindNodeByPath should return empty name for outside path, got %q", foundName)
	}
	if foundNode != nil {
		t.Error("FindNodeByPath should return nil node for outside path")
	}
}

// TestSyncVSCodeWorkspace 测试同步 VSCode workspace
func TestSyncVSCodeWorkspace(t *testing.T) {
	rootPath, _, _, cleanup := setupTestWorkspace(t)
	defer cleanup()

	wm, err := NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// 创建节点
	err = wm.SpawnNode("node1", "feature/one", "main", "", true)
	if err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}
	err = wm.SpawnNode("node2", "feature/two", "main", "", true)
	if err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// 同步 VSCode workspace
	err = wm.SyncVSCodeWorkspace()
	if err != nil {
		t.Fatalf("SyncVSCodeWorkspace failed: %v", err)
	}

	// 验证 workspace 文件已创建
	projectName := filepath.Base(rootPath)
	projectName = strings.TrimSuffix(projectName, "_swarm")
	workspaceFile := filepath.Join(rootPath, projectName+".code-workspace")

	if _, err := os.Stat(workspaceFile); os.IsNotExist(err) {
		t.Errorf("workspace file %s should exist", workspaceFile)
	}
}

// TestGetConfig 测试获取配置
func TestGetConfig(t *testing.T) {
	rootPath, _, _, cleanup := setupTestWorkspace(t)
	defer cleanup()

	wm, err := NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	config, err := wm.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}

	if config.Version != 1 {
		t.Errorf("Config.Version = %d, want %d", config.Version, 1)
	}
	if config.Git.MainBranch != "main" {
		t.Errorf("Config.Git.MainBranch = %q, want %q", config.Git.MainBranch, "main")
	}
}
