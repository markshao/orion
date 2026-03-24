package notification

import "time"

const (
	StateRunning        = "running"
	StateQuietCandidate = "quiet_candidate"
	StateWaitingInput   = "waiting_input"
	StateCompletedIdle  = "completed_idle"
	StateUnknown        = "unknown"
	StateMissing        = "missing"
)

type ServiceConfig struct {
	Enabled             bool
	PollInterval        time.Duration
	SilenceThreshold    time.Duration
	ReminderInterval    time.Duration
	SimilarityThreshold float64
	TailLines           int
	LLMEnabled          bool
}

type Watcher struct {
	NodeName             string    `json:"node_name"`
	SessionName          string    `json:"session_name"`
	PaneID               string    `json:"pane_id"`
	RegisteredAt         time.Time `json:"registered_at"`
	State                string    `json:"state"`
	LastReason           string    `json:"last_reason,omitempty"`
	LastHash             string    `json:"last_hash,omitempty"`
	LastNormalizedScreen string    `json:"last_normalized_screen,omitempty"`
	LastSimilarity       float64   `json:"last_similarity,omitempty"`
	LastChangeAt         time.Time `json:"last_change_at,omitempty"`
	LastObservedAt       time.Time `json:"last_observed_at,omitempty"`
	LastClassifiedHash   string    `json:"last_classified_hash,omitempty"`
	LastClassifiedState  string    `json:"last_classified_state,omitempty"`
	LastClassifiedAt     time.Time `json:"last_classified_at,omitempty"`
	LastLLMReason        string    `json:"last_llm_reason,omitempty"`
	LastNotifyAt         time.Time `json:"last_notify_at,omitempty"`
	LastError            string    `json:"last_error,omitempty"`
}

type Registry struct {
	Watchers map[string]*Watcher `json:"watchers"`
}

type ServiceStatus struct {
	PID        int       `json:"pid"`
	StartedAt  time.Time `json:"started_at,omitempty"`
	LastLoopAt time.Time `json:"last_loop_at,omitempty"`
	LastError  string    `json:"last_error,omitempty"`
}

type Classification struct {
	State  string `json:"state"`
	Reason string `json:"reason"`
}
