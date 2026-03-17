package vscode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateWorkspaceFile(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "vscode-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	repoDir := "main_repo"
	nodesDir := "workspaces"
	nodes := []string{"node1", "node2"}

	// Create the directories
	os.MkdirAll(filepath.Join(tmpDir, repoDir), 0755)
	os.MkdirAll(filepath.Join(tmpDir, nodesDir, "node1"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, nodesDir, "node2"), 0755)

	// Test UpdateWorkspaceFile
	err = UpdateWorkspaceFile(tmpDir, repoDir, nodesDir, nodes)
	if err != nil {
		t.Fatalf("UpdateWorkspaceFile failed: %v", err)
	}

	// Verify workspace file was created
	// The workspace file name is derived from tmpDir's base name
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("failed to read tmp dir: %v", err)
	}

	var workspaceFile string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".code-workspace") {
			workspaceFile = filepath.Join(tmpDir, entry.Name())
			break
		}
	}

	if workspaceFile == "" {
		t.Error("expected workspace file to be created")
		return
	}

	// Read and verify content
	data, err := os.ReadFile(workspaceFile)
	if err != nil {
		t.Fatalf("failed to read workspace file: %v", err)
	}

	var workspace WorkspaceFile
	if err := json.Unmarshal(data, &workspace); err != nil {
		t.Fatalf("failed to unmarshal workspace file: %v", err)
	}

	// Verify folders
	expectedFolders := 3 // repo + 2 nodes
	if len(workspace.Folders) != expectedFolders {
		t.Errorf("expected %d folders, got %d", expectedFolders, len(workspace.Folders))
	}

	// Verify repo folder is first
	if workspace.Folders[0].Path != repoDir {
		t.Errorf("expected first folder to be %s, got %s", repoDir, workspace.Folders[0].Path)
	}

	// Verify node folders
	nodePaths := make(map[string]bool)
	for i := 1; i < len(workspace.Folders); i++ {
		nodePaths[workspace.Folders[i].Path] = true
	}

	for _, node := range nodes {
		expectedPath := filepath.Join(nodesDir, node)
		if !nodePaths[expectedPath] {
			t.Errorf("expected node folder %s not found", expectedPath)
		}
	}
}

func TestUpdateWorkspaceFileWithEmptyNodes(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "vscode-test-empty-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	repoDir := "main_repo"
	nodesDir := "workspaces"
	nodes := []string{}

	os.MkdirAll(filepath.Join(tmpDir, repoDir), 0755)

	err = UpdateWorkspaceFile(tmpDir, repoDir, nodesDir, nodes)
	if err != nil {
		t.Fatalf("UpdateWorkspaceFile failed: %v", err)
	}

	// Find workspace file
	entries, _ := os.ReadDir(tmpDir)
	var workspaceFile string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".code-workspace") {
			workspaceFile = filepath.Join(tmpDir, entry.Name())
			break
		}
	}

	data, err := os.ReadFile(workspaceFile)
	if err != nil {
		t.Fatalf("failed to read workspace file: %v", err)
	}

	var workspace WorkspaceFile
	if err := json.Unmarshal(data, &workspace); err != nil {
		t.Fatalf("failed to unmarshal workspace file: %v", err)
	}

	if len(workspace.Folders) != 1 {
		t.Errorf("expected 1 folder (repo only), got %d", len(workspace.Folders))
	}
}

func TestUpdateWorkspaceFileWithSwarmSuffix(t *testing.T) {
	// Create a temp directory manually to control the name
	tmpBase := filepath.Join(os.TempDir(), "orion-vscode-suffix-test")
	tmpDir := tmpBase + "_swarm"

	// Clean up any existing
	os.RemoveAll(tmpDir)

	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	repoDir := "main_repo"
	nodesDir := "workspaces"
	nodes := []string{}

	os.MkdirAll(filepath.Join(tmpDir, repoDir), 0755)

	err := UpdateWorkspaceFile(tmpDir, repoDir, nodesDir, nodes)
	if err != nil {
		t.Fatalf("UpdateWorkspaceFile failed: %v", err)
	}

	// Find workspace file and check name
	entries, _ := os.ReadDir(tmpDir)
	var workspaceFileName string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".code-workspace") {
			workspaceFileName = entry.Name()
			break
		}
	}

	if workspaceFileName == "" {
		t.Error("expected workspace file to be created")
	} else if workspaceFileName != "orion-vscode-suffix-test.code-workspace" {
		// The _swarm suffix should be removed
		t.Errorf("expected workspace file 'orion-vscode-suffix-test.code-workspace', got '%s'", workspaceFileName)
	}
}

func TestUpdateWorkspaceFileCreatesDirectories(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "vscode-test-mkdir-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	repoDir := "main_repo"
	nodesDir := "workspaces"
	nodes := []string{"new-node"}

	// Only create root dir, not subdirs
	err = UpdateWorkspaceFile(tmpDir, repoDir, nodesDir, nodes)
	if err != nil {
		t.Fatalf("UpdateWorkspaceFile failed: %v", err)
	}

	// Find workspace file
	entries, _ := os.ReadDir(tmpDir)
	var workspaceFile string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".code-workspace") {
			workspaceFile = filepath.Join(tmpDir, entry.Name())
			break
		}
	}

	// Workspace file should be created even if node dirs don't exist
	if workspaceFile == "" {
		t.Error("expected workspace file to be created")
	}
}

func TestWorkspaceFileJSONFormat(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "vscode-test-json-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	repoDir := "main_repo"
	nodesDir := "workspaces"
	nodes := []string{"node1"}

	os.MkdirAll(filepath.Join(tmpDir, repoDir), 0755)
	os.MkdirAll(filepath.Join(tmpDir, nodesDir, "node1"), 0755)

	err = UpdateWorkspaceFile(tmpDir, repoDir, nodesDir, nodes)
	if err != nil {
		t.Fatalf("UpdateWorkspaceFile failed: %v", err)
	}

	// Find workspace file
	entries, _ := os.ReadDir(tmpDir)
	var workspaceFile string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".code-workspace") {
			workspaceFile = filepath.Join(tmpDir, entry.Name())
			break
		}
	}

	data, err := os.ReadFile(workspaceFile)
	if err != nil {
		t.Fatalf("failed to read workspace file: %v", err)
	}

	// Verify it's valid JSON with proper formatting (indented)
	lines := strings.Split(string(data), "\n")
	if len(lines) < 5 {
		t.Error("expected formatted JSON with multiple lines")
	}

	// Check for indentation (2 spaces)
	hasIndent := false
	for _, line := range lines {
		if strings.HasPrefix(line, "  ") {
			hasIndent = true
			break
		}
	}
	if !hasIndent {
		t.Error("expected indented JSON output")
	}
}

func TestUpdateWorkspaceFilePreservesSettings(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "vscode-test-settings-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	repoDir := "main_repo"
	nodesDir := "workspaces"
	nodes := []string{}

	os.MkdirAll(filepath.Join(tmpDir, repoDir), 0755)

	err = UpdateWorkspaceFile(tmpDir, repoDir, nodesDir, nodes)
	if err != nil {
		t.Fatalf("UpdateWorkspaceFile failed: %v", err)
	}

	// Find workspace file
	entries, _ := os.ReadDir(tmpDir)
	var workspaceFile string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".code-workspace") {
			workspaceFile = filepath.Join(tmpDir, entry.Name())
			break
		}
	}

	data, err := os.ReadFile(workspaceFile)
	if err != nil {
		t.Fatalf("failed to read workspace file: %v", err)
	}

	var workspace WorkspaceFile
	if err := json.Unmarshal(data, &workspace); err != nil {
		t.Fatalf("failed to unmarshal workspace file: %v", err)
	}

	// Settings should be an empty map, not nil
	if workspace.Settings == nil {
		t.Error("expected Settings to be an empty map, not nil")
	}
}
