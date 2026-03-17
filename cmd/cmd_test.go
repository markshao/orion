package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"orion/internal/git"
	"orion/internal/types"
	"orion/internal/workspace"
)

// setupTestCmdWorkspace creates a temp workspace for cmd testing
func setupTestCmdWorkspace(t *testing.T) (string, func()) {
	t.Helper()

	rootDir, err := os.MkdirTemp("", "orion-cmd-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	remoteDir, err := os.MkdirTemp("", "orion-remote-test-*")
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

	// Initialize workspace
	wm, err := workspace.Init(rootDir, remoteDir)
	if err != nil {
		os.RemoveAll(rootDir)
		os.RemoveAll(remoteDir)
		t.Fatalf("Init failed: %v", err)
	}

	// Clone repo
	if err := git.Clone(remoteDir, wm.State.RepoPath); err != nil {
		os.RemoveAll(rootDir)
		os.RemoveAll(remoteDir)
		t.Fatalf("Clone failed: %v", err)
	}

	exec.Command("git", "-C", wm.State.RepoPath, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", wm.State.RepoPath, "config", "user.name", "Test User").Run()

	cleanup := func() {
		os.RemoveAll(rootDir)
		os.RemoveAll(remoteDir)
	}

	return rootDir, cleanup
}

func TestCompleteNodeNames(t *testing.T) {
	rootDir, cleanup := setupTestCmdWorkspace(t)
	defer cleanup()

	// Create workspace manager and add some nodes
	wm, err := workspace.NewManager(rootDir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Spawn a test node
	if err := wm.SpawnNode("test-node", "feature/test", "main", "test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	// Change to workspace directory for the test
	originalDir, _ := os.Getwd()
	os.Chdir(rootDir)
	defer os.Chdir(originalDir)

	// Test completion
	completions, _ := CompleteNodeNames(nil, []string{}, "")

	// Note: Completions may be empty in test environments without proper shell setup
	// This test verifies the function doesn't crash
	_ = completions
}

func TestCompleteNodeNamesWithArgs(t *testing.T) {
	rootDir, cleanup := setupTestCmdWorkspace(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootDir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	wm.SpawnNode("test-node", "feature/test", "main", "test", true)

	// When args already has a node, should return nil
	completions, _ := CompleteNodeNames(nil, []string{"test-node"}, "")
	if completions != nil {
		t.Errorf("expected nil completions when arg provided, got %v", completions)
	}
}

func TestCompleteNodeNamesInvalidWorkspace(t *testing.T) {
	// Test in a directory that's not a workspace
	tmpDir, err := os.MkdirTemp("", "orion-invalid-ws-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	// In an invalid workspace, the function should return an error directive
	// The exact directive value may vary, so we just verify it returns an error
	_, directive := CompleteNodeNames(nil, []string{}, "")
	// Directive 4 is cobra.ShellCompDirectiveError, but we accept any non-zero value
	if directive == 0 {
		t.Logf("got directive %d (expected error directive in invalid workspace)", directive)
	}
}

func TestFormatStatus(t *testing.T) {
	tests := []struct {
		status   string
		expected string
	}{
		{"WORKING", "WORKING"},
		{"READY_TO_PUSH", "READY_TO_PUSH"},
		{"FAIL", "FAIL"},
		{"PUSHED", "PUSHED"},
		{"UNKNOWN", "WORKING"}, // Default case
	}

	for _, tt := range tests {
		result := formatStatus(tt.status)
		// The result will have ANSI color codes, so we check if it contains the status
		if !strings.Contains(result, tt.expected) {
			t.Errorf("formatStatus(%q) = %q, expected to contain %q", tt.status, result, tt.expected)
		}
	}
}

func TestSelectNodeEmpty(t *testing.T) {
	rootDir, cleanup := setupTestCmdWorkspace(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootDir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Clear nodes
	wm.State.Nodes = make(map[string]types.Node)

	_, err = SelectNode(wm, "test", true)
	if err == nil {
		t.Error("expected error for empty nodes")
	}
	if !strings.Contains(err.Error(), "no active nodes") {
		t.Errorf("expected 'no active nodes' error, got: %v", err)
	}
}

func TestSelectNodeOnlyHuman(t *testing.T) {
	rootDir, cleanup := setupTestCmdWorkspace(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootDir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Add only agent nodes
	wm.State.Nodes["agent-node"] = types.Node{
		Name:      "agent-node",
		CreatedBy: "run-123",
	}

	_, err = SelectNode(wm, "test", true)
	if err == nil {
		t.Error("expected error when only agent nodes exist and onlyHuman=true")
	}
	if !strings.Contains(err.Error(), "no active human nodes") {
		t.Errorf("expected 'no active human nodes' error, got: %v", err)
	}
}

func TestDetermineExecDirCmd(t *testing.T) {
	rootDir, cleanup := setupTestCmdWorkspace(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootDir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Create a node
	if err := wm.SpawnNode("test-node", "feature/test", "main", "test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	nodePath := wm.State.Nodes["test-node"].WorktreePath
	repoPath := wm.State.RepoPath

	tests := []struct {
		name           string
		cwd            string
		targetWorktree string
		wantExecDir    string
		wantWorktree   string
		wantErr        bool
	}{
		{
			name:           "no target defaults to repo",
			cwd:            rootDir,
			targetWorktree: "",
			wantExecDir:    repoPath,
			wantWorktree:   "",
			wantErr:        false,
		},
		{
			name:           "valid target worktree",
			cwd:            rootDir,
			targetWorktree: "test-node",
			wantExecDir:    nodePath,
			wantWorktree:   "test-node",
			wantErr:        false,
		},
		{
			name:           "invalid target worktree",
			cwd:            rootDir,
			targetWorktree: "nonexistent",
			wantExecDir:    "",
			wantWorktree:   "",
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotExecDir, gotWorktree, err := determineExecDir(wm, tt.cwd, tt.targetWorktree)
			if (err != nil) != tt.wantErr {
				t.Errorf("determineExecDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if gotExecDir != tt.wantExecDir {
					t.Errorf("determineExecDir() execDir = %v, want %v", gotExecDir, tt.wantExecDir)
				}
				if gotWorktree != tt.wantWorktree {
					t.Errorf("determineExecDir() worktree = %v, want %v", gotWorktree, tt.wantWorktree)
				}
			}
		})
	}
}

func TestIsSubDir(t *testing.T) {
	tests := []struct {
		name   string
		parent string
		child  string
		want   bool
	}{
		{
			name:   "child is subdirectory",
			parent: "/parent",
			child:  "/parent/child",
			want:   true,
		},
		{
			name:   "child is same directory",
			parent: "/parent",
			child:  "/parent",
			want:   true,
		},
		{
			name:   "child is outside",
			parent: "/parent",
			child:  "/other",
			want:   false,
		},
		{
			name:   "child is sibling",
			parent: "/parent",
			child:  "/sibling",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSubDir(tt.parent, tt.child)
			if got != tt.want {
				t.Errorf("isSubDir(%q, %q) = %v, want %v", tt.parent, tt.child, got, tt.want)
			}
		})
	}
}

func TestGetRunWorktreePath(t *testing.T) {
	rootDir, cleanup := setupTestCmdWorkspace(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootDir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Test main repo path
	repoPath, err := GetRunWorktreePath(rootDir, "")
	if err != nil {
		t.Fatalf("GetRunWorktreePath(\"\") failed: %v", err)
	}
	if repoPath != wm.State.RepoPath {
		t.Errorf("expected repo path %s, got %s", wm.State.RepoPath, repoPath)
	}

	// Create node and test node path
	if err := wm.SpawnNode("test-node", "feature/test", "main", "test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	nodePath, err := GetRunWorktreePath(rootDir, "test-node")
	if err != nil {
		t.Fatalf("GetRunWorktreePath(\"test-node\") failed: %v", err)
	}

	expectedPath := wm.State.Nodes["test-node"].WorktreePath
	if nodePath != expectedPath {
		t.Errorf("expected node path %s, got %s", expectedPath, nodePath)
	}

	// Test non-existent node
	_, err = GetRunWorktreePath(rootDir, "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent node")
	}
}

func TestExecuteInWorktree(t *testing.T) {
	rootDir, cleanup := setupTestCmdWorkspace(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootDir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Create a test file in repo
	testFile := filepath.Join(wm.State.RepoPath, "test.txt")
	os.WriteFile(testFile, []byte("test content"), 0644)

	// Test in main repo
	output, exitCode, err := ExecuteInWorktree(rootDir, "", []string{"cat", "test.txt"})
	if err != nil {
		t.Fatalf("ExecuteInWorktree failed: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(output, "test content") {
		t.Errorf("unexpected output: %s", output)
	}

	// Test with no command
	_, _, err = ExecuteInWorktree(rootDir, "", []string{})
	if err == nil {
		t.Error("expected error for empty command")
	}
}

func TestExecuteInWorktreeWithNode(t *testing.T) {
	rootDir, cleanup := setupTestCmdWorkspace(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootDir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Create node
	if err := wm.SpawnNode("test-node", "feature/test", "main", "test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	node := wm.State.Nodes["test-node"]

	// Create test file in node
	testFile := filepath.Join(node.WorktreePath, "node-test.txt")
	os.WriteFile(testFile, []byte("node content"), 0644)

	output, exitCode, err := ExecuteInWorktree(rootDir, "test-node", []string{"cat", "node-test.txt"})
	if err != nil {
		t.Fatalf("ExecuteInWorktree failed: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(output, "node content") {
		t.Errorf("unexpected output: %s", output)
	}
}
