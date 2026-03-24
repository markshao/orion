package cmd

import (
	"testing"

	"orion/internal/types"
)

func TestBuildNodeSelectionItemsUsesNameAndLabelOnly(t *testing.T) {
	nodes := map[string]types.Node{
		"alpha": {
			Label:         "review auth flow",
			LogicalBranch: "feature/auth",
			Status:        types.StatusReadyToPush,
		},
		"beta": {
			Label:         "",
			LogicalBranch: "feature/billing",
			Status:        types.StatusFail,
		},
	}

	got := buildNodeSelectionItems(nodes, []string{"alpha", "beta"}, map[string]bool{})
	if len(got) != 2 {
		t.Fatalf("expected 2 selection items, got %d", len(got))
	}

	if got[0].Name != "alpha" || got[0].Label != "review auth flow" {
		t.Fatalf("unexpected first item: %#v", got[0])
	}
	if got[1].Name != "beta" || got[1].Label != "-" {
		t.Fatalf("unexpected second item: %#v", got[1])
	}
	if got[0].NameColumn != "alpha" {
		t.Fatalf("expected alpha name column to be unpadded, got %q", got[0].NameColumn)
	}
	if got[1].NameColumn != "beta " {
		t.Fatalf("expected beta name column to be padded, got %q", got[1].NameColumn)
	}
	if got[0].Row != "alpha  review auth flow" {
		t.Fatalf("unexpected alpha row: %q", got[0].Row)
	}
	if got[1].Row != "beta   -" {
		t.Fatalf("unexpected beta row: %q", got[1].Row)
	}
}

func TestNormalizeNodeLabel(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "trim", input: "  implement parser  ", want: "implement parser"},
		{name: "empty", input: "", want: "-"},
		{name: "whitespace", input: "   ", want: "-"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeNodeLabel(tc.input); got != tc.want {
				t.Fatalf("normalizeNodeLabel(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestBuildNodeSelectionLabelPendingWait(t *testing.T) {
	if got := buildNodeSelectionLabel("review auth flow", true); got != "[wait] review auth flow" {
		t.Fatalf("unexpected pending wait label: %q", got)
	}
	if got := buildNodeSelectionLabel("", true); got != "[wait] -" {
		t.Fatalf("unexpected pending wait label for empty input: %q", got)
	}
}

func TestBuildNodeSelectionItemsAlignsWideCharacterNames(t *testing.T) {
	nodes := map[string]types.Node{
		"alpha": {
			Label: "review auth flow",
		},
		"重构": {
			Label: "更新状态逻辑",
		},
	}

	got := buildNodeSelectionItems(nodes, []string{"alpha", "重构"}, map[string]bool{})
	if len(got) != 2 {
		t.Fatalf("expected 2 selection items, got %d", len(got))
	}

	if displayWidth(got[0].NameColumn) != displayWidth(got[1].NameColumn) {
		t.Fatalf("expected aligned name columns, got %q (%d) and %q (%d)",
			got[0].NameColumn, displayWidth(got[0].NameColumn),
			got[1].NameColumn, displayWidth(got[1].NameColumn),
		)
	}
}

func TestBuildNodeSelectionItemsPrioritizesPendingWait(t *testing.T) {
	nodes := map[string]types.Node{
		"alpha": {Label: "review auth flow"},
		"beta":  {Label: "fix tests"},
	}

	got := buildNodeSelectionItems(nodes, []string{"alpha", "beta"}, map[string]bool{"beta": true})
	if len(got) != 2 {
		t.Fatalf("expected 2 selection items, got %d", len(got))
	}
	if got[0].Name != "beta" || !got[0].PendingWait {
		t.Fatalf("expected pending wait node first, got %#v", got[0])
	}
	if got[0].Label != "[wait] fix tests" {
		t.Fatalf("unexpected pending wait row label: %q", got[0].Label)
	}
}

func TestDisplayWidth(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  int
	}{
		{name: "ascii", input: "alpha", want: 5},
		{name: "han", input: "重构", want: 4},
		{name: "mixed", input: "alpha 重构", want: 10},
		{name: "combining", input: "e\u0301", want: 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := displayWidth(tc.input); got != tc.want {
				t.Fatalf("displayWidth(%q) = %d, want %d", tc.input, got, tc.want)
			}
		})
	}
}
