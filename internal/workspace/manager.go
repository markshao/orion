package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"devswarm/internal/git"
	"devswarm/internal/tmux"
	"devswarm/internal/types"
	"devswarm/internal/vscode"
)

const (
	RepoDir       = "main_repo"  // Was "repo"
	WorkspacesDir = "workspaces" // Was NodesDir
	MetaDir       = ".devswarm"
	StateFile     = "state.json"
	ConfigFile    = "config.yaml"
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

// FindWorkspaceRoot traverses up from startPath looking for .devswarm directory.
func FindWorkspaceRoot(startPath string) (string, error) {
	current := startPath
	for {
		if _, err := os.Stat(filepath.Join(current, MetaDir)); err == nil {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("not a devswarm workspace (or any of the parent directories): %s not found", MetaDir)
		}
		current = parent
	}
}

// SyncVSCodeWorkspace updates the .code-workspace file with current nodes
func (wm *WorkspaceManager) SyncVSCodeWorkspace() error {
	var nodes []string
	for name := range wm.State.Nodes {
		nodes = append(nodes, name)
	}
	return vscode.UpdateWorkspaceFile(wm.RootPath, RepoDir, WorkspacesDir, nodes)
}

// Init creates a new DevSwarm workspace structure.
// It creates the directories but does NOT clone the repo (that's the caller's job via GitManager).
func Init(rootPath, repoURL string) (*WorkspaceManager, error) {
	// 1. Create directory structure
	dirs := []string{
		filepath.Join(rootPath, RepoDir),
		filepath.Join(rootPath, WorkspacesDir),
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
// 2. Creates shadow branch (if isShadow=true) or uses logical branch directly
// 3. Creates git worktree
// 4. Updates state
func (wm *WorkspaceManager) SpawnNode(nodeName, logicalBranch, baseBranch, purpose string, isShadow bool) error {
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
		// Naming: ds-shadow/<node_name>/<logical_branch>
		shadowBranch = fmt.Sprintf("ds-shadow/%s/%s", nodeName, logicalBranch)
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

	// 6. Update VSCode Workspace
	if err := wm.SyncVSCodeWorkspace(); err != nil {
		fmt.Printf("Warning: Failed to update VSCode workspace file: %v\n", err)
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
	if err := wm.SaveState(); err != nil {
		return err
	}

	// 6. Update VSCode Workspace
	if err := wm.SyncVSCodeWorkspace(); err != nil {
		fmt.Printf("Warning: Failed to update VSCode workspace file: %v\n", err)
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

		commitMsg := fmt.Sprintf("Squash merge from DevSwarm node '%s'", nodeName)
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
