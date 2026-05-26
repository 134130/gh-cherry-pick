package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/134130/gh-cherry-pick/internal/app"
)

func TestRenderPlanJSON(t *testing.T) {
	plan := outputTestPlan()

	var out bytes.Buffer
	if err := RenderPlan(&out, plan, PlanOptions{Format: FormatJSON}); err != nil {
		t.Fatalf("RenderPlan returned error: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out.String())
	}
	if got["method"] != "patch" || got["requestedMethod"] != "auto" {
		t.Fatalf("unexpected methods: %#v", got)
	}
	commands, ok := got["commands"].([]any)
	if !ok || len(commands) != 2 {
		t.Fatalf("expected 2 commands, got %#v", got["commands"])
	}
}

func TestRenderBlockedPlanHuman(t *testing.T) {
	plan := outputTestPlan()
	plan.BlockedReason = "working tree or index is dirty"
	plan.Method = ""
	plan.Commands = nil

	var out bytes.Buffer
	if err := RenderPlan(&out, plan, PlanOptions{Format: FormatHuman}); err != nil {
		t.Fatalf("RenderPlan returned error: %v", err)
	}

	text := out.String()
	for _, want := range []string{
		"Method: auto",
		"Blocked: working tree or index is dirty",
		"Commands: none",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected output to contain %q, got %q", want, text)
		}
	}
}

func TestRenderRunResultJSON(t *testing.T) {
	result := app.RunResult{
		Plan:       outputTestPlan(),
		CommitSHAs: []string{"abc123"},
	}

	var out bytes.Buffer
	if err := RenderRunResult(&out, result, FormatJSON); err != nil {
		t.Fatalf("RenderRunResult returned error: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out.String())
	}
	commits, ok := got["commitShas"].([]any)
	if !ok || len(commits) != 1 || commits[0] != "abc123" {
		t.Fatalf("unexpected commitShas: %#v", got["commitShas"])
	}
}

func outputTestPlan() app.Plan {
	plan := app.Plan{
		RequestedMethod: app.MethodAuto,
		Method:          app.MethodPatch,
		MergeShape:      "rebase",
		Destination: app.Destination{
			Branch: "feature",
			Head:   "abcdef123456",
		},
		Commands: []app.Command{
			{Program: "gh", Args: []string{"pr", "diff", "123", "--patch"}},
			{Program: "git", Args: []string{"am", "-3"}},
		},
	}
	plan.PR.Number = 123
	plan.PR.Title = "Fix bug"
	plan.PR.State = "MERGED"
	return plan
}
