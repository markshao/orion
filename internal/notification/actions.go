package notification

import (
	"fmt"
	"strings"

	"orion/internal/tmux"
)

type CardActionMetadata struct {
	Action     string `json:"action"`
	NodeName   string `json:"node_name"`
	Label      string `json:"label,omitempty"`
	PaneID     string `json:"pane_id,omitempty"`
	WaitEvent  int    `json:"wait_event_id,omitempty"`
	ScreenHash string `json:"screen_hash,omitempty"`
	SentAt     string `json:"sent_at,omitempty"`
}

func MuteWaitReminder(rootPath string, meta CardActionMetadata) error {
	if strings.TrimSpace(meta.NodeName) == "" {
		return fmt.Errorf("missing node_name in action metadata")
	}

	return UpdateRegistry(rootPath, func(registry *Registry) error {
		watcher, ok := registry.Watchers[meta.NodeName]
		if !ok {
			return fmt.Errorf("node %s not found", meta.NodeName)
		}
		if strings.TrimSpace(meta.PaneID) != "" && watcher.PaneID != strings.TrimSpace(meta.PaneID) {
			return fmt.Errorf("pane mismatch for node %s", meta.NodeName)
		}

		targetEvent := watcher.WaitEventID
		if meta.WaitEvent > 0 {
			targetEvent = meta.WaitEvent
		}
		if targetEvent <= 0 {
			return nil
		}
		if targetEvent < watcher.WaitEventID {
			return nil
		}
		watcher.MutedWaitEventID = targetEvent
		return nil
	})
}

func RouteReplyToNode(rootPath string, meta CardActionMetadata, reply string) error {
	reply = strings.TrimSpace(reply)
	if reply == "" {
		return fmt.Errorf("empty reply")
	}
	if strings.TrimSpace(meta.NodeName) == "" {
		return fmt.Errorf("missing node_name in action metadata")
	}

	var targetPane string
	err := UpdateRegistry(rootPath, func(registry *Registry) error {
		watcher, ok := registry.Watchers[meta.NodeName]
		if !ok {
			return fmt.Errorf("node %s not found", meta.NodeName)
		}
		if watcher.State != StateWaitingInput {
			return fmt.Errorf("node %s is not waiting for input", meta.NodeName)
		}
		if strings.TrimSpace(meta.PaneID) != "" && watcher.PaneID != strings.TrimSpace(meta.PaneID) {
			return fmt.Errorf("pane mismatch for node %s", meta.NodeName)
		}
		// Accept replies from slightly stale cards and route to the current
		// waiting pane. This keeps mobile quick-reply robust when state flips.
		if meta.WaitEvent > 0 && watcher.WaitEventID < meta.WaitEvent {
			return fmt.Errorf("invalid wait_event_id for node %s", meta.NodeName)
		}
		targetPane = watcher.PaneID
		return nil
	})
	if err != nil {
		return err
	}
	if targetPane == "" {
		return fmt.Errorf("target pane is empty")
	}

	if err := tmux.SendLiteralKeysToPane(targetPane, reply); err != nil {
		return err
	}
	for i := 0; i < enterCountForReply(reply); i++ {
		if err := tmux.SendEnterToPane(targetPane); err != nil {
			return err
		}
	}
	return nil
}

func enterCountForReply(reply string) int {
	reply = strings.TrimLeft(reply, " \t")
	if strings.HasPrefix(reply, "$") {
		return 2
	}
	return 1
}
