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

type watcherObservation struct {
	now              time.Time
	sessionName      string
	screen           string
	screenHash       string
	normalizedScreen string
	similarity       float64
	stable           bool
	stableFor        time.Duration
}

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

func EnsureWatcher(rootPath, nodeName, label, sessionName string) error {
	primaryPane, err := tmux.GetPrimaryPane(sessionName)
	if err != nil {
		return err
	}

	return UpdateRegistry(rootPath, func(registry *Registry) error {
		if watcher, ok := registry.Watchers[nodeName]; ok {
			if watcher.PaneID != "" && tmux.PaneExists(watcher.PaneID) {
				watcher.Label = label
				return nil
			}
			watcher.Label = label
			watcher.SessionName = sessionName
			watcher.PaneID = primaryPane.PaneID
			watcher.LastError = ""
			return nil
		}

		registry.Watchers[nodeName] = &Watcher{
			NodeName:       nodeName,
			Label:          label,
			SessionName:    sessionName,
			PaneID:         primaryPane.PaneID,
			RegisteredAt:   time.Now(),
			State:          StateRunning,
			StateEnteredAt: time.Now(),
			LastReason:     "registered",
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

func AcknowledgeWaitEvent(rootPath, nodeName string) error {
	return UpdateRegistry(rootPath, func(registry *Registry) error {
		watcher, ok := registry.Watchers[nodeName]
		if !ok {
			return nil
		}
		watcher.AckedWaitEventID = watcher.WaitEventID
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
	watcher.LastObservedAt = now

	if watcher.PaneID == "" || !tmux.PaneExists(watcher.PaneID) {
		transitionWatcherState(watcher, StateMissing, "pane_missing", now)
		watcher.LastError = "pane is no longer available"
		return
	}

	meta, err := tmux.GetPaneMeta(watcher.PaneID)
	if err != nil {
		transitionWatcherState(watcher, StateUnknown, "pane_metadata_error", now)
		watcher.LastError = err.Error()
		return
	}

	screen, err := tmux.CapturePane(watcher.PaneID, meta.AlternateOn, cfg.TailLines)
	if err != nil && meta.AlternateOn {
		screen, err = tmux.CapturePane(watcher.PaneID, false, cfg.TailLines)
	}
	if err != nil {
		transitionWatcherState(watcher, StateUnknown, "capture_error", now)
		watcher.LastError = err.Error()
		return
	}

	observation := buildWatcherObservation(watcher, meta.SessionName, screen, now, cfg.SimilarityThreshold)
	applyWatcherObservation(watcher, observation, cfg, classifier)
	watcher.LastError = ""
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

func buildWatcherObservation(watcher *Watcher, sessionName, screen string, now time.Time, similarityThreshold float64) watcherObservation {
	previousScreen := watcher.LastNormalizedScreen
	normalizedScreen := normalizeScreen(screen)
	similarity := screenSimilarity(previousScreen, normalizedScreen)
	stable := previousScreen != "" && similarity >= similarityThreshold

	if !stable {
		watcher.StableSince = now
	}
	if watcher.StableSince.IsZero() {
		watcher.StableSince = now
	}

	watcher.SessionName = sessionName
	watcher.LastNormalizedScreen = normalizedScreen
	watcher.LastSimilarity = similarity
	watcher.LastHash = hashScreen(screen)
	watcher.LastChangeAt = now
	if stable {
		watcher.LastChangeAt = watcher.StableSince
	}

	return watcherObservation{
		now:              now,
		sessionName:      sessionName,
		screen:           screen,
		screenHash:       watcher.LastHash,
		normalizedScreen: normalizedScreen,
		similarity:       similarity,
		stable:           stable,
		stableFor:        now.Sub(watcher.StableSince),
	}
}

func applyWatcherObservation(watcher *Watcher, observation watcherObservation, cfg ServiceConfig, classifier SnapshotClassifier) {
	previousState := watcher.State
	if !observation.stable {
		transitionWatcherState(watcher, StateRunning, fmt.Sprintf("screen_changed_similarity=%.4f", observation.similarity), observation.now)
		return
	}

	transitionWatcherState(watcher, StateQuietCandidate, fmt.Sprintf("stable_screen_similarity=%.4f", observation.similarity), observation.now)
	classification := classifyQuietScreen(watcher, classifier, observation.screen, observation.stableFor)
	transitionWatcherState(watcher, classification.State, classification.Reason, observation.now)

	if shouldNotify(previousState, watcher, cfg.ReminderInterval, observation.now) {
		if err := sendWatcherNotification(watcher.NodeName, watcher.Label, classification.Reason); err != nil {
			watcher.LastError = err.Error()
			return
		}
		watcher.LastNotifyAt = observation.now
		watcher.NotifyCount++
	}
}

func transitionWatcherState(watcher *Watcher, nextState, reason string, now time.Time) {
	if watcher.State != nextState {
		if nextState == StateWaitingInput && watcher.State != StateWaitingInput {
			watcher.WaitEventID++
		}
		watcher.State = nextState
		watcher.StateEnteredAt = now
	}
	watcher.LastReason = reason
	if nextState != StateWaitingInput {
		watcher.NotifyCount = 0
	}
}

func shouldNotify(previousState string, watcher *Watcher, reminderInterval time.Duration, now time.Time) bool {
	if watcher.State != StateWaitingInput {
		return false
	}
	if previousState != StateWaitingInput {
		return true
	}
	if watcher.LastNotifyAt.IsZero() {
		return true
	}

	nextReminderAfter := reminderInterval
	for i := 1; i < watcher.NotifyCount; i++ {
		nextReminderAfter *= 2
	}
	return now.Sub(watcher.LastNotifyAt) >= nextReminderAfter
}
