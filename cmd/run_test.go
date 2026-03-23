package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"orion/internal/git"
	"orion/internal/workspace"
)

func setupTestWorkspaceForRun(t *testing.T) (rootPath, repoPath string, cleanup func()) {
	t.Helper()

	rootDir, err := os.MkdirTemp("", "orion-run-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	remoteDir, err := os.MkdirTemp("", "orion-remote-test")
	if err != nil {
		os.RemoveAll(rootDir)
		t.Fatalf("failed to create temp remote dir: %v", err)
	}

	exec.Command("git", "init", remoteDir).Run()
	exec.Command("git", "-C", remoteDir, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", remoteDir, "config", "user.name", "Test User").Run()
	exec.Command("git", "-C", remoteDir, "checkout", "-b", "main").Run()
	os.WriteFile(filepath.Join(remoteDir, "README.md"), []byte("# Remote"), 0644)
	exec.Command("git", "-C", remoteDir, "add", ".").Run()
	exec.Command("git", "-C", remoteDir, "commit", "-m", "Initial commit").Run()

	wm, err := workspace.Init(rootDir, remoteDir)
	if err != nil {
		os.RemoveAll(rootDir)
		os.RemoveAll(remoteDir)
		t.Fatalf("Init failed: %v", err)
	}

	if err := git.CloneBare(remoteDir, wm.State.RepoPath); err != nil {
		os.RemoveAll(rootDir)
		os.RemoveAll(remoteDir)
		t.Fatalf("CloneBare failed: %v", err)
	}
	if err := git.Fetch(wm.State.RepoPath); err != nil {
		os.RemoveAll(rootDir)
		os.RemoveAll(remoteDir)
		t.Fatalf("Fetch failed: %v", err)
	}

	exec.Command("git", "-C", wm.State.RepoPath, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", wm.State.RepoPath, "config", "user.name", "Test User").Run()

	cleanup = func() {
		os.RemoveAll(rootDir)
		os.RemoveAll(remoteDir)
	}

	return rootDir, wm.State.RepoPath, cleanup
}

func TestIsGitCommand(t *testing.T) {
	if !isGitCommand([]string{"git", "status"}) {
		t.Fatal("expected git command to be allowed")
	}
	if isGitCommand([]string{"make", "test"}) {
		t.Fatal("expected non-git command to be rejected")
	}
	if isGitCommand(nil) {
		t.Fatal("expected empty command to be rejected")
	}
}

func TestRunInBareRepoAllowsGitCommands(t *testing.T) {
	rootPath, _, cleanup := setupTestWorkspaceForRun(t)
	defer cleanup()

	output, exitCode, err := ExecuteInWorktree(rootPath, "", []string{"git", "rev-parse", "--is-bare-repository"})
	if err != nil {
		t.Fatalf("ExecuteInWorktree failed: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, output: %s", exitCode, output)
	}
	if !strings.Contains(output, "true") {
		t.Fatalf("expected bare repo output, got: %s", output)
	}
}

func TestRunInBareRepoRejectsNonGitCommands(t *testing.T) {
	rootPath, _, cleanup := setupTestWorkspaceForRun(t)
	defer cleanup()

	output, exitCode, err := ExecuteInWorktree(rootPath, "", []string{"cat", "README.md"})
	if err != nil {
		t.Fatalf("ExecuteInWorktree returned unexpected error: %v", err)
	}
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(output, "bare repo context only supports git commands") {
		t.Fatalf("unexpected output: %s", output)
	}
}

func TestRunInWorktreeAllowsFileTreeCommands(t *testing.T) {
	rootPath, _, cleanup := setupTestWorkspaceForRun(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	nodeName := "test-node"
	if err := wm.SpawnNode(nodeName, "feature/test", "main", "test", true); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	node := wm.State.Nodes[nodeName]
	testFile := filepath.Join(node.WorktreePath, "worktree_file.txt")
	if err := os.WriteFile(testFile, []byte("hello from worktree"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	output, exitCode, err := ExecuteInWorktree(rootPath, nodeName, []string{"cat", "worktree_file.txt"})
	if err != nil {
		t.Fatalf("ExecuteInWorktree failed: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(output, "hello from worktree") {
		t.Fatalf("unexpected output: %s", output)
	}
}

func TestRunWithNonExistentWorktree(t *testing.T) {
	rootPath, _, cleanup := setupTestWorkspaceForRun(t)
	defer cleanup()

	_, err := GetRunWorktreePath(rootPath, "non-existent-node")
	if err == nil {
		t.Fatal("expected error for non-existent node")
	}
	if !strings.Contains(err.Error(), "non-existent-node") {
		t.Fatalf("error message should mention node name: %v", err)
	}
}

func TestRunExitCodePropagation(t *testing.T) {
	rootPath, _, cleanup := setupTestWorkspaceForRun(t)
	defer cleanup()

	_, exitCode, err := ExecuteInWorktree(rootPath, "", []string{"git", "show", "does-not-exist"})
	if err != nil {
		t.Fatalf("unexpected execution error: %v", err)
	}
	if exitCode == 0 {
		t.Fatal("expected non-zero exit code for failing git command")
	}

	_, exitCode, err = ExecuteInWorktree(rootPath, "", []string{"git", "rev-parse", "--is-bare-repository"})
	if err != nil {
		t.Fatalf("unexpected execution error: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
}

func TestRunInNestedDirectoryDefaultsToBareRepo(t *testing.T) {
	rootPath, repoPath, cleanup := setupTestWorkspaceForRun(t)
	defer cleanup()

	subDir := filepath.Join(rootPath, "some", "nested", "dir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}

	foundRoot, err := workspace.FindWorkspaceRoot(subDir)
	if err != nil {
		t.Fatalf("FindWorkspaceRoot failed: %v", err)
	}
	if foundRoot != rootPath {
		t.Fatalf("expected root %s, got %s", rootPath, foundRoot)
	}

	output, exitCode, err := ExecuteInWorktree(rootPath, "", []string{"git", "rev-parse", "--absolute-git-dir"})
	if err != nil {
		t.Fatalf("ExecuteInWorktree failed: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	gotPath, _ := filepath.EvalSymlinks(strings.TrimSpace(output))
	wantPath, _ := filepath.EvalSymlinks(repoPath)
	if gotPath != wantPath {
		t.Fatalf("expected git dir %s, got %s", wantPath, gotPath)
	}
}

func TestDetermineExecDir(t *testing.T) {
	rootPath, repoPath, cleanup := setupTestWorkspaceForRun(t)
	defer cleanup()

	wm, _ := workspace.NewManager(rootPath)

	nodeName := "test-node"
	if err := wm.SpawnNode(nodeName, "feature/test", "main", "test", true); err != nil {
		t.Fatalf("failed to spawn node: %v", err)
	}
	nodePath := wm.State.Nodes[nodeName].WorktreePath

	if err := os.MkdirAll(filepath.Join(nodePath, "src"), 0755); err != nil {
		t.Fatalf("failed to create node subdir: %v", err)
	}

	tests := []struct {
		name           string
		cwd            string
		targetWorktree string
		wantExecDir    string
		wantWorktree   string
		wantErr        bool
	}{
		{
			name:           "Workspace root defaults to bare repo",
			cwd:            rootPath,
			targetWorktree: "",
			wantExecDir:    repoPath,
			wantWorktree:   "",
			wantErr:        false,
		},
		{
			name:           "Inside node defaults to bare repo",
			cwd:            nodePath,
			targetWorktree: "",
			wantExecDir:    repoPath,
			wantWorktree:   "",
			wantErr:        false,
		},
		{
			name:           "Inside node subdir keeps same subdir when targeting node",
			cwd:            filepath.Join(nodePath, "src"),
			targetWorktree: nodeName,
			wantExecDir:    filepath.Join(nodePath, "src"),
			wantWorktree:   nodeName,
			wantErr:        false,
		},
		{
			name:           "Workspace root switches to node root",
			cwd:            rootPath,
			targetWorktree: nodeName,
			wantExecDir:    nodePath,
			wantWorktree:   nodeName,
			wantErr:        false,
		},
		{
			name:           "Invalid node returns error",
			cwd:            rootPath,
			targetWorktree: "invalid",
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotExecDir, gotWorktree, err := determineExecDir(wm, tt.cwd, tt.targetWorktree)
			if (err != nil) != tt.wantErr {
				t.Fatalf("determineExecDir() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			gotEval, _ := filepath.EvalSymlinks(gotExecDir)
			wantEval, _ := filepath.EvalSymlinks(tt.wantExecDir)
			if gotEval != wantEval {
				t.Fatalf("determineExecDir() execDir = %s, want %s", gotEval, wantEval)
			}
			if gotWorktree != tt.wantWorktree {
				t.Fatalf("determineExecDir() worktree = %s, want %s", gotWorktree, tt.wantWorktree)
			}
		})
	}
}
