package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"devswarm/internal/types"
	"devswarm/internal/workspace"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

type Engine struct {
	wm *workspace.WorkspaceManager
}

func NewEngine(wm *workspace.WorkspaceManager) *Engine {
	return &Engine{wm: wm}
}

// StartRun initializes a new run and starts executing it.
// Currently synchronous.
func (e *Engine) StartRun(workflowName, trigger string) (*Run, error) {
	// 1. Load workflow definition
	wfPath := filepath.Join(e.wm.RootPath, workspace.MetaDir, workspace.WorkflowsDir, workflowName+".yaml")
	wfData, err := os.ReadFile(wfPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow %s: %w", workflowName, err)
	}

	var wf types.Workflow
	if err := yaml.Unmarshal(wfData, &wf); err != nil {
		return nil, fmt.Errorf("failed to parse workflow %s: %w", workflowName, err)
	}

	// 2. Create Run structure
	runID := fmt.Sprintf("run-%s-%s", time.Now().Format("20060102"), uuid.New().String()[:8])
	runDir := filepath.Join(e.wm.RootPath, workspace.MetaDir, workspace.RunsDir, runID)
	if err := os.MkdirAll(runDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create run directory: %w", err)
	}

	run := &Run{
		ID:        runID,
		Workflow:  workflowName,
		Trigger:   trigger,
		Status:    StatusRunning, // Mark as running immediately
		StartTime: time.Now(),
		Steps:     make([]StepStatus, len(wf.Pipeline)),
	}

	for i, step := range wf.Pipeline {
		run.Steps[i] = StepStatus{
			ID:     step.ID,
			Agent:  step.Agent,
			Status: StatusPending,
		}
	}

	// 3. Persist initial status
	if err := e.saveRunStatus(run); err != nil {
		return nil, err
	}

	// 4. Execute pipeline (Synchronous for now to ensure completion in CLI)
	// In a real system, this might be handed off to a worker pool or daemon.
	e.executePipeline(run, &wf)

	return run, nil
}

func (e *Engine) executePipeline(run *Run, wf *types.Workflow) {
	// Simple sequential execution
	for i, stepDef := range wf.Pipeline {
		step := &run.Steps[i]
		step.StartTime = time.Now()
		step.Status = StatusRunning
		step.NodeName = fmt.Sprintf("%s-%s-%s", run.ID, step.ID, stepDef.Suffix)
		_ = e.saveRunStatus(run)

		// Create Node and Execute Agent
		err := e.executeStep(run, step, &stepDef)
		
		step.EndTime = time.Now()
		if err != nil {
			step.Status = StatusFailed
			step.Error = err.Error()
			run.Status = StatusFailed
			run.EndTime = time.Now()
			_ = e.saveRunStatus(run)
			return // Stop execution on failure
		}
		step.Status = StatusSuccess
		_ = e.saveRunStatus(run)
	}

	run.Status = StatusSuccess
	run.EndTime = time.Now()
	_ = e.saveRunStatus(run)
}

func (e *Engine) executeStep(run *Run, step *StepStatus, stepDef *types.PipelineStep) error {
	// TODO: Implement actual node spawning and agent execution
	// 1. Spawn Node (Shadow Branch)
	// 2. Inject Env
	// 3. Run Agent Command
	
	// Simulation for now
	time.Sleep(2 * time.Second)
	return nil
}

func (e *Engine) saveRunStatus(run *Run) error {
	path := filepath.Join(e.wm.RootPath, workspace.MetaDir, workspace.RunsDir, run.ID, "status.json")
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewEncoder(file).Encode(run)
}

func (e *Engine) ListRuns() ([]Run, error) {
	runsDir := filepath.Join(e.wm.RootPath, workspace.MetaDir, workspace.RunsDir)
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Run{}, nil
		}
		return nil, err
	}

	var runs []Run
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		
		statusPath := filepath.Join(runsDir, entry.Name(), "status.json")
		data, err := os.ReadFile(statusPath)
		if err != nil {
			continue // Skip corrupted/incomplete runs
		}

		var run Run
		if err := json.Unmarshal(data, &run); err == nil {
			runs = append(runs, run)
		}
	}

	// Sort by start time descending
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].StartTime.After(runs[j].StartTime)
	})

	return runs, nil
}

func (e *Engine) GetRun(runID string) (*Run, error) {
	path := filepath.Join(e.wm.RootPath, workspace.MetaDir, workspace.RunsDir, runID, "status.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var run Run
	if err := json.Unmarshal(data, &run); err != nil {
		return nil, err
	}
	return &run, nil
}
