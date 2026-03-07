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
// This is persisted to .devswarm/runs/<run-id>/status.json
type Run struct {
	ID        string       `json:"id"`
	Workflow  string       `json:"workflow"`
	Trigger   string       `json:"trigger"` // e.g. "commit", "manual"
	Status    RunStatus    `json:"status"`
	StartTime time.Time    `json:"start_time"`
	EndTime   time.Time    `json:"end_time,omitempty"`
	Steps     []StepStatus `json:"steps"`
}

type StepStatus struct {
	ID        string    `json:"id"`
	Agent     string    `json:"agent"`
	Status    RunStatus `json:"status"`
	StartTime time.Time `json:"start_time,omitempty"`
	EndTime   time.Time `json:"end_time,omitempty"`
	NodeName  string    `json:"node_name,omitempty"`
	LogPath   string    `json:"log_path,omitempty"`
	Error     string    `json:"error,omitempty"`
}
