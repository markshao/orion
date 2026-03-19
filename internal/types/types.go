package types

import "time"

// NodeStatus represents the current state of a node
type NodeStatus string

const (
	StatusWorking      NodeStatus = "WORKING"       // Initial state after spawn
	StatusReadyToPush  NodeStatus = "READY_TO_PUSH" // Workflow succeeded, ready to push
	StatusFail         NodeStatus = "FAIL"          // Workflow failed
	StatusPushed       NodeStatus = "PUSHED"        // Successfully pushed to remote
)

// Node represents a development unit in Orion.
// A Node can exist without a Tmux session (e.g. just created, or session killed).
type Node struct {
	Name          string     `json:"name"`
	LogicalBranch string     `json:"logical_branch"`         // The user-facing branch (e.g. feature/login)
	BaseBranch    string     `json:"base_branch,omitempty"`  // The base branch (e.g. main)
	ShadowBranch  string     `json:"shadow_branch"`          // The actual branch for this node (e.g. orion/login-test/feature/login)
	WorktreePath  string     `json:"worktree_path"`          // Absolute path to the worktree
	TmuxSession   string     `json:"tmux_session,omitempty"` // Tmux session name, empty if not running
	Label         string     `json:"label,omitempty"`        // User-defined tag (e.g. "review", "test")
	CreatedBy     string     `json:"created_by,omitempty"`   // "user" for human, or <run-id> for workflow
	AppliedRuns   []string   `json:"applied_runs,omitempty"` // List of workflow run IDs applied to this node
	Status        NodeStatus `json:"status,omitempty"`       // Current status of the node
	CreatedAt     time.Time  `json:"created_at"`
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
	Version   int          `yaml:"version"`
	Workspace string       `yaml:"workspace"`
	Git       GitConfig    `yaml:"git"`
	Agents    AgentsConfig `yaml:"agents"`
	Workflow  map[string]string
	Runtime   RuntimeConfig `yaml:"runtime"`
}

type AgentsConfig struct {
	DefaultProvider string                      `yaml:"default_provider"`
	Providers       map[string]ProviderSettings `yaml:"providers"`
}

type GitConfig struct {
	MainBranch string `yaml:"main_branch"`
	User       string `yaml:"user,omitempty"`
	Email      string `yaml:"email,omitempty"`
}

type RuntimeConfig struct {
	ArtifactDir string `yaml:"artifact_dir"`
}

// --- Workflow V2 Types ---

// StepType represents the type of a workflow step
type StepType string

const (
	StepTypeAgent StepType = "agent" // AI Agent execution step
	StepTypeBash  StepType = "bash"  // Bash command execution step
)

// Workflow represents a workflow definition (e.g. workflows/default.yaml)
type Workflow struct {
	Name     string          `yaml:"name"`
	Trigger  WorkflowTrigger `yaml:"trigger"`
	Pipeline []PipelineStep  `yaml:"pipeline"`
}

type WorkflowTrigger struct {
	Event string `yaml:"event"`
}

// PipelineStep represents a single step in the workflow pipeline
type PipelineStep struct {
	ID        string   `yaml:"id"`
	Type      StepType `yaml:"type,omitempty"` // "agent" or "bash", defaults to "agent" for backward compat
	DependsOn []string `yaml:"depends_on,omitempty"`

	// Agent Step fields (type = "agent" or empty)
	Agent      string `yaml:"agent"`       // Agent configuration name (e.g., "dev-agent")
	BaseBranch string `yaml:"base-branch"` // Base branch for shadow branch creation, required for agent steps

	// Bash Step fields (type = "bash")
	Run  string `yaml:"run"`  // Command to execute
	Node string `yaml:"node"` // Target node (variable reference like "${input.node}" or "${steps.xxx.node}"), empty means run in workflow directory

	// Optional environment variables for this step
	Env map[string]string `yaml:"env,omitempty"`

	// Legacy fields (deprecated, kept for backward compatibility)
	Branch string `yaml:"branch,omitempty"` // Deprecated: replaced by base-branch
	Suffix string `yaml:"suffix,omitempty"` // Deprecated: node name now auto-generated
}

// IsAgent returns true if this is an agent step
func (s PipelineStep) IsAgent() bool {
	return s.Type == StepTypeAgent || s.Type == ""
}

// IsBash returns true if this is a bash step
func (s PipelineStep) IsBash() bool {
	return s.Type == StepTypeBash
}

// Agent represents an agent definition (e.g. agents/dev-agent.yaml)
type Agent struct {
	Name    string       `yaml:"name"`
	Runtime AgentRuntime `yaml:"runtime"`
	Prompt  string       `yaml:"prompt"`
	Env     []string     `yaml:"env"`
}

type AgentRuntime struct {
	Provider string            `yaml:"provider"`
	Model    string            `yaml:"model"`
	Params   map[string]string `yaml:"params"`
	Command  string            `yaml:"command"` // Override or specific command
}

type ProviderSettings struct {
	APIKeyEnv string            `yaml:"api_key_env"`
	Model     string            `yaml:"model"`
	Endpoint  string            `yaml:"endpoint"`
	Command   string            `yaml:"command"` // Custom command template, e.g. 'coco -py "{{.PromptFile}}"'
	Params    map[string]string `yaml:"params"`
}
