package workflow

import (
	"time"
)

type RunStatus string

const (
	StatusPending RunStatus = "pending"
	StatusRunning RunStatus = "running"
	StatusSuccess RunStatus = "success"
	StatusFailed  RunStatus = "failed"
)

// Run represents a single execution of a workflow.
// This is persisted to .orion/runs/<run-id>/status.json
type Run struct {
	ID        string       `json:"id"`
	Workflow  string       `json:"workflow"`
	Trigger   string       `json:"trigger"` // e.g. "commit", "manual"
	TriggerData string     `json:"trigger_data,omitempty"` // e.g. commit hash
	BaseBranch string      `json:"base_branch"`
	TriggeredByNode string `json:"triggered_by_node,omitempty"` // The human node that triggered this run
	Status    RunStatus    `json:"status"`
	StartTime time.Time    `json:"start_time"`
	EndTime   time.Time    `json:"end_time,omitempty"`
	Steps     []StepStatus `json:"steps"`
}

type StepStatus struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"`                     // "agent" or "bash"
	Agent        string    `json:"agent,omitempty"`          // for agent steps
	Status       RunStatus `json:"status"`
	StartTime    time.Time `json:"start_time,omitempty"`
	EndTime      time.Time `json:"end_time,omitempty"`
	NodeName     string    `json:"node_name,omitempty"`
	ShadowBranch string    `json:"shadow_branch,omitempty"`
	LogPath      string    `json:"log_path,omitempty"`
	Error        string    `json:"error,omitempty"`
}
