package workspace

import (
	"orion/internal/git"
	"orion/internal/types"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// Helper to create a temp workspace and repo
func setupTestWorkspace(t *testing.T) (*WorkspaceManager, func()) {
	t.Helper()

	// 1. Create root dir for workspace
	rootDir, err := os.MkdirTemp("", "orion-ws-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// 2. Initialize a separate git repo to serve as "remote"
	remoteDir, err := os.MkdirTemp("", "orion-remote-test")
	if err != nil {
		os.RemoveAll(rootDir)
		t.Fatalf("failed to create temp remote dir: %v", err)
	}

	// Initialize remote repo
	exec.Command("git", "init", remoteDir).Run()
	exec.Command("git", "-C", remoteDir, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", remoteDir, "config", "user.name", "Test User").Run()
	exec.Command("git", "-C", remoteDir, "checkout", "-b", "main").Run()
	os.WriteFile(filepath.Join(remoteDir, "README.md"), []byte("# Remote"), 0644)
	exec.Command("git", "-C", remoteDir, "add", ".").Run()
	exec.Command("git", "-C", remoteDir, "commit", "-m", "Initial commit").Run()

	// 3. Run Init (creates .orion, etc.)
	wm, err := Init(rootDir, remoteDir)
	if err != nil {
		os.RemoveAll(rootDir)
		os.RemoveAll(remoteDir)
		t.Fatalf("Init failed: %v", err)
	}

	// 4. Manually clone the repo as bare (simulating CLI behavior)
	if err := git.CloneBare(remoteDir, wm.State.RepoPath); err != nil {
		os.RemoveAll(rootDir)
		os.RemoveAll(remoteDir)
		t.Fatalf("CloneBare failed: %v", err)
	}

	// Configure user for local repo as well
	exec.Command("git", "-C", wm.State.RepoPath, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", wm.State.RepoPath, "config", "user.name", "Test User").Run()

	cleanup := func() {
		os.RemoveAll(rootDir)
		os.RemoveAll(remoteDir)
	}

	return wm, cleanup
}

func TestInit(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	// Verify directory structure
	if _, err := os.Stat(filepath.Join(wm.RootPath, MetaDir)); os.IsNotExist(err) {
		t.Errorf(".orion directory not created")
	}
	if _, err := os.Stat(filepath.Join(wm.RootPath, RepoDir)); os.IsNotExist(err) {
		t.Errorf("repo directory not created")
	}
	if _, err := os.Stat(filepath.Join(wm.RootPath, WorkspacesDir)); os.IsNotExist(err) {
		t.Errorf("workspaces directory not created")
	}

	// Verify V1 config files are generated
	if _, err := os.Stat(filepath.Join(wm.RootPath, MetaDir, ConfigFile)); os.IsNotExist(err) {
		t.Errorf("config.yaml not created")
	}

	// Verify GetConfig parses config.yaml correctly
	config, err := wm.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}
	if config.Git.MainBranch != "main" {
		t.Errorf("config.Git.MainBranch = %q, want %q", config.Git.MainBranch, "main")
	}
	if config.Workspace != "workspaces" {
		t.Errorf("config.Workspace = %q, want %q", config.Workspace, "workspaces")
	}

	// Verify state file
	if _, err := os.Stat(filepath.Join(wm.RootPath, MetaDir, StateFile)); os.IsNotExist(err) {
		t.Errorf("state.json not created")
	}
}

func TestSpawnAndRemoveNode(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	nodeName := "test-node"
	logicalBranch := "feature/login"

	// 1. Spawn Node
	err := wm.SpawnNode(nodeName, logicalBranch, "main", "Testing login", true)
	if err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// Verify state
	node, exists := wm.State.Nodes[nodeName]
	if !exists {
		t.Errorf("Node not found in state")
	}
	if node.BaseRef == "" {
		t.Errorf("expected BaseRef to be recorded")
	}
	if node.BaseCommit == "" {
		t.Errorf("expected BaseCommit to be recorded")
	}
	if node.HeadBranch == "" {
		t.Errorf("expected HeadBranch to be recorded")
	}

	// Verify worktree exists
	if _, err := os.Stat(node.WorktreePath); os.IsNotExist(err) {
		t.Errorf("Worktree directory not created at %s", node.WorktreePath)
	}

	// Verify shadow branch exists
	if err := git.VerifyBranch(wm.State.RepoPath, node.ShadowBranch); err != nil {
		t.Errorf("Shadow branch not created")
	}

	// 2. Remove Node
	err = wm.RemoveNode(nodeName)
	if err != nil {
		t.Fatalf("RemoveNode failed: %v", err)
	}

	// Verify state removed
	if _, exists := wm.State.Nodes[nodeName]; exists {
		t.Errorf("Node still exists in state after removal")
	}

	// Verify worktree removed (directory might be gone or empty)
	if _, err := os.Stat(node.WorktreePath); !os.IsNotExist(err) {
		// If it exists, it should be empty
		entries, _ := os.ReadDir(node.WorktreePath)
		if len(entries) > 0 {
			t.Errorf("Worktree directory not cleaned up")
		}
	}
}

func TestMergeNode(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	nodeName := "merge-node"
	logicalBranch := "feature/merge-test"

	// 1. Spawn
	wm.SpawnNode(nodeName, logicalBranch, "main", "Merge Test", true)
	node := wm.State.Nodes[nodeName]

	// 2. Make changes in the node's worktree
	newFile := filepath.Join(node.WorktreePath, "new-feature.txt")
	os.WriteFile(newFile, []byte("content"), 0644)

	exec.Command("git", "-C", node.WorktreePath, "add", ".").Run()
	exec.Command("git", "-C", node.WorktreePath, "commit", "-m", "Work in node").Run()

	// 3. Merge
	err := wm.MergeNode(nodeName, false)
	if err != nil {
		t.Fatalf("MergeNode failed: %v", err)
	}

	// 4. Verify changes in logicalBranch via bare repo object lookup.
	out, err := exec.Command("git", "-C", wm.State.RepoPath, "show", logicalBranch+":new-feature.txt").CombinedOutput()
	if err != nil {
		t.Fatalf("failed to inspect merged file from bare repo: %v, output: %s", err, string(out))
	}
	if string(out) != "content" {
		t.Errorf("unexpected merged file content: got %q, want %q", string(out), "content")
	}
}

func TestSpawnNodeFeatureMode(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	nodeName := "feature-node"
	logicialBranch := "feature/feature-mode"

	// Feature mode: isShadow=false should create a worktree directly on the logical branch.
	if err := wm.SpawnNode(nodeName, logicialBranch, "main", "Feature mode", false); err != nil {
		t.Fatalf("SpawnNode (feature mode) failed: %v", err)
	}

	node, exists := wm.State.Nodes[nodeName]
	if !exists {
		t.Fatalf("node not found in state after SpawnNode")
	}

	if node.ShadowBranch != logicialBranch {
		t.Errorf("expected shadow branch to equal logical branch, got %q", node.ShadowBranch)
	}

	if _, err := os.Stat(node.WorktreePath); os.IsNotExist(err) {
		t.Errorf("worktree directory not created at %s", node.WorktreePath)
	}

	// logical branch should exist in the main repo
	if err := git.VerifyBranch(wm.State.RepoPath, logicialBranch); err != nil {
		t.Errorf("logical branch %q not created in repo: %v", logicialBranch, err)
	}
}

func TestGetConfig(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	config, err := wm.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig returned error: %v", err)
	}

	if config.Workspace == "" {
		t.Errorf("expected workspace field to be non-empty")
	}
	if config.Git.MainBranch != "main" {
		t.Errorf("expected git.main_branch to be 'main', got %q", config.Git.MainBranch)
	}
}

func TestFindNodeByPath(t *testing.T) {
	// Keep the original unit test logic but maybe use the helper if we want integration test?
	// The original test used a mock state. Let's keep it simple and just use a struct literal state like before,
	// because creating real worktrees for path testing is slow/overkill.

	// Setup a mock workspace manager
	wm := &WorkspaceManager{
		State: &types.State{
			Nodes: map[string]types.Node{
				"node1": {
					Name:         "node1",
					WorktreePath: "/Users/user/orion_ws/nodes/node1",
				},
				"node2": {
					Name:         "node2",
					WorktreePath: "/Users/user/orion_ws/nodes/node2",
				},
			},
		},
	}

	tests := []struct {
		name      string
		inputPath string
		wantNode  string
		wantFound bool
	}{
		{
			name:      "Exact match file inside node",
			inputPath: "/Users/user/orion_ws/nodes/node1/main.go",
			wantNode:  "node1",
			wantFound: true,
		},
		{
			name:      "Exact match directory inside node",
			inputPath: "/Users/user/orion_ws/nodes/node1/src",
			wantNode:  "node1",
			wantFound: true,
		},
		{
			name:      "Path outside nodes",
			inputPath: "/Users/user/orion_ws/repo/main.go",
			wantNode:  "",
			wantFound: false,
		},
		{
			name:      "Partial prefix match (should fail)",
			inputPath: "/Users/user/orion_ws/nodes/node1-suffix/main.go",
			wantNode:  "",
			wantFound: false,
		},
	}

	// Add case-insensitive tests for macOS/Windows
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		wm.State.Nodes["node_lower"] = types.Node{
			Name:         "node_lower",
			WorktreePath: "/users/user/orion_ws/nodes/node_lower",
		}

		tests = append(tests, struct {
			name      string
			inputPath string
			wantNode  string
			wantFound bool
		}{
			name:      "Case mismatch on macOS (Input mixed, Node lower)",
			inputPath: "/Users/User/Orion_ws/Nodes/node_lower/main.go",
			wantNode:  "node_lower",
			wantFound: true,
		}, struct {
			name      string
			inputPath string
			wantNode  string
			wantFound bool
		}{
			name:      "Case mismatch on macOS (Input lower, Node mixed)",
			inputPath: "/users/user/orion_ws/nodes/node1/main.go",
			wantNode:  "node1",
			wantFound: true,
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: This test relies on filepath.Abs and EvalSymlinks which hit the FS.
			// Since we are using fake paths, this might fail if we don't mock them.
			// However, FindNodeByPath logic handles errors from EvalSymlinks by falling back.
			// So it should work for string comparison logic mostly.

			gotNode, _, _ := wm.FindNodeByPath(tt.inputPath)
			if gotNode != tt.wantNode {
				// On Linux, paths that don't exist might behave differently with Abs/Rel
				// But let's see. If it fails, we know we need to mock FS.
				// For now, let's allow it to fail if it must, but ideally we should only run FS tests on real FS.
				// But this specific test block is testing string logic.

				// ACTUALLY: filepath.Abs works on string. EvalSymlinks fails if not exist.
				// The code:
				// canonicalPath, err := filepath.EvalSymlinks(absPath)
				// if err != nil { canonicalPath = absPath }
				// So it falls back to absPath.
				// Then: rel, err := filepath.Rel(nodePath, canonicalPath)
				// This should work fine for fake paths.

				t.Errorf("FindNodeByPath(%q) = %v, want %v", tt.inputPath, gotNode, tt.wantNode)
			}
		})
	}
}

func TestAppliedRunsPersistence(t *testing.T) {
	wm, cleanup := setupTestWorkspace(t)
	defer cleanup()

	nodeName := "test-node-applied"
	logicalBranch := "feature/applied"

	// 1. Spawn Node
	// Note: SpawnNode now takes (nodeName, logicalBranch, baseBranch, label, isShadow)
	err := wm.SpawnNode(nodeName, logicalBranch, "main", "Testing persistence", true)
	if err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// 2. Modify node state (add applied runs)
	node := wm.State.Nodes[nodeName]
	node.AppliedRuns = []string{"run-1", "run-2"}
	wm.State.Nodes[nodeName] = node

	// 3. Save State
	if err := wm.SaveState(); err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	// 4. Reload Manager
	wm2, err := NewManager(wm.RootPath)
	if err != nil {
		t.Fatalf("Failed to reload manager: %v", err)
	}

	loadedNode, exists := wm2.State.Nodes[nodeName]
	if !exists {
		t.Fatalf("Node not found after reload")
	}

	if len(loadedNode.AppliedRuns) != 2 {
		t.Errorf("Expected 2 applied runs, got %d", len(loadedNode.AppliedRuns))
	}
	if loadedNode.AppliedRuns[0] != "run-1" || loadedNode.AppliedRuns[1] != "run-2" {
		t.Errorf("AppliedRuns content mismatch: %v", loadedNode.AppliedRuns)
	}
}
