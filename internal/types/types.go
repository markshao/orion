package types

import "time"

// Node represents a development unit in Orion.
// A Node can exist without a Tmux session (e.g. just created, or session killed).
type Node struct {
	Name          string    `json:"name"`
	LogicalBranch string    `json:"logical_branch"`         // The user-facing branch (e.g. feature/login)
	BaseBranch    string    `json:"base_branch,omitempty"`  // The base branch (e.g. main)
	ShadowBranch  string    `json:"shadow_branch"`          // The actual branch for this node (e.g. orion/login-test/feature/login)
	WorktreePath  string    `json:"worktree_path"`          // Absolute path to the worktree
	TmuxSession   string    `json:"tmux_session,omitempty"` // Tmux session name, empty if not running
	Label         string    `json:"label,omitempty"`        // User-defined tag (e.g. "review", "test")
	CreatedBy     string    `json:"created_by,omitempty"`   // "user" for human, or <run-id> for workflow
	AppliedRuns   []string  `json:"applied_runs,omitempty"` // List of workflow run IDs applied to this node
	CreatedAt     time.Time `json:"created_at"`
}

// State represents the global state of the Orion workspace.
// This is persisted to .orion/state.json
type State struct {
	RepoURL  string          `json:"repo_url"`
	RepoPath string          `json:"repo_path"` // Absolute path to the main repo (source of truth)
	Nodes    map[string]Node `json:"nodes"`     // Key: Node Name
}

// --- V1 Configuration Types ---

// Config represents the .orion/config.yaml structure
type Config struct {
	Version   int               `yaml:"version"`
	Workspace string            `yaml:"workspace"`
	Git       GitConfig         `yaml:"git"`
	Workflow  map[string]string `yaml:"workflow"`
	Runtime   RuntimeConfig     `yaml:"runtime"`
}

type GitConfig struct {
	MainBranch string `yaml:"main_branch"`
	User       string `yaml:"user,omitempty"`
	Email      string `yaml:"email,omitempty"`
}

type RuntimeConfig struct {
	ArtifactDir string `yaml:"artifact_dir"`
}

// Workflow represents a workflow definition (e.g. workflows/default.yaml)
type Workflow struct {
	Name     string          `yaml:"name"`
	Trigger  WorkflowTrigger `yaml:"trigger"`
	Pipeline []PipelineStep  `yaml:"pipeline"`
}

type WorkflowTrigger struct {
	Event string `yaml:"event"`
}

type PipelineStep struct {
	ID        string   `yaml:"id"`
	Agent     string   `yaml:"agent"`
	Branch    string   `yaml:"branch"`
	Suffix    string   `yaml:"suffix"`
	DependsOn []string `yaml:"depends_on,omitempty"`
}

// Agent represents an agent definition (e.g. agents/ut-agent.yaml)
type Agent struct {
	Name    string       `yaml:"name"`
	Runtime AgentRuntime `yaml:"runtime"`
	Prompt  string       `yaml:"prompt"`
	Env     []string     `yaml:"env"`
}

type AgentRuntime struct {
	Executor  string `yaml:"executor"`
	CodeAgent string `yaml:"code-agent"`
}
