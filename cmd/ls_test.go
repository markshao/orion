package cmd

import (
	"strings"
	"testing"
	"time"

	"orion/internal/types"

	"github.com/fatih/color"
)

func TestSortedNodeNamesFiltersAndSorts(t *testing.T) {
	nodes := map[string]types.Node{
		"zeta":  {CreatedBy: "user"},
		"alpha": {CreatedBy: "workflow-1"},
		"beta":  {CreatedBy: "user"},
	}

	got := sortedNodeNames(nodes, false)
	if len(got) != 2 {
		t.Fatalf("expected 2 user nodes, got %d", len(got))
	}
	if got[0] != "beta" || got[1] != "zeta" {
		t.Fatalf("unexpected order: %v", got)
	}

	gotAll := sortedNodeNames(nodes, true)
	if len(gotAll) != 3 {
		t.Fatalf("expected 3 nodes with --all, got %d", len(gotAll))
	}
	if gotAll[0] != "alpha" || gotAll[1] != "beta" || gotAll[2] != "zeta" {
		t.Fatalf("unexpected all-node order: %v", gotAll)
	}
}

func TestRenderNodeCardIncludesStableFields(t *testing.T) {
	prev := color.NoColor
	color.NoColor = true
	defer func() { color.NoColor = prev }()

	node := types.Node{
		LogicalBranch: "feature/bare-repo-concept",
		CreatedAt:     time.Date(2026, 3, 23, 9, 52, 0, 0, time.UTC),
	}

	got := renderNodeCardContent("bare-repo-dev", node, "-")

	expectedSnippets := []string{
		"bare-repo-dev",
		"  git       WORKING",
		"  branch    feature/bare-repo-concept",
		"  base-sync -",
		"  label     -",
		"  created   2026-03-23 09:52",
	}
	for _, snippet := range expectedSnippets {
		if !strings.Contains(got, snippet) {
			t.Fatalf("expected output to contain %q, got:\n%s", snippet, got)
		}
	}
}

func TestRenderNodeListEmpty(t *testing.T) {
	got := renderNodeList("", map[string]types.Node{}, false)
	if got != "No nodes found.\n" {
		t.Fatalf("unexpected empty output: %q", got)
	}
}
