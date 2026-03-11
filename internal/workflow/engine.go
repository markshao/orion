package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"

	agent_pkg "orion/internal/agent"
	"orion/internal/git"
	"orion/internal/tmux"
	"orion/internal/types"
	"orion/internal/workspace"

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
func (e *Engine) StartRun(workflowName, trigger, baseBranch, triggeredByNode string) (*Run, error) {
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

	// 2. Use provided baseBranch or default to current branch of main repo if empty
	if baseBranch == "" {
		var err error
		baseBranch, err = git.GetCurrentBranch(e.wm.State.RepoPath)
		if err != nil {
			return nil, fmt.Errorf("failed to determine base branch: %w", err)
		}
	}

	// Capture trigger data (e.g. commit hash) if triggered by commit
	triggerData := ""
	if trigger == "commit" {
		// Get latest commit hash from main repo
		hash, err := git.GetLatestCommitHash(e.wm.State.RepoPath)
		if err == nil {
			triggerData = hash[:7] // Short hash
		}
	}

	// 3. Create Run structure
	runID := fmt.Sprintf("run-%s-%s", time.Now().Format("20060102"), uuid.New().String()[:8])
	runDir := filepath.Join(e.wm.RootPath, workspace.MetaDir, workspace.RunsDir, runID)
	if err := os.MkdirAll(runDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create run directory: %w", err)
	}

	run := &Run{
		ID:              runID,
		Workflow:        workflowName,
		Trigger:         trigger,
		TriggerData:     triggerData,
		BaseBranch:      baseBranch,
		TriggeredByNode: triggeredByNode,
		Status:          StatusRunning, // Mark as running immediately
		StartTime:       time.Now(),
		Steps:           make([]StepStatus, len(wf.Pipeline)),
	}

	for i, step := range wf.Pipeline {
		run.Steps[i] = StepStatus{
			ID:     step.ID,
			Agent:  step.Agent,
			Status: StatusPending,
		}
	}

	// 4. Persist initial status
	if err := e.saveRunStatus(run); err != nil {
		return nil, err
	}

	// 5. Execute pipeline (Synchronous)
	// We run this synchronously to ensure the process stays alive while agents are running.
	// Users can use 'nohup' or '&' to run in background if needed.
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
	// 1. Determine Base Branch (Dependency Chaining)
	baseBranch, err := e.resolveBaseBranch(run, stepDef)
	if err != nil {
		return fmt.Errorf("failed to resolve base branch: %w", err)
	}

	// 2. Define Shadow Branch
	// Naming: orion/<run-id>/<step-id>
	shadowBranch := fmt.Sprintf("orion/%s/%s", run.ID, step.ID)
	step.ShadowBranch = shadowBranch

	// 3. Spawn Node (Worktree + Shadow Branch + Tmux)
	node, err := e.wm.CreateAgentNode(step.NodeName, shadowBranch, baseBranch, run.ID)
	if err != nil {
		return fmt.Errorf("failed to spawn node: %w", err)
	}

	// 4. Setup Git Identity for the Node
	// Strategy:
	// 1. Try to get identity from config.yaml
	// 2. If missing, try to get from main repo (fallback)
	// 3. Apply to node worktree

	config, err := e.wm.GetConfig()
	userName := ""
	userEmail := ""

	if err == nil {
		userName = config.Git.User
		userEmail = config.Git.Email
	}

	// Fallback to main repo config if missing
	if userName == "" {
		userName, _ = git.GetConfig(e.wm.State.RepoPath, "user.name")
	}
	if userEmail == "" {
		userEmail, _ = git.GetConfig(e.wm.State.RepoPath, "user.email")
	}

	// Apply identity
	if userName == "" {
		userName = "ds_agent"
	}
	if userEmail == "" {
		userEmail = "ds_agent@orion.local"
	}

	_ = git.SetConfig(node.WorktreePath, "user.name", userName)
	_ = git.SetConfig(node.WorktreePath, "user.email", userEmail)

	// 5. Load Agent Configuration
	agentPath := filepath.Join(e.wm.RootPath, workspace.MetaDir, workspace.AgentsDir, stepDef.Agent+".yaml")
	agentData, err := os.ReadFile(agentPath)
	if err != nil {
		return fmt.Errorf("failed to load agent config %s: %w", stepDef.Agent, err)
	}
	var agent types.Agent
	if err := yaml.Unmarshal(agentData, &agent); err != nil {
		return fmt.Errorf("failed to parse agent config: %w", err)
	}

	// 6. Load and Render Prompt
	// a. Load Base Prompt
	basePromptPath := filepath.Join(e.wm.RootPath, workspace.MetaDir, workspace.PromptsDir, "base.md")
	basePromptData, err := os.ReadFile(basePromptPath)
	if err != nil {
		// Fallback to simple format if base.md missing (e.g. old workspace)
		basePromptData = []byte(`{{.UserPrompt}}

Context:
- Branch: {{.Branch}}
- Commit: {{.CommitID}}
`)
	}

	// b. Load User Prompt (Agent specific)
	var userPromptContent []byte
	// Check if prompt is a file path relative to prompts directory
	promptPath := filepath.Join(e.wm.RootPath, workspace.MetaDir, workspace.PromptsDir, agent.Prompt)
	fileInfo, err := os.Stat(promptPath)
	if err == nil && !fileInfo.IsDir() {
		// It's a file, read it
		userPromptContent, err = os.ReadFile(promptPath)
		if err != nil {
			return fmt.Errorf("failed to load prompt file %s: %w", agent.Prompt, err)
		}
	} else {
		// It's not a file, treat agent.Prompt as the content directly
		userPromptContent = []byte(agent.Prompt)
	}

	// c. Create Artifact Directory
	artifactDir := filepath.Join(e.wm.RootPath, workspace.MetaDir, workspace.RunsDir, run.ID, "artifacts", step.ID)
	if err := os.MkdirAll(artifactDir, 0755); err != nil {
		return fmt.Errorf("failed to create artifact directory: %w", err)
	}

	// Get absolute path for artifact dir to ensure agent can write to it regardless of CWD
	absArtifactDir, err := filepath.Abs(artifactDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for artifact dir: %w", err)
	}

	// Get latest commit hash
	commitID, err := git.GetLatestCommitHash(node.WorktreePath)
	if err != nil {
		commitID = "unknown"
	}

	// d. Render Prompt
	// First, render the UserPrompt itself if it has variables (like {{.ArtifactDir}})
	// We construct the data map
	templateData := map[string]string{
		"Branch":      shadowBranch,
		"CommitID":    commitID,
		"ArtifactDir": absArtifactDir,
	}

	renderedUserPrompt, err := e.renderPrompt(string(userPromptContent), templateData)
	if err != nil {
		// If fails (maybe user prompt has no template), fallback to raw
		renderedUserPrompt = string(userPromptContent)
	}

	// Add the rendered user prompt to data
	templateData["UserPrompt"] = renderedUserPrompt

	// We render the Base Prompt, which includes the User Prompt
	renderedPrompt, err := e.renderPrompt(string(basePromptData), templateData)
	if err != nil {
		return fmt.Errorf("failed to render prompt: %w", err)
	}

	// Write prompt to file in worktree for agent to consume
	promptFile := filepath.Join(node.WorktreePath, "agent_prompt.md")
	if err := os.WriteFile(promptFile, []byte(renderedPrompt), 0644); err != nil {
		return fmt.Errorf("failed to write prompt file: %w", err)
	}

	// 7. Prepare Environment Variables
	// Use 'agent' (types.Agent) loaded earlier
	// Note: 'agentDef' is the PipelineStep, 'agent' is the loaded Agent config.
	var resolvedEnv []string
	if len(agent.Env) > 0 {
		resolvedEnv = make([]string, 0, len(agent.Env)+1)
		for _, envName := range agent.Env {
			if val, ok := os.LookupEnv(envName); ok {
				resolvedEnv = append(resolvedEnv, fmt.Sprintf("%s=%s", envName, val))
			}
		}
	} else {
		resolvedEnv = make([]string, 0, 1)
	}
	// Inject Artifact Dir
	resolvedEnv = append(resolvedEnv, fmt.Sprintf("ORION_ARTIFACT_DIR=%s", absArtifactDir))

	// 8. Execute Agent
	// Check if provider has a custom command template in config.yaml
	config, err = e.wm.GetConfig()
	var commandTemplate string
	if err == nil {
		if provider, ok := config.Agents.Providers[agent.Runtime.Provider]; ok {
			commandTemplate = provider.Command
		}
	}

	// Allow override from agent definition
	if agent.Runtime.Command != "" {
		commandTemplate = agent.Runtime.Command
	}

	if commandTemplate != "" {
		// Custom Command Execution
		// Replace {{.PromptFile}} with the absolute path to the prompt file
		cmdStr := strings.ReplaceAll(commandTemplate, "{{.PromptFile}}", promptFile)
		// Replace {{.Prompt}} with the prompt content (safe-ish, but beware of shell injection if not careful)
		// To be safe, we should probably escape it, but simple replacement is what was asked.
		// However, inserting multi-line text into a shell command string is very fragile.
		// If the prompt contains quotes, it will break the command.
		// We can't easily fix that with simple string replacement.
		// But since we are generating a script file, we can be smarter.

		// If {{.Prompt}} is present, we need to inject the content.
		if strings.Contains(cmdStr, "{{.Prompt}}") {
			// Use shell command substitution to read the prompt content at runtime.
			// This avoids escaping issues in Go and lets the shell handle it.
			// We wrap promptFile in quotes for safety.
			// Example: coco "{{.Prompt}}" -> coco "$(cat "/path/to/prompt.md")"
			cmdStr = strings.ReplaceAll(cmdStr, "{{.Prompt}}", fmt.Sprintf("$(cat %q)", promptFile))
		}

		// Replace {{.ArtifactDir}} just in case
		cmdStr = strings.ReplaceAll(cmdStr, "{{.ArtifactDir}}", absArtifactDir)

		// Prepare script content with auth context injection
		// We capture current environment variables for SSH and Kerberos to ensure the agent
		// has the same access rights as the user running Orion.
		sshAuthSock := os.Getenv("SSH_AUTH_SOCK")
		krb5ccName := os.Getenv("KRB5CCNAME")

		authEnvInjection := ""
		if sshAuthSock != "" {
			authEnvInjection += fmt.Sprintf("export SSH_AUTH_SOCK=%q\n", sshAuthSock)
		}
		if krb5ccName != "" {
			authEnvInjection += fmt.Sprintf("export KRB5CCNAME=%q\n", krb5ccName)
		}

		scriptContent := fmt.Sprintf(`#!/bin/sh
set -x
# Inject Authentication Context
%s
%s
EXIT_CODE=$?
echo $EXIT_CODE > .agent_exit_code
exit $EXIT_CODE
`, authEnvInjection, cmdStr)

		scriptPath := filepath.Join(node.WorktreePath, "run_agent.sh")
		if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
			return fmt.Errorf("failed to write run script: %w", err)
		}

		// Ensure clean state
		_ = os.Remove(filepath.Join(node.WorktreePath, ".agent_exit_code"))

		sessionName := fmt.Sprintf("orion-%s", step.NodeName)
		if err := tmux.SendKeys(sessionName, "./run_agent.sh"); err != nil {
			return fmt.Errorf("failed to send command to tmux: %w", err)
		}

		// Wait for completion
		exitCode, err := e.waitForAgent(sessionName, node.WorktreePath)
		if err != nil {
			return fmt.Errorf("agent execution error: %w", err)
		}
		if exitCode != 0 {
			return fmt.Errorf("agent failed with exit code %d", exitCode)
		}
	} else {
		// Fallback to internal Provider implementation (e.g. QwenProvider)
		// We need to construct agent_pkg.Config
		providerConfig := agent_pkg.Config{
			Provider: agent.Runtime.Provider,
			Model:    agent.Runtime.Model,
			Params:   agent.Runtime.Params,
		}

		prov, err := agent_pkg.NewProvider(providerConfig)
		if err != nil {
			return fmt.Errorf("failed to initialize provider %s: %w", agent.Runtime.Provider, err)
		}

		// We use the finalPrompt we rendered manually
		// 'finalPrompt' is defined in block d. Render Prompt
		// Need to ensure finalPrompt is visible here. It was defined with := inside a block?
		// No, it was defined in block 6.d.
		// Wait, block 6.d might be in scope if not inside {} block.
		// Let's check previous code.
		// It seems finalPrompt was defined: finalPrompt, err := e.renderPrompt(...)
		// If that was inside a block (e.g. if error check), it might be scoped.

		// The error says "undefined: finalPrompt".
		// This means finalPrompt is not in scope.
		// I need to declare it before or ensure it's in scope.

		// Let's look at where finalPrompt is defined.
		// It is defined around line 240.

		// Let's redefine finalPrompt if needed or just assume the previous block made it available.
		// Actually, I can just reload it from file since I wrote it to file.

		promptContent, readErr := os.ReadFile(filepath.Join(node.WorktreePath, "agent_prompt.md"))
		if readErr != nil {
			return fmt.Errorf("failed to read prompt file: %w", readErr)
		}

		output, err := prov.Run(context.Background(), string(promptContent), node.WorktreePath, resolvedEnv)
		if err != nil {
			return fmt.Errorf("agent execution failed: %w", err)
		}
		_ = output
	}

	// 8. Commit Changes
	// The agent modified files in the worktree. We commit them to the shadow branch.
	if err := e.commitChanges(node.WorktreePath, fmt.Sprintf("Agent %s Result", step.ID)); err != nil {
		return fmt.Errorf("failed to commit agent changes: %w", err)
	}

	return nil
}

func (e *Engine) resolveBaseBranch(run *Run, stepDef *types.PipelineStep) (string, error) {
	if len(stepDef.DependsOn) == 0 {
		return run.BaseBranch, nil
	}

	// Find the shadow branch of the dependency
	// Assuming single dependency for now
	depID := stepDef.DependsOn[0]
	for _, s := range run.Steps {
		if s.ID == depID {
			if s.ShadowBranch == "" {
				return "", fmt.Errorf("dependency %s has no shadow branch", depID)
			}
			return s.ShadowBranch, nil
		}
	}
	return "", fmt.Errorf("dependency %s not found", depID)
}

func (e *Engine) getDiffContext(path, from, to string) (string, error) {
	cmd := exec.Command("git", "diff", from, to)
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func (e *Engine) renderPrompt(tmplContent string, data interface{}) (string, error) {
	tmpl, err := template.New("prompt").Parse(tmplContent)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (e *Engine) waitForAgent(sessionName, worktreePath string) (int, error) {
	markerFile := filepath.Join(worktreePath, ".agent_exit_code")
	// TODO: Make timeout configurable or unlimited for long running agents
	timeout := time.After(30 * time.Minute)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return -1, fmt.Errorf("timeout waiting for agent")
		case <-ticker.C:
			// 1. Check if exit code file exists (Success/Failure)
			data, err := os.ReadFile(markerFile)
			if err == nil {
				// File exists, read exit code
				codeStr := strings.TrimSpace(string(data))
				var code int
				fmt.Sscanf(codeStr, "%d", &code)
				return code, nil
			}

			// 2. Check if Tmux session is still alive (Process Status)
			// If marker file is missing BUT session is gone, it means the process was killed/crashed
			// without writing the exit code.
			exists := tmux.SessionExists(sessionName)
			if !exists {
				// Session is gone, but no exit code? Must be killed.
				return -1, fmt.Errorf("agent process (session %s) was killed or crashed", sessionName)
			}
		}
	}
}

func (e *Engine) commitChanges(worktreePath, msg string) error {
	// Clean up transient files before committing
	_ = os.Remove(filepath.Join(worktreePath, ".agent_exit_code"))
	_ = os.Remove(filepath.Join(worktreePath, "agent_prompt.md"))
	_ = os.Remove(filepath.Join(worktreePath, "run_agent.sh"))

	// Configure git user if not set locally (using Orion's config or defaults)
	// We set it locally for this worktree to avoid messing with global config
	config, err := e.wm.GetConfig()
	user := "orion_agent"
	email := "ai@orion.dev"
	if err == nil {
		if config.Git.User != "" {
			user = config.Git.User
		}
		if config.Git.Email != "" {
			email = config.Git.Email
		}
	}

	// Set local config
	setUser := exec.Command("git", "config", "user.name", user)
	setUser.Dir = worktreePath
	_ = setUser.Run()

	setEmail := exec.Command("git", "config", "user.email", email)
	setEmail.Dir = worktreePath
	_ = setEmail.Run()

	// git add .
	addCmd := exec.Command("git", "add", ".")
	addCmd.Dir = worktreePath
	if err := addCmd.Run(); err != nil {
		return err
	}

	// git commit -m msg
	// Check if there are changes first?
	// git diff --cached --quiet returns 0 if no changes
	checkCmd := exec.Command("git", "diff", "--cached", "--quiet")
	checkCmd.Dir = worktreePath
	if err := checkCmd.Run(); err == nil {
		// No changes to commit
		return nil
	}

	commitCmd := exec.Command("git", "commit", "-m", msg)
	commitCmd.Dir = worktreePath
	return commitCmd.Run()
}

func (e *Engine) saveRunStatus(run *Run) error {
	path := filepath.Join(e.wm.RootPath, workspace.MetaDir, workspace.RunsDir, run.ID, "status.json")
	// Ensure parent directory exists to avoid failures when called from tests or
	// auxiliary tooling that may not have created the run directory yet.
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
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

	// Deduplicate runs by ID (in case of stale data or file system glitches)
	seen := make(map[string]bool)
	uniqueRuns := []Run{}
	for _, run := range runs {
		if !seen[run.ID] {
			seen[run.ID] = true
			uniqueRuns = append(uniqueRuns, run)
		}
	}

	return uniqueRuns, nil
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
