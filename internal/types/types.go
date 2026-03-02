package types

import "time"

// Node represents a development unit in DevSwarm.
// A Node can exist without a Tmux session (e.g. just created, or session killed).
type Node struct {
	Name          string    `json:"name"`
	LogicalBranch string    `json:"logical_branch"` // The user-facing branch (e.g. feature/login)
	ShadowBranch  string    `json:"shadow_branch"`  // The actual branch for this node (e.g. devswarm/login-test/feature/login)
	WorktreePath  string    `json:"worktree_path"`  // Absolute path to the worktree
	TmuxSession   string    `json:"tmux_session,omitempty"` // Tmux session name, empty if not running
	Purpose       string    `json:"purpose,omitempty"` // User-defined tag (e.g. "review", "test-agent")
	CreatedAt     time.Time `json:"created_at"`
}

// State represents the global state of the DevSwarm workspace.
// This is persisted to .devswarm/state.json
type State struct {
	RepoURL  string          `json:"repo_url"`
	RepoPath string          `json:"repo_path"` // Absolute path to the main repo (source of truth)
	Nodes    map[string]Node `json:"nodes"`     // Key: Node Name
}
