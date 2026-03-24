package notification

import "testing"

func TestScreenSimilarityIdenticalPrompt(t *testing.T) {
	previous := `
Would you like to run the following command?
1. Yes, proceed
2. No
Press enter to confirm or esc to cancel
`
	current := `
would you like to run the following command?
1. yes, proceed
2. no
press enter to confirm or esc to cancel
`

	similarity := screenSimilarity(previous, current)
	if similarity != 1 {
		t.Fatalf("expected identical normalized screens to have similarity 1, got %f", similarity)
	}
}

func TestScreenSimilarityAllowsMinorDiffs(t *testing.T) {
	previous := `
would you like to run the following command?
1. yes, proceed
2. no
3. show details
press enter to confirm or esc to cancel
workspace: /tmp/demo-node
reason: command needs approval before execution
`
	current := `
would you like to run the following command?
1. yes, proceed
2. no
3. show details
press enter to confirm or esc to cancel
workspace: /tmp/demo-node
reason: command needs approval before execution.
`

	similarity := screenSimilarity(previous, current)
	if similarity < 0.99 {
		t.Fatalf("expected minor diff similarity >= 0.99, got %f", similarity)
	}
}

func TestHeuristicClassifyStableScreen(t *testing.T) {
	previous := `
would you like to run the following command?
1. yes, proceed
2. no
press enter to confirm or esc to cancel
`
	current := `
would you like to run the following command?
1. yes, proceed
2. no
press enter to confirm or esc to cancel
`

	classification := HeuristicClassify(previous, current, 0.99)
	if classification.State != StateQuietCandidate {
		t.Fatalf("expected quiet_candidate, got %s", classification.State)
	}
}

func TestHeuristicClassifyChangedScreen(t *testing.T) {
	previous := `
processing file 1/10
`
	current := `
processing file 9/10
`

	classification := HeuristicClassify(previous, current, 0.99)
	if classification.State != StateRunning {
		t.Fatalf("expected running, got %s", classification.State)
	}
}

func TestHeuristicClassifyFirstSnapshot(t *testing.T) {
	classification := HeuristicClassify("", "waiting for input", 0.99)
	if classification.State != StateRunning {
		t.Fatalf("expected first snapshot to remain running, got %s", classification.State)
	}
}
