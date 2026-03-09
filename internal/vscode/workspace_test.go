package vscode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestUpdateWorkspaceFile verifies that the generated .code-workspace file
// contains the repo folder and all node folders with the expected paths.
func TestUpdateWorkspaceFile(t *testing.T) {
	rootDir, err := os.MkdirTemp("", "orion-vscode-ws-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(rootDir)

	repoDir := "main_repo"
	nodesDir := "workspaces"
	nodes := []string{"node-a", "node-b"}

	if err := UpdateWorkspaceFile(rootDir, repoDir, nodesDir, nodes); err != nil {
		t.Fatalf("UpdateWorkspaceFile returned error: %v", err)
	}

	// Workspace file name is derived from base name of root dir.
	projectName := filepath.Base(rootDir)
	workspacePath := filepath.Join(rootDir, projectName+".code-workspace")

	data, err := os.ReadFile(workspacePath)
	if err != nil {
		t.Fatalf("failed to read workspace file: %v", err)
	}

	var ws WorkspaceFile
	if err := json.Unmarshal(data, &ws); err != nil {
		t.Fatalf("failed to unmarshal workspace JSON: %v", err)
	}

	// We expect 1 repo folder + len(nodes) node folders
	expectedCount := 1 + len(nodes)
	if len(ws.Folders) != expectedCount {
		t.Fatalf("unexpected folders count: got %d, want %d", len(ws.Folders), expectedCount)
	}

	// First folder should be the repo folder
	if ws.Folders[0].Path != repoDir {
		t.Errorf("repo folder path = %q, want %q", ws.Folders[0].Path, repoDir)
	}

	// Subsequent folders should be nodesDir/nodeName
	for i, node := range nodes {
		idx := i + 1
		wantPath := filepath.Join(nodesDir, node)
		if ws.Folders[idx].Path != wantPath {
			t.Errorf("folder[%d].Path = %q, want %q", idx, ws.Folders[idx].Path, wantPath)
		}
	}
}

