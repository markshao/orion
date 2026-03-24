package notification

import (
	"testing"
	"time"
)

type stubClassifier struct {
	calls  int
	state  string
	reason string
}

func (s *stubClassifier) Classify(nodeName, screen string, stableFor time.Duration) (Classification, error) {
	s.calls++
	return Classification{State: s.state, Reason: s.reason}, nil
}

func TestClassifyQuietScreenCachesByScreenHash(t *testing.T) {
	watcher := &Watcher{
		NodeName: "demo",
		LastHash: hashScreen("waiting for approval"),
	}
	classifier := &stubClassifier{state: StateWaitingInput, reason: "approval prompt"}

	first := classifyQuietScreen(watcher, classifier, "waiting for approval", 5*time.Second)
	if first.State != StateWaitingInput {
		t.Fatalf("expected first classification to return waiting_input, got %s", first.State)
	}
	if classifier.calls != 1 {
		t.Fatalf("expected classifier to be called once, got %d", classifier.calls)
	}

	second := classifyQuietScreen(watcher, classifier, "waiting for approval", 10*time.Second)
	if second.State != StateWaitingInput {
		t.Fatalf("expected cached classification to return waiting_input, got %s", second.State)
	}
	if classifier.calls != 1 {
		t.Fatalf("expected cached classification to avoid extra classifier calls, got %d", classifier.calls)
	}
}

func TestStablePromptFlowTriggersClassifier(t *testing.T) {
	previous := `
would you like to run the following command?
1. yes, proceed
2. no
press enter to confirm or esc to cancel
`
	current := previous

	classification := HeuristicClassify(previous, current, 0.99)
	if classification.State != StateQuietCandidate {
		t.Fatalf("expected stable prompt to become quiet_candidate, got %s", classification.State)
	}

	watcher := &Watcher{
		NodeName: "demo",
		LastHash: hashScreen(current),
	}
	classifier := &stubClassifier{state: StateWaitingInput, reason: "prompt requires approval"}

	final := classifyQuietScreen(watcher, classifier, current, 5*time.Second)
	if final.State != StateWaitingInput {
		t.Fatalf("expected stable prompt to classify as waiting_input, got %s", final.State)
	}
	if classifier.calls != 1 {
		t.Fatalf("expected classifier to be called once, got %d", classifier.calls)
	}
}

func TestShouldNotifyWaitingInputBackoff(t *testing.T) {
	now := time.Now()
	watcher := &Watcher{
		State:        StateWaitingInput,
		LastNotifyAt: now.Add(-90 * time.Second),
		NotifyCount:  1,
	}

	if shouldNotify(StateWaitingInput, watcher, 2*time.Minute, now) {
		t.Fatalf("expected no reminder before initial backoff interval")
	}

	watcher.LastNotifyAt = now.Add(-2*time.Minute - time.Second)
	if !shouldNotify(StateWaitingInput, watcher, 2*time.Minute, now) {
		t.Fatalf("expected reminder after initial backoff interval")
	}

	watcher.NotifyCount = 2
	watcher.LastNotifyAt = now.Add(-3 * time.Minute)
	if shouldNotify(StateWaitingInput, watcher, 2*time.Minute, now) {
		t.Fatalf("expected second reminder to wait for doubled interval")
	}

	watcher.LastNotifyAt = now.Add(-4*time.Minute - time.Second)
	if !shouldNotify(StateWaitingInput, watcher, 2*time.Minute, now) {
		t.Fatalf("expected second reminder after doubled interval")
	}
}

func TestShouldNotifyResetsAfterLeavingWaitingInput(t *testing.T) {
	watcher := &Watcher{
		State:       StateWaitingInput,
		NotifyCount: 0,
	}

	if !shouldNotify(StateRunning, watcher, 2*time.Minute, time.Now()) {
		t.Fatalf("expected entering waiting_input to notify immediately")
	}
}

func TestApplyWatcherObservationResetsWaitingStateOnScreenChange(t *testing.T) {
	now := time.Now()
	watcher := &Watcher{
		NodeName:       "demo",
		State:          StateWaitingInput,
		StateEnteredAt: now.Add(-5 * time.Minute),
		StableSince:    now.Add(-5 * time.Minute),
		LastNotifyAt:   now.Add(-2 * time.Minute),
		NotifyCount:    2,
	}

	applyWatcherObservation(watcher, watcherObservation{
		now:        now,
		screen:     "processing next step",
		similarity: 0.72,
		stable:     false,
	}, ServiceConfig{ReminderInterval: 2 * time.Minute}, &stubClassifier{state: StateWaitingInput, reason: "unused"})

	if watcher.State != StateRunning {
		t.Fatalf("expected changed screen to reset to running, got %s", watcher.State)
	}
	if watcher.NotifyCount != 0 {
		t.Fatalf("expected notify count reset after leaving waiting_input, got %d", watcher.NotifyCount)
	}
	if watcher.StateEnteredAt != now {
		t.Fatalf("expected state transition time to update")
	}
}

func TestApplyWatcherObservationTreatsNewWaitingInputAsNewEvent(t *testing.T) {
	origNotify := sendWatcherNotification
	defer func() { sendWatcherNotification = origNotify }()
	sendWatcherNotification = func(nodeName, label, reason string) error { return nil }

	now := time.Now()
	watcher := &Watcher{
		NodeName:       "demo",
		Label:          "review",
		State:          StateRunning,
		StateEnteredAt: now.Add(-30 * time.Second),
		StableSince:    now.Add(-10 * time.Second),
		LastHash:       hashScreen("prompt 1"),
	}
	classifier := &stubClassifier{state: StateWaitingInput, reason: "approval required"}

	applyWatcherObservation(watcher, watcherObservation{
		now:        now,
		screen:     "prompt 2",
		similarity: 1,
		stable:     true,
		stableFor:  10 * time.Second,
	}, ServiceConfig{ReminderInterval: 2 * time.Minute}, classifier)

	if watcher.State != StateWaitingInput {
		t.Fatalf("expected stable prompt to enter waiting_input, got %s", watcher.State)
	}
	if watcher.WaitEventID != 1 {
		t.Fatalf("expected new waiting_input event id 1, got %d", watcher.WaitEventID)
	}
	if watcher.NotifyCount != 1 {
		t.Fatalf("expected new waiting_input event to notify immediately, got notify count %d", watcher.NotifyCount)
	}
	if watcher.LastNotifyAt != now {
		t.Fatalf("expected notify time to be recorded")
	}
}

func TestTransitionWatcherStateIncrementsWaitEventOnlyOnEntry(t *testing.T) {
	now := time.Now()
	watcher := &Watcher{State: StateRunning}

	transitionWatcherState(watcher, StateWaitingInput, "approval required", now)
	if watcher.WaitEventID != 1 {
		t.Fatalf("expected first wait event id to be 1, got %d", watcher.WaitEventID)
	}

	transitionWatcherState(watcher, StateWaitingInput, "approval required", now.Add(time.Second))
	if watcher.WaitEventID != 1 {
		t.Fatalf("expected repeated waiting_input state to keep same event id, got %d", watcher.WaitEventID)
	}

	transitionWatcherState(watcher, StateRunning, "screen changed", now.Add(2*time.Second))
	transitionWatcherState(watcher, StateWaitingInput, "approval required again", now.Add(3*time.Second))
	if watcher.WaitEventID != 2 {
		t.Fatalf("expected second wait event id to be 2, got %d", watcher.WaitEventID)
	}
}
