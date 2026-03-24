package notification

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"orion/internal/ai"
)

type SnapshotClassifier interface {
	Classify(nodeName, screen string, stableFor time.Duration) (Classification, error)
}

type LLMClassifier struct {
	client *ai.Client
}

func NewLLMClassifier() (*LLMClassifier, error) {
	client, err := ai.NewClient()
	if err != nil {
		return nil, err
	}
	return &LLMClassifier{client: client}, nil
}

func normalizeScreen(screen string) string {
	lines := strings.Split(screen, "\n")
	for i, line := range lines {
		lines[i] = strings.ToLower(strings.TrimSpace(line))
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func hashScreen(screen string) string {
	sum := sha256.Sum256([]byte(normalizeScreen(screen)))
	return fmt.Sprintf("%x", sum[:])
}

func tail(screen string, lines int) string {
	if lines <= 0 {
		return screen
	}
	parts := strings.Split(screen, "\n")
	if len(parts) <= lines {
		return screen
	}
	return strings.Join(parts[len(parts)-lines:], "\n")
}

func screenSimilarity(previous, current string) float64 {
	previous = normalizeScreen(previous)
	current = normalizeScreen(current)

	switch {
	case previous == "" && current == "":
		return 1
	case previous == "" || current == "":
		return 0
	case previous == current:
		return 1
	}

	distance := levenshteinDistance(previous, current)
	maxLen := max(len(previous), len(current))
	if maxLen == 0 {
		return 1
	}

	similarity := 1 - float64(distance)/float64(maxLen)
	if similarity < 0 {
		return 0
	}
	return math.Min(1, similarity)
}

func levenshteinDistance(a, b string) int {
	if a == b {
		return 0
	}

	ar := []rune(a)
	br := []rune(b)
	if len(ar) == 0 {
		return len(br)
	}
	if len(br) == 0 {
		return len(ar)
	}

	prev := make([]int, len(br)+1)
	curr := make([]int, len(br)+1)
	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= len(ar); i++ {
		curr[0] = i
		for j := 1; j <= len(br); j++ {
			cost := 0
			if ar[i-1] != br[j-1] {
				cost = 1
			}
			curr[j] = min(
				prev[j]+1,
				curr[j-1]+1,
				prev[j-1]+cost,
			)
		}
		prev, curr = curr, prev
	}

	return prev[len(br)]
}

func HeuristicClassify(previousScreen, currentScreen string, similarityThreshold float64) Classification {
	similarity := screenSimilarity(previousScreen, currentScreen)
	if similarity >= similarityThreshold {
		return Classification{State: StateQuietCandidate, Reason: fmt.Sprintf("stable_screen_similarity=%.4f", similarity)}
	}
	return Classification{State: StateRunning, Reason: fmt.Sprintf("screen_changed_similarity=%.4f", similarity)}
}

func min(values ...int) int {
	result := values[0]
	for _, value := range values[1:] {
		if value < result {
			result = value
		}
	}
	return result
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (c *LLMClassifier) Classify(nodeName, screen string, stableFor time.Duration) (Classification, error) {
	systemPrompt := `You are classifying a stable terminal snapshot from an interactive coding agent.

Return JSON only:
{
  "state": "waiting_input|completed_idle|still_working|unknown",
  "reason": "short reason"
}

Definitions:
- waiting_input: the screen is clearly asking the human to confirm, choose, approve, or provide input.
- completed_idle: the agent appears finished and is not asking for more input.
- still_working: the agent appears to still be processing or actively working.
- unknown: not enough evidence.

Be conservative. Only choose waiting_input if the terminal clearly requests user action.`

	userPrompt := fmt.Sprintf("Node: %s\nStable for: %s\n\nTerminal snapshot:\n%s", nodeName, stableFor.Round(time.Second), tail(normalizeScreen(screen), 80))
	content, err := c.client.GenerateText(systemPrompt, userPrompt, 0, 128)
	if err != nil {
		return Classification{}, err
	}

	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var out struct {
		State  string `json:"state"`
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(content), &out); err != nil {
		return Classification{}, fmt.Errorf("failed to parse LLM classification: %w", err)
	}

	switch out.State {
	case "waiting_input":
		return Classification{State: StateWaitingInput, Reason: strings.TrimSpace(out.Reason)}, nil
	case "completed_idle":
		return Classification{State: StateCompletedIdle, Reason: strings.TrimSpace(out.Reason)}, nil
	case "still_working":
		return Classification{State: StateRunning, Reason: strings.TrimSpace(out.Reason)}, nil
	case "unknown":
		return Classification{State: StateUnknown, Reason: strings.TrimSpace(out.Reason)}, nil
	default:
		return Classification{}, fmt.Errorf("unexpected LLM state %q", out.State)
	}
}
