package workflow

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"orion/internal/types"
	"orion/internal/workspace"
)

// setupTestWorkflowWorkspace creates a workspace with workflow configuration for testing
func setupTestWorkflowWorkspace(t *testing.T) (string, func()) {
	t.Helper()

	// Create temp directory
	rootDir, err := os.MkdirTemp("", "workflow-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Create directory structure
	dirs := []string{
		filepath.Join(rootDir, "main_repo"),
		filepath.Join(rootDir, ".orion"),
		filepath.Join(rootDir, ".orion", "workflows"),
		filepath.Join(rootDir, ".orion", "agents"),
		filepath.Join(rootDir, ".orion", "prompts"),
		filepath.Join(rootDir, ".orion", "runs"),
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			os.RemoveAll(rootDir)
			t.Fatalf("failed to create directory %s: %v", d, err)
		}
	}

	// Create state file
	state := map[string]interface{}{
		"repo_url":  rootDir,
		"repo_path": filepath.Join(rootDir, "main_repo"),
		"nodes":     make(map[string]interface{}),
	}
	statePath := filepath.Join(rootDir, ".orion", "state.json")
	stateFile, _ := os.Create(statePath)
	json.NewEncoder(stateFile).Encode(state)
	stateFile.Close()

	// Create config file
	configContent := `version: 1
workspace: workspaces
git:
  main_branch: main
  user: test
  email: test@example.com
workflow:
  default: default
runtime:
  artifact_dir: .orion/runs
agents:
  default_provider: qwen
  providers:
    qwen:
      command: 'echo "mock"'
`
	os.WriteFile(filepath.Join(rootDir, ".orion", "config.yaml"), []byte(configContent), 0644)

	// Create default workflow
	workflowContent := `name: default
trigger:
  event: commit
pipeline:
  - id: ut
    agent: ut-agent
    branch: shadow
    suffix: ut
`
	os.WriteFile(filepath.Join(rootDir, ".orion", "workflows", "default.yaml"), []byte(workflowContent), 0644)

	// Create ut-agent config
	agentContent := `name: ut-agent
runtime:
  provider: qwen
  model: qwen-max
prompt: ut.md
`
	os.WriteFile(filepath.Join(rootDir, ".orion", "agents", "ut-agent.yaml"), []byte(agentContent), 0644)

	// Create base prompt
	promptContent := `Test prompt for {{.Branch}}
Artifact Dir: {{.ArtifactDir}}
Commit: {{.CommitID}}
`
	os.WriteFile(filepath.Join(rootDir, ".orion", "prompts", "base.md"), []byte(promptContent), 0644)
	os.WriteFile(filepath.Join(rootDir, ".orion", "prompts", "ut.md"), []byte("UT prompt"), 0644)

	// Initialize git repo in main_repo
	initGitRepo(t, filepath.Join(rootDir, "main_repo"))

	cleanup := func() {
		os.RemoveAll(rootDir)
	}

	return rootDir, cleanup
}

func initGitRepo(t *testing.T, repoPath string) {
	t.Helper()
	execCommand("git", "init", repoPath)
	execCommand("git", "-C", repoPath, "config", "user.email", "test@example.com")
	execCommand("git", "-C", repoPath, "config", "user.name", "Test User")
	execCommand("git", "-C", repoPath, "checkout", "-b", "main")
	os.WriteFile(filepath.Join(repoPath, "README.md"), []byte("# Test"), 0644)
	execCommand("git", "-C", repoPath, "add", ".")
	execCommand("git", "-C", repoPath, "commit", "-m", "Initial commit")
}

func execCommand(name string, args ...string) {
	cmd := exec.Command(name, args...)
	_ = cmd.Run()
}

func TestNewEngine(t *testing.T) {
	rootDir, cleanup := setupTestWorkflowWorkspace(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootDir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	engine := NewEngine(wm)
	if engine == nil {
		t.Error("expected non-nil engine")
	}
	if engine.wm != wm {
		t.Error("expected engine to have workspace manager reference")
	}
}

func TestEngineListRunsEmpty(t *testing.T) {
	rootDir, cleanup := setupTestWorkflowWorkspace(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootDir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	engine := NewEngine(wm)
	runs, err := engine.ListRuns()
	if err != nil {
		t.Fatalf("ListRuns failed: %v", err)
	}
	if len(runs) != 0 {
		t.Errorf("expected 0 runs, got %d", len(runs))
	}
}

func TestEngineGetRunNotFound(t *testing.T) {
	rootDir, cleanup := setupTestWorkflowWorkspace(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootDir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	engine := NewEngine(wm)
	_, err = engine.GetRun("non-existent-run")
	if err == nil {
		t.Error("expected error for non-existent run")
	}
}

func TestRunStatusSerialization(t *testing.T) {
	run := &Run{
		ID:         "run-test-123",
		Workflow:   "default",
		Trigger:    "manual",
		BaseBranch: "main",
		Status:     StatusRunning,
		StartTime:  time.Now(),
		Steps: []StepStatus{
			{
				ID:     "ut",
				Agent:  "ut-agent",
				Status: StatusPending,
			},
		},
	}

	data, err := json.Marshal(run)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var loaded Run
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if loaded.ID != run.ID {
		t.Errorf("expected ID %s, got %s", run.ID, loaded.ID)
	}
	if loaded.Workflow != run.Workflow {
		t.Errorf("expected workflow %s, got %s", run.Workflow, loaded.Workflow)
	}
	if loaded.Status != run.Status {
		t.Errorf("expected status %s, got %s", run.Status, loaded.Status)
	}
}

func TestStepStatusSerialization(t *testing.T) {
	step := StepStatus{
		ID:           "ut",
		Agent:        "ut-agent",
		Status:       StatusSuccess,
		StartTime:    time.Now(),
		EndTime:      time.Now(),
		NodeName:     "test-node",
		ShadowBranch: "orion/run-123/ut",
		Error:        "",
	}

	data, err := json.Marshal(step)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var loaded StepStatus
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if loaded.ID != step.ID {
		t.Errorf("expected ID %s, got %s", step.ID, loaded.ID)
	}
	if loaded.Agent != step.Agent {
		t.Errorf("expected agent %s, got %s", step.Agent, loaded.Agent)
	}
	if loaded.Status != step.Status {
		t.Errorf("expected status %s, got %s", step.Status, loaded.Status)
	}
}

func TestRunStatusConstants(t *testing.T) {
	tests := []struct {
		status   RunStatus
		expected string
	}{
		{StatusPending, "pending"},
		{StatusRunning, "running"},
		{StatusSuccess, "success"},
		{StatusFailed, "failed"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("expected status %q, got %q", tt.expected, tt.status)
		}
	}
}

func TestEngineListRuns(t *testing.T) {
	rootDir, cleanup := setupTestWorkflowWorkspace(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootDir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	runID := "run-20260101-test123"
	runDir := filepath.Join(rootDir, ".orion", "runs", runID)
	if err := os.MkdirAll(runDir, 0755); err != nil {
		t.Fatalf("failed to create run dir: %v", err)
	}

	mockRun := &Run{
		ID:         runID,
		Workflow:   "default",
		Trigger:    "manual",
		BaseBranch: "main",
		Status:     StatusSuccess,
		StartTime:  time.Now(),
		EndTime:    time.Now(),
		Steps:      []StepStatus{},
	}

	statusPath := filepath.Join(runDir, "status.json")
	file, _ := os.Create(statusPath)
	json.NewEncoder(file).Encode(mockRun)
	file.Close()

	engine := NewEngine(wm)
	runs, err := engine.ListRuns()
	if err != nil {
		t.Fatalf("ListRuns failed: %v", err)
	}

	if len(runs) != 1 {
		t.Errorf("expected 1 run, got %d", len(runs))
	} else if runs[0].ID != runID {
		t.Errorf("expected run ID %s, got %s", runID, runs[0].ID)
	}
}

func TestEngineListRunsMultiple(t *testing.T) {
	rootDir, cleanup := setupTestWorkflowWorkspace(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootDir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	runIDs := []string{
		"run-20260101-aaa",
		"run-20260101-bbb",
		"run-20260101-ccc",
	}

	for i, runID := range runIDs {
		runDir := filepath.Join(rootDir, ".orion", "runs", runID)
		os.MkdirAll(runDir, 0755)

		mockRun := &Run{
			ID:         runID,
			Workflow:   "default",
			Trigger:    "manual",
			BaseBranch: "main",
			Status:     StatusSuccess,
			StartTime:  time.Now().Add(time.Duration(i) * time.Hour),
			Steps:      []StepStatus{},
		}

		statusPath := filepath.Join(runDir, "status.json")
		file, _ := os.Create(statusPath)
		json.NewEncoder(file).Encode(mockRun)
		file.Close()
	}

	engine := NewEngine(wm)
	runs, err := engine.ListRuns()
	if err != nil {
		t.Fatalf("ListRuns failed: %v", err)
	}

	if len(runs) != 3 {
		t.Errorf("expected 3 runs, got %d", len(runs))
	}
}

func TestEngineListRunsDeduplication(t *testing.T) {
	rootDir, cleanup := setupTestWorkflowWorkspace(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootDir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	runID := "run-20260101-dup"
	runDir := filepath.Join(rootDir, ".orion", "runs", runID)
	os.MkdirAll(runDir, 0755)

	mockRun := &Run{
		ID:         runID,
		Workflow:   "default",
		Status:     StatusSuccess,
		StartTime:  time.Now(),
		Steps:      []StepStatus{},
	}

	statusPath := filepath.Join(runDir, "status.json")
	file, _ := os.Create(statusPath)
	json.NewEncoder(file).Encode(mockRun)
	file.Close()

	engine := NewEngine(wm)
	runs, err := engine.ListRuns()
	if err != nil {
		t.Fatalf("ListRuns failed: %v", err)
	}

	count := 0
	for _, run := range runs {
		if run.ID == runID {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 1 instance of run ID, got %d", count)
	}
}

func TestEngineListRunsSkipsInvalid(t *testing.T) {
	rootDir, cleanup := setupTestWorkflowWorkspace(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootDir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	runDir := filepath.Join(rootDir, ".orion", "runs", "run-invalid")
	os.MkdirAll(runDir, 0755)

	filePath := filepath.Join(rootDir, ".orion", "runs", "not-a-dir")
	os.WriteFile(filePath, []byte("not a directory"), 0644)

	engine := NewEngine(wm)
	runs, err := engine.ListRuns()
	if err != nil {
		t.Fatalf("ListRuns failed: %v", err)
	}

	if len(runs) != 0 {
		t.Errorf("expected 0 valid runs, got %d", len(runs))
	}
}

func TestSaveRunStatus(t *testing.T) {
	rootDir, cleanup := setupTestWorkflowWorkspace(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootDir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	engine := NewEngine(wm)

	run := &Run{
		ID:         "run-save-test",
		Workflow:   "default",
		Trigger:    "manual",
		BaseBranch: "main",
		Status:     StatusRunning,
		StartTime:  time.Now(),
		Steps:      []StepStatus{},
	}

	runDir := filepath.Join(rootDir, ".orion", "runs", run.ID)
	os.MkdirAll(runDir, 0755)

	err = engine.saveRunStatus(run)
	if err != nil {
		t.Fatalf("saveRunStatus failed: %v", err)
	}

	statusPath := filepath.Join(runDir, "status.json")
	if _, err := os.Stat(statusPath); os.IsNotExist(err) {
		t.Error("expected status.json to be created")
	}

	data, err := os.ReadFile(statusPath)
	if err != nil {
		t.Fatalf("failed to read status file: %v", err)
	}

	var loaded Run
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("failed to unmarshal status: %v", err)
	}

	if loaded.ID != run.ID {
		t.Errorf("expected ID %s, got %s", run.ID, loaded.ID)
	}
	if loaded.Status != run.Status {
		t.Errorf("expected status %s, got %s", run.Status, loaded.Status)
	}
}

func TestResolveBaseBranchNoDeps(t *testing.T) {
	rootDir, cleanup := setupTestWorkflowWorkspace(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootDir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	engine := NewEngine(wm)

	run := &Run{
		ID:         "run-test",
		BaseBranch: "main",
		Steps: []StepStatus{
			{ID: "ut", Agent: "ut-agent"},
		},
	}

	stepDef := &types.PipelineStep{
		ID:        "ut",
		Agent:     "ut-agent",
		DependsOn: []string{},
	}

	baseBranch, err := engine.resolveBaseBranch(run, stepDef)
	if err != nil {
		t.Fatalf("resolveBaseBranch failed: %v", err)
	}

	if baseBranch != "main" {
		t.Errorf("expected base branch 'main', got '%s'", baseBranch)
	}
}

func TestRenderPrompt(t *testing.T) {
	rootDir, cleanup := setupTestWorkflowWorkspace(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootDir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	engine := NewEngine(wm)

	tmplContent := `Hello {{.Branch}}
Commit: {{.CommitID}}
Artifacts: {{.ArtifactDir}}
`

	data := map[string]string{
		"Branch":      "feature/test",
		"CommitID":    "abc123",
		"ArtifactDir": "/tmp/artifacts",
	}

	result, err := engine.renderPrompt(tmplContent, data)
	if err != nil {
		t.Fatalf("renderPrompt failed: %v", err)
	}

	if !strings.Contains(result, "Hello feature/test") {
		t.Error("expected rendered branch name")
	}
	if !strings.Contains(result, "Commit: abc123") {
		t.Error("expected rendered commit ID")
	}
	if !strings.Contains(result, "/tmp/artifacts") {
		t.Error("expected rendered artifact dir")
	}
}

func TestRenderPromptWithInvalidTemplate(t *testing.T) {
	rootDir, cleanup := setupTestWorkflowWorkspace(t)
	defer cleanup()

	wm, err := workspace.NewManager(rootDir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	engine := NewEngine(wm)

	tmplContent := `Hello {{.Branch`

	_, err = engine.renderPrompt(tmplContent, map[string]string{})
	if err == nil {
		t.Error("expected error for invalid template")
	}
}
