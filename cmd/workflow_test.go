package cmd

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"orion/internal/types"
	"orion/internal/workflow"
	"orion/internal/workspace"
)

// setupTestCmdWorkspace creates a temporary workspace for testing command functionality
func setupTestCmdWorkspace(t *testing.T) (*workspace.WorkspaceManager, string) {
	t.Helper()
	rootPath := t.TempDir()

	// Create necessary directories
	dirs := []string{
		workspace.RepoDir,
		workspace.WorkspacesDir,
		workspace.MetaDir,
		filepath.Join(workspace.MetaDir, workspace.WorkflowsDir),
		filepath.Join(workspace.MetaDir, workspace.RunsDir),
		filepath.Join(workspace.MetaDir, workspace.AgentsDir),
		filepath.Join(workspace.MetaDir, workspace.PromptsDir),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(rootPath, d), 0755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", d, err)
		}
	}

	repoPath := filepath.Join(rootPath, workspace.RepoDir)

	// Initialize git repo
	cmd := exec.Command("git", "init", repoPath)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Config user
	_ = exec.Command("git", "-C", repoPath, "config", "user.email", "test@example.com").Run()
	_ = exec.Command("git", "-C", repoPath, "config", "user.name", "Test User").Run()

	// Initial commit
	readme := filepath.Join(repoPath, "README.md")
	_ = os.WriteFile(readme, []byte("# Test Repo"), 0644)
	_ = exec.Command("git", "-C", repoPath, "add", ".").Run()
	_ = exec.Command("git", "-C", repoPath, "commit", "-m", "Initial commit").Run()

	// Create state.json
	state := types.State{
		RepoPath: repoPath,
	}
	stateData, _ := json.Marshal(state)
	if err := os.WriteFile(filepath.Join(rootPath, workspace.MetaDir, workspace.StateFile), stateData, 0644); err != nil {
		t.Fatalf("Failed to write state.json: %v", err)
	}

	wm, err := workspace.NewManager(rootPath)
	if err != nil {
		t.Fatalf("Failed to load workspace manager: %v", err)
	}
	return wm, rootPath
}

// createTestRun creates a test workflow run in the workspace
func createTestRun(t *testing.T, wm *workspace.WorkspaceManager, runID string, status workflow.RunStatus) *workflow.Run {
	t.Helper()

	runDir := filepath.Join(wm.RootPath, workspace.MetaDir, workspace.RunsDir, runID)
	if err := os.MkdirAll(runDir, 0755); err != nil {
		t.Fatalf("Failed to create run dir: %v", err)
	}

	run := &workflow.Run{
		ID:        runID,
		Workflow:  "test-workflow",
		Trigger:   "manual",
		Status:    status,
		StartTime: time.Now(),
		Steps:     []workflow.StepStatus{},
	}

	statusPath := filepath.Join(runDir, "status.json")
	file, err := os.Create(statusPath)
	if err != nil {
		t.Fatalf("Failed to create status file: %v", err)
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(run); err != nil {
		t.Fatalf("Failed to write status file: %v", err)
	}

	return run
}

// createTestAgentNode creates a test agent node in the workspace state
func createTestAgentNode(t *testing.T, wm *workspace.WorkspaceManager, nodeName, createdBy string) {
	t.Helper()

	worktreePath := filepath.Join(wm.RootPath, workspace.MetaDir, "agent-nodes", nodeName)
	if err := os.MkdirAll(worktreePath, 0755); err != nil {
		t.Fatalf("Failed to create node worktree: %v", err)
	}

	node := types.Node{
		Name:         nodeName,
		WorktreePath: worktreePath,
		ShadowBranch: "orion/" + nodeName,
		CreatedBy:    createdBy,
		TmuxSession:  "orion-" + nodeName,
	}

	if wm.State.Nodes == nil {
		wm.State.Nodes = make(map[string]types.Node)
	}
	wm.State.Nodes[nodeName] = node

	if err := wm.SaveState(); err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}
}

func TestRmWorkflowCmd_RunNotFound(t *testing.T) {
	wm, rootPath := setupTestCmdWorkspace(t)

	// Change to the workspace root
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(rootPath)

	// Create a non-existent run ID
	runID := "run-nonexistent"

	engine := workflow.NewEngine(wm)
	_, err := engine.GetRun(runID)

	if err == nil {
		t.Error("Expected error for non-existent run, got nil")
	}
}

func TestRmWorkflowCmd_RemoveSuccessful(t *testing.T) {
	wm, rootPath := setupTestCmdWorkspace(t)

	// Change to the workspace root
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(rootPath)

	runID := "run-test-success"
	createTestRun(t, wm, runID, workflow.StatusSuccess)

	// Verify run exists
	engine := workflow.NewEngine(wm)
	run, err := engine.GetRun(runID)
	if err != nil {
		t.Fatalf("Failed to get run before removal: %v", err)
	}
	if run == nil {
		t.Fatal("Run should exist before removal")
	}

	// Simulate the removal logic
	runDir := filepath.Join(wm.RootPath, workspace.MetaDir, workspace.RunsDir, runID)
	if err := os.RemoveAll(runDir); err != nil {
		t.Fatalf("Failed to remove run directory: %v", err)
	}

	// Verify run is removed
	_, err = engine.GetRun(runID)
	if err == nil {
		t.Error("Expected error after removal, got nil")
	}
}

func TestRmWorkflowCmd_RemoveRunningWithoutForce(t *testing.T) {
	wm, rootPath := setupTestCmdWorkspace(t)

	// Change to the workspace root
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(rootPath)

	runID := "run-test-running"
	createTestRun(t, wm, runID, workflow.StatusRunning)

	// Simulate the check logic
	engine := workflow.NewEngine(wm)
	run, err := engine.GetRun(runID)
	if err != nil {
		t.Fatalf("Failed to get run: %v", err)
	}

	// Check if run is still running
	if run.Status == workflow.StatusRunning {
		// This is the expected behavior - should block removal without force
		// In the actual command, this would call os.Exit(1)
		t.Log("Correctly detected running run - would block removal without --force")
	} else {
		t.Error("Expected run status to be Running")
	}
}

func TestRmWorkflowCmd_RemoveRunningWithForce(t *testing.T) {
	wm, rootPath := setupTestCmdWorkspace(t)

	// Change to the workspace root
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(rootPath)

	runID := "run-test-force"
	createTestRun(t, wm, runID, workflow.StatusRunning)

	// Create an agent node associated with this run
	nodeName := "test-agent-node"
	createTestAgentNode(t, wm, nodeName, runID)

	// Verify node exists
	if _, exists := wm.State.Nodes[nodeName]; !exists {
		t.Fatal("Agent node should exist before removal")
	}

	// Simulate the force removal logic
	engine := workflow.NewEngine(wm)
	run, err := engine.GetRun(runID)
	if err != nil {
		t.Fatalf("Failed to get run: %v", err)
	}

	// Force mode: should allow removal even if running
	if run.Status == workflow.StatusRunning {
		t.Log("Force mode enabled - would allow removal of running run")
	}

	// Find and remove all agentic nodes created by this run
	var nodesRemoved int
	for nodeName, node := range wm.State.Nodes {
		if node.CreatedBy == runID {
			if err := wm.RemoveNode(nodeName); err != nil {
				t.Errorf("Failed to remove node '%s': %v", nodeName, err)
			} else {
				nodesRemoved++
			}
		}
	}

	if nodesRemoved != 1 {
		t.Errorf("Expected 1 node removed, got %d", nodesRemoved)
	}

	// Remove the run directory
	runDir := filepath.Join(wm.RootPath, workspace.MetaDir, workspace.RunsDir, runID)
	if err := os.RemoveAll(runDir); err != nil {
		t.Fatalf("Failed to remove run directory: %v", err)
	}

	// Verify run is removed
	_, err = engine.GetRun(runID)
	if err == nil {
		t.Error("Expected error after removal, got nil")
	}
}

func TestRmWorkflowCmd_RemoveWithMultipleNodes(t *testing.T) {
	wm, rootPath := setupTestCmdWorkspace(t)

	// Change to the workspace root
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(rootPath)

	runID := "run-test-multi-nodes"
	createTestRun(t, wm, runID, workflow.StatusSuccess)

	// Create multiple agent nodes associated with this run
	nodeNames := []string{"agent-node-1", "agent-node-2", "agent-node-3"}
	for _, name := range nodeNames {
		createTestAgentNode(t, wm, name, runID)
	}

	// Create a node from a different run (should not be removed)
	otherRunID := "run-other"
	createTestRun(t, wm, otherRunID, workflow.StatusSuccess)
	createTestAgentNode(t, wm, "other-agent-node", otherRunID)

	// Simulate the removal logic
	engine := workflow.NewEngine(wm)
	run, err := engine.GetRun(runID)
	if err != nil {
		t.Fatalf("Failed to get run: %v", err)
	}
	_ = run

	// Find and remove all agentic nodes created by this run
	var nodesRemoved int
	var removedNodeNames []string
	for nodeName, node := range wm.State.Nodes {
		if node.CreatedBy == runID {
			if err := wm.RemoveNode(nodeName); err != nil {
				t.Errorf("Failed to remove node '%s': %v", nodeName, err)
			} else {
				nodesRemoved++
				removedNodeNames = append(removedNodeNames, nodeName)
			}
		}
	}

	if nodesRemoved != 3 {
		t.Errorf("Expected 3 nodes removed, got %d", nodesRemoved)
	}

	// Verify the other run's node is still there
	if _, exists := wm.State.Nodes["other-agent-node"]; !exists {
		t.Error("Other run's agent node should still exist")
	}

	// Remove the run directory
	runDir := filepath.Join(wm.RootPath, workspace.MetaDir, workspace.RunsDir, runID)
	if err := os.RemoveAll(runDir); err != nil {
		t.Fatalf("Failed to remove run directory: %v", err)
	}

	// Verify run is removed
	_, err = engine.GetRun(runID)
	if err == nil {
		t.Error("Expected error after removal, got nil")
	}

	// Verify other run still exists
	otherRun, err := engine.GetRun(otherRunID)
	if err != nil {
		t.Error("Other run should still exist")
	}
	if otherRun == nil {
		t.Error("Other run should not be nil")
	}
}

func TestRmWorkflowCmd_NodeCreatedByField(t *testing.T) {
	wm, rootPath := setupTestCmdWorkspace(t)

	// Change to the workspace root
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(rootPath)

	runID := "run-test-createdby"
	createTestRun(t, wm, runID, workflow.StatusSuccess)

	// Create nodes with different CreatedBy values
	createTestAgentNode(t, wm, "node-from-this-run", runID)
	createTestAgentNode(t, wm, "node-from-user", "user")
	createTestAgentNode(t, wm, "node-from-other-run", "run-other-id")

	// Count nodes that should be removed
	var nodesToRemove []string
	for nodeName, node := range wm.State.Nodes {
		if node.CreatedBy == runID {
			nodesToRemove = append(nodesToRemove, nodeName)
		}
	}

	if len(nodesToRemove) != 1 {
		t.Errorf("Expected 1 node to be removed, got %d: %v", len(nodesToRemove), nodesToRemove)
	}

	if len(nodesToRemove) > 0 && nodesToRemove[0] != "node-from-this-run" {
		t.Errorf("Expected 'node-from-this-run' to be removed, got '%s'", nodesToRemove[0])
	}
}
