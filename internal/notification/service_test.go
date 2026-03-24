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
