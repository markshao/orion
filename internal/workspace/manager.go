package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"devswarm/internal/git"
	"devswarm/internal/tmux"
	"devswarm/internal/types"
)

const (
	RepoDir    = "repo"
	NodesDir   = "nodes"
	MetaDir    = ".devswarm"
	StateFile  = "state.json"
	ConfigFile = "config.yaml"
)

// WorkspaceManager handles all high-level operations on the DevSwarm workspace.
type WorkspaceManager struct {
	RootPath string
	State    *types.State
}

// NewManager creates a manager for an existing workspace.
// It checks if the current directory is a valid DevSwarm workspace root.
func NewManager(rootPath string) (*WorkspaceManager, error) {
	// Strict check: .devswarm MUST exist in the current directory
	metaPath := filepath.Join(rootPath, MetaDir)
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("not a devswarm workspace: %s not found in %s", MetaDir, rootPath)
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

// Init creates a new DevSwarm workspace structure.
// It creates the directories but does NOT clone the repo (that's the caller's job via GitManager).
func Init(rootPath, repoURL string) (*WorkspaceManager, error) {
	// 1. Create directory structure
	dirs := []string{
		filepath.Join(rootPath, RepoDir),
		filepath.Join(rootPath, NodesDir),
		filepath.Join(rootPath, MetaDir),
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

	return wm, nil
}

// SaveState persists the current state to .devswarm/state.json
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

// LoadState loads the state from .devswarm/state.json
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
// 2. Creates shadow branch
// 3. Creates git worktree
// 4. Updates state
func (wm *WorkspaceManager) SpawnNode(nodeName, logicalBranch, baseBranch, purpose string) error {
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
	// Shadow branch format: devswarm/<node_name>/<logical_branch>
	shadowBranch := fmt.Sprintf("devswarm/%s/%s", nodeName, logicalBranch)
	worktreePath := filepath.Join(wm.RootPath, NodesDir, nodeName)

	// 3. Create worktree + shadow branch
	// This runs: git worktree add -b <shadow> <path> <logical>
	if err := git.AddWorktree(wm.State.RepoPath, worktreePath, shadowBranch, logicalBranch); err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	// 4. Update State
	newNode := types.Node{
		Name:          nodeName,
		LogicalBranch: logicalBranch,
		ShadowBranch:  shadowBranch,
		WorktreePath:  worktreePath,
		Purpose:       purpose,
		CreatedAt:     time.Now(),
		// TmuxSession is empty initially
	}

	wm.State.Nodes[nodeName] = newNode

	// 5. Persist State
	if err := wm.SaveState(); err != nil {
		// Rollback? ideally yes, but for MVP let's just warn
		return fmt.Errorf("node created but state save failed: %w", err)
	}

	return nil
}

// EnterNode launches or attaches to a tmux session for the given node.
func (wm *WorkspaceManager) EnterNode(nodeName string) error {
	node, exists := wm.State.Nodes[nodeName]
	if !exists {
		return fmt.Errorf("node '%s' does not exist", nodeName)
	}

	sessionName := fmt.Sprintf("devswarm-%s", nodeName)

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
	sessionName := fmt.Sprintf("devswarm-%s", nodeName)
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
	return wm.SaveState()
}
