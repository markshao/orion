package notification

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"orion/internal/log"
	"orion/internal/tmux"
)

func GetServiceStatus(rootPath string) (*ServiceStatus, bool, error) {
	status, err := ReadStatus(rootPath)
	if err != nil {
		return nil, false, err
	}
	running := IsProcessRunning(status.PID)
	return status, running, nil
}

func EnsureStarted(rootPath string) error {
	cfg, err := LoadServiceConfig(rootPath)
	if err != nil {
		return err
	}
	if !cfg.Enabled {
		return nil
	}

	status, running, err := GetServiceStatus(rootPath)
	if err == nil && running {
		return nil
	}
	if status != nil && !running && status.PID > 0 {
		_ = RemovePID(rootPath)
	}

	if err := ensureRuntimeDir(rootPath); err != nil {
		return err
	}

	executable, err := os.Executable()
	if err != nil {
		return err
	}

	logFile, err := os.OpenFile(logPath(rootPath), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer logFile.Close()

	cmd := exec.Command(executable, "notification-service", "run", "--workspace", rootPath)
	cmd.Dir = rootPath
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start notification service: %w", err)
	}
	_ = WritePID(rootPath, cmd.Process.Pid)
	_ = WriteStatus(rootPath, &ServiceStatus{
		PID:        cmd.Process.Pid,
		StartedAt:  time.Now(),
		LastLoopAt: time.Now(),
	})
	if err := cmd.Process.Release(); err != nil {
		return fmt.Errorf("failed to release notification service process: %w", err)
	}
	return nil
}

func Stop(rootPath string) error {
	pid, err := ReadPID(rootPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if !IsProcessRunning(pid) {
		_ = RemovePID(rootPath)
		return nil
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return err
	}

	for i := 0; i < 20; i++ {
		if !IsProcessRunning(pid) {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}

	_ = RemovePID(rootPath)
	status := &ServiceStatus{}
	_ = WriteStatus(rootPath, status)
	return nil
}

func EnsureWatcher(rootPath, nodeName, sessionName string) error {
	primaryPane, err := tmux.GetPrimaryPane(sessionName)
	if err != nil {
		return err
	}

	return UpdateRegistry(rootPath, func(registry *Registry) error {
		if watcher, ok := registry.Watchers[nodeName]; ok {
			if watcher.PaneID != "" && tmux.PaneExists(watcher.PaneID) {
				return nil
			}
			watcher.SessionName = sessionName
			watcher.PaneID = primaryPane.PaneID
			watcher.LastError = ""
			return nil
		}

		registry.Watchers[nodeName] = &Watcher{
			NodeName:     nodeName,
			SessionName:  sessionName,
			PaneID:       primaryPane.PaneID,
			RegisteredAt: time.Now(),
			State:        StateRunning,
			LastReason:   "registered",
		}
		return nil
	})
}

func UnregisterWatcher(rootPath, nodeName string) error {
	return UpdateRegistry(rootPath, func(registry *Registry) error {
		delete(registry.Watchers, nodeName)
		return nil
	})
}

func Run(rootPath string) error {
	cfg, err := LoadServiceConfig(rootPath)
	if err != nil {
		return err
	}
	if err := ensureRuntimeDir(rootPath); err != nil {
		return err
	}

	status := &ServiceStatus{
		PID:        os.Getpid(),
		StartedAt:  time.Now(),
		LastLoopAt: time.Now(),
	}
	if err := WritePID(rootPath, status.PID); err != nil {
		return err
	}
	if err := WriteStatus(rootPath, status); err != nil {
		return err
	}
	defer func() {
		_ = RemovePID(rootPath)
		status.PID = 0
		_ = WriteStatus(rootPath, status)
	}()

	var classifier SnapshotClassifier
	if cfg.LLMEnabled {
		llmClassifier, err := NewLLMClassifier()
		if err != nil {
			status.LastError = fmt.Sprintf("failed to initialize LLM classifier: %v", err)
			log.Error("%s", status.LastError)
		} else {
			classifier = llmClassifier
		}
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)

	if err := tick(rootPath, cfg, classifier, status); err != nil {
		status.LastError = err.Error()
		_ = WriteStatus(rootPath, status)
	}

	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-signals:
			return nil
		case <-ticker.C:
			cfg, err = LoadServiceConfig(rootPath)
			if err != nil {
				status.LastError = err.Error()
				_ = WriteStatus(rootPath, status)
				continue
			}
			if err := tick(rootPath, cfg, classifier, status); err != nil {
				status.LastError = err.Error()
			} else {
				status.LastError = ""
			}
			_ = WriteStatus(rootPath, status)
		}
	}
}

func tick(rootPath string, cfg ServiceConfig, classifier SnapshotClassifier, status *ServiceStatus) error {
	status.LastLoopAt = time.Now()
	if !cfg.Enabled {
		return nil
	}

	registry, err := ReadRegistry(rootPath)
	if err != nil {
		return err
	}

	updated := make(map[string]*Watcher, len(registry.Watchers))
	for nodeName, watcher := range registry.Watchers {
		clone := *watcher
		evaluateWatcher(&clone, cfg, classifier)
		updated[nodeName] = &clone
	}

	return UpdateRegistry(rootPath, func(current *Registry) error {
		for nodeName, evaluated := range updated {
			existing, ok := current.Watchers[nodeName]
			if !ok {
				continue
			}
			if existing.PaneID != evaluated.PaneID {
				continue
			}
			current.Watchers[nodeName] = evaluated
		}
		return nil
	})
}

func evaluateWatcher(watcher *Watcher, cfg ServiceConfig, classifier SnapshotClassifier) {
	now := time.Now()
	previousState := watcher.State
	watcher.LastObservedAt = now

	if watcher.PaneID == "" || !tmux.PaneExists(watcher.PaneID) {
		watcher.State = StateMissing
		watcher.LastReason = "pane_missing"
		watcher.LastError = "pane is no longer available"
		return
	}

	meta, err := tmux.GetPaneMeta(watcher.PaneID)
	if err != nil {
		watcher.State = StateUnknown
		watcher.LastReason = "pane_metadata_error"
		watcher.LastError = err.Error()
		return
	}

	screen, err := tmux.CapturePane(watcher.PaneID, meta.AlternateOn, cfg.TailLines)
	if err != nil && meta.AlternateOn {
		screen, err = tmux.CapturePane(watcher.PaneID, false, cfg.TailLines)
	}
	if err != nil {
		watcher.State = StateUnknown
		watcher.LastReason = "capture_error"
		watcher.LastError = err.Error()
		return
	}

	screenHash := hashScreen(screen)
	normalizedScreen := normalizeScreen(screen)
	previousScreen := watcher.LastNormalizedScreen
	watcher.LastNormalizedScreen = normalizedScreen
	watcher.LastSimilarity = screenSimilarity(previousScreen, normalizedScreen)
	if watcher.LastHash == "" || watcher.LastHash != screenHash {
		watcher.LastHash = screenHash
	}
	if previousScreen == "" || watcher.LastSimilarity < cfg.SimilarityThreshold {
		watcher.LastChangeAt = now
	}

	stableFor := now.Sub(watcher.LastChangeAt)
	classification := HeuristicClassify(previousScreen, normalizedScreen, cfg.SimilarityThreshold)
	if classification.State == StateQuietCandidate {
		classification = classifyQuietScreen(watcher, classifier, screen, stableFor)
	}

	watcher.SessionName = meta.SessionName
	watcher.State = classification.State
	watcher.LastReason = classification.Reason
	watcher.LastError = ""

	if shouldNotify(previousState, watcher, cfg.ReminderInterval) {
		if err := NotifyWatcher(watcher.NodeName, classification.Reason); err != nil {
			watcher.LastError = err.Error()
		} else {
			watcher.LastNotifyAt = now
		}
	}
}

func classifyQuietScreen(watcher *Watcher, classifier SnapshotClassifier, screen string, stableFor time.Duration) Classification {
	if classifier == nil {
		return Classification{State: StateUnknown, Reason: "stable_output_no_classifier"}
	}

	if watcher.LastClassifiedHash == watcher.LastHash && watcher.LastClassifiedState != "" {
		return Classification{
			State:  watcher.LastClassifiedState,
			Reason: watcher.LastLLMReason,
		}
	}

	classification, err := classifier.Classify(watcher.NodeName, screen, stableFor)
	if err != nil {
		return Classification{State: StateUnknown, Reason: "llm_classification_failed"}
	}

	watcher.LastClassifiedHash = watcher.LastHash
	watcher.LastClassifiedState = classification.State
	watcher.LastClassifiedAt = time.Now()
	watcher.LastLLMReason = classification.Reason
	return classification
}

func shouldNotify(previousState string, watcher *Watcher, reminderInterval time.Duration) bool {
	if watcher.State != StateWaitingInput {
		return false
	}
	if previousState != StateWaitingInput {
		return true
	}
	if watcher.LastNotifyAt.IsZero() {
		return true
	}
	return time.Since(watcher.LastNotifyAt) >= reminderInterval
}
