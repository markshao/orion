package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"orion/internal/workspace"
)

func TestRunSyncRefUpdatesBareRepoBranch(t *testing.T) {
	rootPath, repoPath, cleanup := setupTestWorkspaceForRun(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	nodeName := "sync-node"
	if err := wm.SpawnNode(nodeName, "feature/sync", "main", "test", false); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	node := wm.State.Nodes[nodeName]
	testFile := filepath.Join(node.WorktreePath, "sync.txt")
	if err := os.WriteFile(testFile, []byte("synced"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	if output, err := exec.Command("git", "-C", node.WorktreePath, "add", "sync.txt").CombinedOutput(); err != nil {
		t.Fatalf("git add failed: %v, output: %s", err, output)
	}
	if output, err := exec.Command("git", "-C", node.WorktreePath, "commit", "-m", "Sync commit").CombinedOutput(); err != nil {
		t.Fatalf("git commit failed: %v, output: %s", err, output)
	}

	shaBytes, err := exec.Command("git", "-C", node.WorktreePath, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatalf("rev-parse HEAD failed: %v", err)
	}
	wantSHA := strings.TrimSpace(string(shaBytes))

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := runSyncRef(node.WorktreePath, &stdout, &stderr); err != nil {
		t.Fatalf("runSyncRef failed: %v, stderr: %s", err, stderr.String())
	}

	gotSHABytes, err := exec.Command("git", "-C", repoPath, "rev-parse", "refs/heads/feature/sync").Output()
	if err != nil {
		t.Fatalf("rev-parse bare ref failed: %v", err)
	}
	gotSHA := strings.TrimSpace(string(gotSHABytes))
	if gotSHA != wantSHA {
		t.Fatalf("expected bare ref to point to %s, got %s", wantSHA, gotSHA)
	}

	if !strings.Contains(stdout.String(), "Syncing node 'sync-node' branch 'feature/sync' into repo.git") {
		t.Fatalf("missing sync banner in output: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Updated refs/heads/feature/sync -> "+wantSHA) {
		t.Fatalf("missing updated ref output: %s", stdout.String())
	}
}

func TestRunSyncRefWarnsForUncommittedChanges(t *testing.T) {
	rootPath, _, cleanup := setupTestWorkspaceForRun(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	nodeName := "dirty-node"
	if err := wm.SpawnNode(nodeName, "feature/dirty", "main", "test", false); err != nil {
		t.Fatalf("SpawnNode failed: %v", err)
	}

	node := wm.State.Nodes[nodeName]
	if err := os.WriteFile(filepath.Join(node.WorktreePath, "dirty.txt"), []byte("dirty"), 0644); err != nil {
		t.Fatalf("failed to write dirty file: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := runSyncRef(node.WorktreePath, &stdout, &stderr); err != nil {
		t.Fatalf("runSyncRef failed: %v, stderr: %s", err, stderr.String())
	}

	if !strings.Contains(stdout.String(), "uncommitted changes are not included") {
		t.Fatalf("expected dirty worktree warning, got: %s", stdout.String())
	}
}

func TestRunSyncRefRequiresNodeWorktree(t *testing.T) {
	rootPath, _, cleanup := setupTestWorkspaceForRun(t)
	defer cleanup()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runSyncRef(rootPath, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error when running outside a node worktree")
	}
	if !strings.Contains(err.Error(), "must be run inside a node worktree") {
		t.Fatalf("unexpected error: %v", err)
	}
}
