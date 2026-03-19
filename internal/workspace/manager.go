package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"orion/internal/git"
	"orion/internal/tmux"
	"orion/internal/types"
	"orion/internal/vscode"
)

const (
	RepoDir       = "main_repo"  // Was "repo"
	WorkspacesDir = "workspaces" // Was NodesDir
	MetaDir       = ".orion"
	StateFile     = "state.json"
	ConfigFile    = "config.yaml"

	// V1 Directories inside .orion
	WorkflowsDir = "workflows"
	AgentsDir    = "agents"
	PromptsDir   = "prompts"
	RunsDir      = "runs"
)

// WorkspaceManager handles all high-level operations on the Orion workspace.
type WorkspaceManager struct {
	RootPath string
	State    *types.State
}

// NewManager creates a manager for an existing workspace.
// It checks if the current directory is a valid Orion workspace root.
func NewManager(rootPath string) (*WorkspaceManager, error) {
	// Strict check: .orion MUST exist in the current directory
	metaPath := filepath.Join(rootPath, MetaDir)
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("not a orion workspace: %s not found in %s", MetaDir, rootPath)
	}

	wm := &WorkspaceManager{
		RootPath: rootPath,
		State:    &types.State{},
	}

	if err := wm.LoadState(); err != nil {
		return nil, fmt.Errorf("failed to load workspace state: %w", err)
	}

	return wm, nil
}

// FindWorkspaceRoot traverses up from startPath looking for .orion directory.
func FindWorkspaceRoot(startPath string) (string, error) {
	current := startPath
	for {
		if _, err := os.Stat(filepath.Join(current, MetaDir)); err == nil {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("not a orion workspace (or any of the parent directories): %s not found", MetaDir)
		}
		current = parent
	}
}

// SyncVSCodeWorkspace updates the .code-workspace file with current nodes
func (wm *WorkspaceManager) SyncVSCodeWorkspace() error {
	var nodes []string
	for name, node := range wm.State.Nodes {
		// Only include user-created nodes
		if node.CreatedBy == "user" {
			nodes = append(nodes, name)
		}
	}
	return vscode.UpdateWorkspaceFile(wm.RootPath, RepoDir, WorkspacesDir, nodes)
}

// Init creates a new Orion workspace structure.
// It creates the directories but does NOT clone the repo (that's the caller's job via GitManager).
func Init(rootPath, repoURL string) (*WorkspaceManager, error) {
	// 1. Create directory structure
	dirs := []string{
		filepath.Join(rootPath, RepoDir),
		filepath.Join(rootPath, WorkspacesDir),
		filepath.Join(rootPath, MetaDir),
		// V1 Directories
		filepath.Join(rootPath, MetaDir, WorkflowsDir),
		filepath.Join(rootPath, MetaDir, AgentsDir),
		filepath.Join(rootPath, MetaDir, PromptsDir),
		filepath.Join(rootPath, MetaDir, RunsDir),
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", d, err)
		}
	}

	// 2. Initialize state
	state := &types.State{
		RepoURL:  repoURL,
		RepoPath: filepath.Join(rootPath, RepoDir),
		Nodes:    make(map[string]types.Node),
	}

	wm := &WorkspaceManager{
		RootPath: rootPath,
		State:    state,
	}

	// 3. Persist initial state
	if err := wm.SaveState(); err != nil {
		return nil, fmt.Errorf("failed to save initial state: %w", err)
	}

	// 4. Generate default V1 configuration files
	if err := wm.generateV1Configs(); err != nil {
		return nil, fmt.Errorf("failed to generate v1 configs: %w", err)
	}

	return wm, nil
}

// generateV1Configs generates default V1 configuration files
func (wm *WorkspaceManager) generateV1Configs() error {
	// 1. config.yaml
	configContent := `version: 1

workspace: workspaces

git:
  main_branch: main
  user: orion
  email: agent@orion.dev

runtime:
  artifact_dir: .orion/runs

agents:
  default_provider: traecli
  providers:
    traecli:
      command: 'traecli "{{.Prompt}}" -py'
    qwen:
      command: 'qwen "{{.Prompt}}" -y'
`
	if err := os.WriteFile(filepath.Join(wm.RootPath, MetaDir, ConfigFile), []byte(configContent), 0644); err != nil {
		return err
	}

	// 2. prompts/base.md
	basePromptContent := `You are an intelligent agent working in the Orion environment.

# Context
- Current Branch: {{.Branch}}
- Artifact Directory: {{.ArtifactDir}}

# Capabilities
1. **Code Changes**: You can edit files in the current directory. All changes will be committed to {{.Branch}}.
2. **Artifacts**: You can generate non-code outputs (reports, summaries) into {{.ArtifactDir}}.
   **IMPORTANT**: {{.ArtifactDir}} is a mounted storage for this execution step.
   - ALL non-code outputs MUST be written to this directory to be persisted and visible externally.
   - For a summary, write to {{.ArtifactDir}}/summary.md.
   - For structured data, write to {{.ArtifactDir}}/report.json.

# Rules
- If you modify any code files, you MUST ensure they are saved. The system will automatically commit all changes in the current directory after you finish.
- Do NOT perform git commit manually; just edit the files.

# Task Specific Instructions
{{.UserPrompt}}

# Version Context
Commit: {{.CommitID}}
`
	if err := os.WriteFile(filepath.Join(wm.RootPath, MetaDir, PromptsDir, "base.md"), []byte(basePromptContent), 0644); err != nil {
		return err
	}

	return nil
}

// SaveState persists the current state to .orion/state.json
func (wm *WorkspaceManager) SaveState() error {
	statePath := filepath.Join(wm.RootPath, MetaDir, StateFile)

	file, err := os.Create(statePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(wm.State)
}

// LoadState loads the state from .orion/state.json
func (wm *WorkspaceManager) LoadState() error {
	statePath := filepath.Join(wm.RootPath, MetaDir, StateFile)

	file, err := os.Open(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Initialize empty state if file doesn't exist
			wm.State = &types.State{Nodes: make(map[string]types.Node)}
			return nil
		}
		return err
	}
	defer file.Close()

	return json.NewDecoder(file).Decode(&wm.State)
}

// SpawnNode creates a new development node.
// 1. Validates inputs (creates logical branch if needed)
// 2. Creates shadow branch (if isShadow=true) or uses logical branch directly
// 3. Creates git worktree
// 4. Updates state
func (wm *WorkspaceManager) SpawnNode(nodeName, logicalBranch, baseBranch, label string, isShadow bool) error {
	// 0. Check if node already exists
	if wm.State.Nodes == nil {
		wm.State.Nodes = make(map[string]types.Node)
	}
	if _, exists := wm.State.Nodes[nodeName]; exists {
		return fmt.Errorf("node '%s' already exists", nodeName)
	}

	// 1. Validate logical branch, create if missing and base provided
	if err := git.VerifyBranch(wm.State.RepoPath, logicalBranch); err != nil {
		if baseBranch == "" {
			return fmt.Errorf("logical branch '%s' invalid: %w. Provide --base to create it", logicalBranch, err)
		}

		fmt.Printf("Logical branch '%s' not found. Creating from '%s'...\n", logicalBranch, baseBranch)
		if err := git.CreateBranch(wm.State.RepoPath, logicalBranch, baseBranch); err != nil {
			return err
		}
	}

	// 2. Define paths and branch names
	worktreePath := filepath.Join(wm.RootPath, WorkspacesDir, nodeName)
	var shadowBranch string

	if isShadow {
		// Shadow Mode: Create a temporary branch based on logicalBranch
		// Naming: orion-shadow/<node_name>/<logical_branch>
		shadowBranch = fmt.Sprintf("orion-shadow/%s/%s", nodeName, logicalBranch)
		// We use logicalBranch as the base for the shadow branch
		if err := git.AddWorktree(wm.State.RepoPath, worktreePath, shadowBranch, logicalBranch); err != nil {
			return fmt.Errorf("failed to create shadow worktree: %w", err)
		}
	} else {
		// Feature Mode: Directly use the logical branch
		shadowBranch = logicalBranch // In this mode, shadow branch IS the logical branch
		// We use logicalBranch as both the branch to checkout and the base (git worktree add <path> <branch>)
		// Note: git worktree add <path> <branch> checks out that branch.
		// Unlike 'git worktree add -b <new> <path> <base>', here we don't use -b.
		// We need to adjust internal/git/git.go AddWorktree to support this, or call a different function.
		// For now, let's see if we can reuse AddWorktree with same names or if we need a new helper.
		// Current AddWorktree uses `git worktree add -b <shadow> <path> <base>`
		// If shadow == base, we should use `git worktree add <path> <base>`
		if err := git.AddWorktree(wm.State.RepoPath, worktreePath, shadowBranch, logicalBranch); err != nil {
			return fmt.Errorf("failed to create worktree on branch '%s': %w", logicalBranch, err)
		}
	}

	// 4. Update State
	newNode := types.Node{
		Name:          nodeName,
		LogicalBranch: logicalBranch,
		BaseBranch:    baseBranch,
		ShadowBranch:  shadowBranch,
		WorktreePath:  worktreePath,
		Label:         label,
		CreatedBy:     "user",
		Status:        types.StatusWorking,
		CreatedAt:     time.Now(),
		// TmuxSession is empty initially
	}

	wm.State.Nodes[nodeName] = newNode

	// 5. Apply Git Config from config.yaml
	if err := wm.applyGitConfigToWorktree(worktreePath); err != nil {
		fmt.Printf("Warning: Failed to apply git config: %v\n", err)
	}

	// 6. Persist State
	if err := wm.SaveState(); err != nil {
		// Rollback? ideally yes, but for MVP let's just warn
		return fmt.Errorf("node created but state save failed: %w", err)
	}

	// 7. Update VSCode Workspace
	if err := wm.SyncVSCodeWorkspace(); err != nil {
		fmt.Printf("Warning: Failed to update VSCode workspace file: %v\n", err)
	}

	return nil
}

// CreateAgentNode creates a dedicated ephemeral node for an AI agent.
// It sets up the shadow branch, worktree, and tmux session.
func (wm *WorkspaceManager) CreateAgentNode(nodeName, shadowBranch, baseBranch, createdBy string) (*types.Node, error) {
	// Agent nodes are stored in .orion/agent-nodes/ to keep them hidden from the main workspace list
	agentNodesDir := filepath.Join(wm.RootPath, MetaDir, "agent-nodes")
	if err := os.MkdirAll(agentNodesDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create agent nodes directory: %w", err)
	}
	worktreePath := filepath.Join(agentNodesDir, nodeName)

	// 1. Create Shadow Branch & Worktree
	if err := git.AddWorktree(wm.State.RepoPath, worktreePath, shadowBranch, baseBranch); err != nil {
		return nil, err
	}

	// 2. Apply Git Config from config.yaml
	if err := wm.applyGitConfigToWorktree(worktreePath); err != nil {
		fmt.Printf("Warning: Failed to apply git config: %v\n", err)
	}

	// 3. Create Tmux Session
	sessionName := fmt.Sprintf("orion-%s", nodeName)
	if err := tmux.NewSession(sessionName, worktreePath); err != nil {
		return nil, fmt.Errorf("failed to create tmux session: %w", err)
	}

	// 3. Update State
	node := types.Node{
		Name:          nodeName,
		LogicalBranch: baseBranch, // Logically related to base
		ShadowBranch:  shadowBranch,
		WorktreePath:  worktreePath,
		Label:         "agent",
		CreatedBy:     createdBy,
		TmuxSession:   sessionName,
		CreatedAt:     time.Now(),
	}

	if wm.State.Nodes == nil {
		wm.State.Nodes = make(map[string]types.Node)
	}
	wm.State.Nodes[nodeName] = node

	// 4. Persist State
	if err := wm.SaveState(); err != nil {
		return nil, err
	}
	// We might not want to sync agent nodes to VSCode workspace to avoid clutter
	// wm.SyncVSCodeWorkspace()

	return &node, nil
}

// applyGitConfigToWorktree applies git user.name and user.email from config.yaml to the worktree.
func (wm *WorkspaceManager) applyGitConfigToWorktree(worktreePath string) error {
	config, err := wm.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Only apply if values are set in config
	if config.Git.User != "" {
		if err := git.SetConfig(worktreePath, "user.name", config.Git.User); err != nil {
			return fmt.Errorf("failed to set user.name: %w", err)
		}
	}
	if config.Git.Email != "" {
		if err := git.SetConfig(worktreePath, "user.email", config.Git.Email); err != nil {
			return fmt.Errorf("failed to set user.email: %w", err)
		}
	}

	return nil
}

// EnterNode launches or attaches to a tmux session for the given node.
func (wm *WorkspaceManager) EnterNode(nodeName string) error {
	node, exists := wm.State.Nodes[nodeName]
	if !exists {
		return fmt.Errorf("node '%s' does not exist", nodeName)
	}

	// Use node's configured session name if available, otherwise construct default
	sessionName := node.TmuxSession
	if sessionName == "" {
		sessionName = fmt.Sprintf("orion-%s", nodeName)
	}

	// Check if we are already inside tmux
	if tmux.IsInsideTmux() {
		// If inside tmux, we should switch client instead of attaching
		// But first ensure the target session exists
		if !tmux.SessionExists(sessionName) {
			if err := tmux.NewSession(sessionName, node.WorktreePath); err != nil {
				return fmt.Errorf("failed to create session: %w", err)
			}
		}
		return tmux.SwitchClient(sessionName)
	}

	// If outside tmux, attach (or create and attach)
	// This will replace the current process
	return tmux.EnsureAndAttach(sessionName, node.WorktreePath)
}

// RemoveNode removes a node and cleans up resources.
func (wm *WorkspaceManager) RemoveNode(nodeName string) error {
	node, exists := wm.State.Nodes[nodeName]
	if !exists {
		return fmt.Errorf("node '%s' does not exist", nodeName)
	}

	// 1. Kill Tmux Session
	sessionName := fmt.Sprintf("orion-%s", nodeName)
	if err := tmux.KillSession(sessionName); err != nil {
		fmt.Printf("Warning: Failed to kill tmux session: %v\n", err)
	}

	// 2. Remove Worktree
	// We check if path exists first to avoid git error
	if _, err := os.Stat(node.WorktreePath); !os.IsNotExist(err) {
		if err := git.RemoveWorktree(wm.State.RepoPath, node.WorktreePath); err != nil {
			fmt.Printf("Warning: Failed to remove worktree: %v. You may need to clean up manually.\n", err)
		}
	}

	// 3. Delete Shadow Branch
	if err := git.DeleteBranch(wm.State.RepoPath, node.ShadowBranch); err != nil {
		fmt.Printf("Warning: Failed to delete shadow branch: %v\n", err)
	}

	// 4. Remove from State
	delete(wm.State.Nodes, nodeName)

	// 5. Persist State
	if err := wm.SaveState(); err != nil {
		return err
	}

	// 6. Update VSCode Workspace
	if err := wm.SyncVSCodeWorkspace(); err != nil {
		fmt.Printf("Warning: Failed to update VSCode workspace file: %v\n", err)
	}

	return nil
}

// UpdateNodeStatus updates the status of a node and persists the state.
func (wm *WorkspaceManager) UpdateNodeStatus(nodeName string, status types.NodeStatus) error {
	node, exists := wm.State.Nodes[nodeName]
	if !exists {
		return fmt.Errorf("node '%s' does not exist", nodeName)
	}

	node.Status = status
	wm.State.Nodes[nodeName] = node

	if err := wm.SaveState(); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	return nil
}

// MergeNode merges the node's shadow branch into its logical branch.
func (wm *WorkspaceManager) MergeNode(nodeName string, cleanup bool) error {
	node, exists := wm.State.Nodes[nodeName]
	if !exists {
		return fmt.Errorf("node '%s' does not exist", nodeName)
	}

	if node.ShadowBranch == node.LogicalBranch {
		fmt.Printf("Node '%s' is running directly on logical branch '%s'. No merge needed.\n", nodeName, node.LogicalBranch)
		// If cleanup is requested, we still proceed to cleanup
	} else {
		fmt.Printf("Merging node '%s' (shadow: %s) into logical branch '%s'...\n", nodeName, node.ShadowBranch, node.LogicalBranch)

		commitMsg := fmt.Sprintf("Squash merge from Orion node '%s'", nodeName)
		if err := git.SquashMerge(wm.State.RepoPath, node.LogicalBranch, node.ShadowBranch, commitMsg); err != nil {
			return fmt.Errorf("merge failed: %w", err)
		}

		fmt.Println("Merge successful!")
	}

	if cleanup {
		fmt.Printf("Cleaning up node '%s'...\n", nodeName)
		return wm.RemoveNode(nodeName)
	}

	return nil
}

// FindNodeByPath finds which node (if any) contains the given file path.
// It returns the node name and the node object.
func (wm *WorkspaceManager) FindNodeByPath(path string) (string, *types.Node, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", nil, err
	}

	// Resolve symlinks to handle case sensitivity on macOS properly
	// e.g. /Users/foo/Desktop vs /Users/foo/desktop
	canonicalPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		// If path doesn't exist, we can't eval symlinks, fall back to absPath
		canonicalPath = absPath
	}

	// Check if path is inside any node's worktree
	for name, node := range wm.State.Nodes {
		// Also resolve node worktree path to canonical form
		nodePath, err := filepath.EvalSymlinks(node.WorktreePath)
		if err != nil {
			nodePath = node.WorktreePath
		}

		// We use simple string prefix check, but to be safe we should check directory boundary
		// e.g. /foo/bar matches /foo/bar/baz but not /foo/bar-baz
		rel, err := filepath.Rel(nodePath, canonicalPath)
		if err == nil && !filepath.IsAbs(rel) && rel != ".." && !filepath.HasPrefix(rel, "../") {
			return name, &node, nil
		}

		// Fallback: Case-insensitive check for macOS/Windows
		// If EvalSymlinks didn't normalize case (e.g. if paths don't fully exist or on some FS),
		// we try comparing lowercase versions.
		if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
			nodePathLower := strings.ToLower(nodePath)
			canonicalPathLower := strings.ToLower(canonicalPath)
			rel, err := filepath.Rel(nodePathLower, canonicalPathLower)
			if err == nil && !filepath.IsAbs(rel) && rel != ".." && !filepath.HasPrefix(rel, "../") {
				return name, &node, nil
			}
		}
	}

	return "", nil, nil
}
