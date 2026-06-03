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

func TestRenderPlanHumanOmitsCommandsByDefault(t *testing.T) {
	plan := outputTestPlan()

	var out bytes.Buffer
	if err := RenderPlan(&out, plan, PlanOptions{Format: FormatHuman}); err != nil {
		t.Fatalf("RenderPlan returned error: %v", err)
	}

	text := out.String()
	for _, want := range []string{
		"Apply PR #123: Fix bug",
		"Destination: feature @ abcdef1",
		"Strategy: patch, auto-detected from GitHub rebase merge",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected output to contain %q, got %q", want, text)
		}
	}
	if strings.Contains(text, "Commands:") {
		t.Fatalf("did not expect commands by default, got %q", text)
	}
}

func TestRenderPlanHumanVerboseIncludesCommands(t *testing.T) {
	plan := outputTestPlan()

	var out bytes.Buffer
	if err := RenderPlan(&out, plan, PlanOptions{Format: FormatHuman, Verbose: true}); err != nil {
		t.Fatalf("RenderPlan returned error: %v", err)
	}

	text := out.String()
	for _, want := range []string{
		"Commands:",
		"gh pr diff 123 --patch",
		"git am -3",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected output to contain %q, got %q", want, text)
		}
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
		"Cannot apply PR #123: working tree or index is dirty.",
		"Commit, stash, or discard local changes, then retry.",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected output to contain %q, got %q", want, text)
		}
	}
	if strings.Contains(text, "Commands:") {
		t.Fatalf("did not expect commands for blocked plan, got %q", text)
	}
}

func TestRenderRunResultJSON(t *testing.T) {
	result := app.RunResult{
		Plan:       outputTestPlan(),
		CommitSHAs: []string{"abc123"},
	}

	var out bytes.Buffer
	if err := RenderRunResult(&out, result, RunResultOptions{Format: FormatJSON}); err != nil {
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

func TestRenderRunResultHumanUsesShortSHAs(t *testing.T) {
	result := app.RunResult{
		Plan:       outputTestPlan(),
		CommitSHAs: []string{"abcdef1234567890"},
	}

	var out bytes.Buffer
	if err := RenderRunResult(&out, result, RunResultOptions{Format: FormatHuman}); err != nil {
		t.Fatalf("RenderRunResult returned error: %v", err)
	}

	text := out.String()
	for _, want := range []string{
		"Applied PR #123 using patch.",
		"Created 1 commit:",
		"  abcdef1",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected output to contain %q, got %q", want, text)
		}
	}
	if strings.Contains(text, "abcdef1234567890") {
		t.Fatalf("did not expect full SHA in human output, got %q", text)
	}
}

func TestRenderExecutionErrorHuman(t *testing.T) {
	err := &app.ApplyFailure{
		Summary:         "Conflict while applying PR #123.",
		ConflictedFiles: []string{"file.txt"},
		Guidance:        "Resolve the conflict, stage the files, then commit manually.",
		Detail:          "raw git detail",
	}

	var out bytes.Buffer
	rendered, renderErr := RenderExecutionError(&out, err, ErrorOptions{})
	if renderErr != nil {
		t.Fatalf("RenderExecutionError returned error: %v", renderErr)
	}
	if !rendered {
		t.Fatalf("expected error to be rendered")
	}

	text := out.String()
	for _, want := range []string{
		"Conflict while applying PR #123.",
		"Conflicted files:",
		"  file.txt",
		"Resolve the conflict, stage the files, then commit manually.",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected output to contain %q, got %q", want, text)
		}
	}
	if strings.Contains(text, "raw git detail") {
		t.Fatalf("did not expect detail without verbose, got %q", text)
	}
}

func TestRenderExecutionErrorVerboseIncludesDetail(t *testing.T) {
	err := &app.ApplyFailure{
		Summary:  "Conflict while applying PR #123.",
		Guidance: "Resolve the conflict, stage the files, then commit manually.",
		Detail:   "raw git detail",
	}

	var out bytes.Buffer
	rendered, renderErr := RenderExecutionError(&out, err, ErrorOptions{Verbose: true})
	if renderErr != nil {
		t.Fatalf("RenderExecutionError returned error: %v", renderErr)
	}
	if !rendered {
		t.Fatalf("expected error to be rendered")
	}

	text := out.String()
	if !strings.Contains(text, "Details:") || !strings.Contains(text, "raw git detail") {
		t.Fatalf("expected verbose detail, got %q", text)
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
