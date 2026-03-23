package cmd

import (
	"reflect"
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

	got := buildNodeSelectionItems(nodes, []string{"alpha", "beta"})
	want := []nodeSelectionItem{
		{Name: "alpha", Label: "review auth flow"},
		{Name: "beta", Label: "-"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected selection items:\nwant: %#v\ngot:  %#v", want, got)
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
