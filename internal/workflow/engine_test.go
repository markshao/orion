package workflow

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"orion/internal/types"
	"orion/internal/workspace"

	"gopkg.in/yaml.v3"
)

// setupTestWorkspace creates a temporary workspace for testing
func setupTestWorkspace(t *testing.T) (*workspace.WorkspaceManager, string) {
	t.Helper()
	rootPath := t.TempDir()

	// Create necessary directories
	dirs := []string{
		workspace.RepoDir,
		workspace.WorkspacesDir,
		workspace.MetaDir,
		filepath.Join(workspace.MetaDir, workspace.WorkflowsDir),
		filepath.Join(workspace.MetaDir, workspace.RunsDir),
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

func TestStartRun(t *testing.T) {
	wm, rootPath := setupTestWorkspace(t)
	engine := NewEngine(wm)

	// Create a dummy workflow
	wf := types.Workflow{
		Name: "test-workflow",
		Pipeline: []types.PipelineStep{
			{ID: "step1", Agent: "test-agent"},
		},
	}
	wfData, _ := yaml.Marshal(wf)
	wfPath := filepath.Join(rootPath, workspace.MetaDir, workspace.WorkflowsDir, "test-workflow.yaml")
	if err := os.WriteFile(wfPath, wfData, 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// We expect StartRun to fail at execution step because "test-agent" config doesn't exist
	// and we don't want to actually run agent.
	// But it should create the Run object and persist it before failing.

	// Create dummy agent config to pass config loading
	agentPath := filepath.Join(rootPath, workspace.MetaDir, workspace.AgentsDir)
	os.MkdirAll(agentPath, 0755)
	agentConfig := types.Agent{
		Name:   "test-agent",
		Prompt: "test.md",
		Runtime: types.AgentRuntime{
			Provider: "test-provider",
			Model:    "test-model",
		},
	}
	agentData, _ := yaml.Marshal(agentConfig)
	os.WriteFile(filepath.Join(agentPath, "test-agent.yaml"), agentData, 0644)

	// Create dummy prompt
	promptPath := filepath.Join(rootPath, workspace.MetaDir, workspace.PromptsDir)
	os.MkdirAll(promptPath, 0755)
	os.WriteFile(filepath.Join(promptPath, "test.md"), []byte("hello"), 0644)

	// Start Run
	run, err := engine.StartRun("test-workflow", "manual", "master", "test-node")

	// Even if it fails during execution (e.g. git worktree add might fail if we are not careful with branches),
	// it should return a run object.
	// Actually StartRun executes synchronously. If execution fails, it returns the run object but with error?
	// No, StartRun returns (*Run, error). If setup fails, it returns error.
	// If pipeline execution fails, it returns run (with status Failed) and nil error.

	if err != nil {
		t.Fatalf("StartRun failed: %v", err)
	}

	if run.ID == "" {
		t.Error("Run ID is empty")
	}
	if run.Status != StatusSuccess && run.Status != StatusFailed {
		t.Errorf("Unexpected run status: %s", run.Status)
	}
	if run.TriggeredByNode != "test-node" {
		t.Errorf("Expected TriggeredByNode 'test-node', got '%s'", run.TriggeredByNode)
	}

	// Verify persistence
	runs, err := engine.ListRuns()
	if err != nil {
		t.Fatalf("ListRuns failed: %v", err)
	}
	if len(runs) != 1 {
		t.Errorf("Expected 1 run, got %d", len(runs))
	}
	if runs[0].ID != run.ID {
		t.Errorf("Run ID mismatch in ListRuns")
	}
}
