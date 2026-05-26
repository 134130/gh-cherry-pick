package cli

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/134130/gh-cherry-pick/internal/app"
)

func TestRunDryRunRendersPlanWithoutExecuting(t *testing.T) {
	restore := stubRunHooks(t)
	defer restore()

	buildPlanFunc = func(context.Context, Config) (app.Plan, error) {
		return testPlan(), nil
	}
	executePlanFunc = func(context.Context, Config, app.Plan) (app.RunResult, error) {
		t.Fatalf("executePlanFunc should not be called")
		return app.RunResult{}, nil
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := RunWithIO(context.Background(), []string{"123", "--dry-run", "--no-color"}, strings.NewReader(""), &stdout, &stderr)
	if err != nil {
		t.Fatalf("RunWithIO returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "PR #123") || !strings.Contains(stdout.String(), "Commands:") {
		t.Fatalf("expected plan output, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr, got %q", stderr.String())
	}
}

func TestRunApplyRejectsNonTerminalWithoutYesBeforeBuildingPlan(t *testing.T) {
	restore := stubRunHooks(t)
	defer restore()

	var built bool
	buildPlanFunc = func(context.Context, Config) (app.Plan, error) {
		built = true
		return app.Plan{}, nil
	}

	err := RunWithIO(context.Background(), []string{"123"}, strings.NewReader("yes\n"), &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "requires --yes") {
		t.Fatalf("unexpected error: %v", err)
	}
	if built {
		t.Fatalf("did not expect plan to be built")
	}
}

func TestRunApplyConfirmsThenExecutes(t *testing.T) {
	restore := stubRunHooks(t)
	defer restore()
	restoreTerminal := stubTerminalInput(t, true)
	defer restoreTerminal()

	buildPlanFunc = func(context.Context, Config) (app.Plan, error) {
		return testPlan(), nil
	}
	var executed bool
	executePlanFunc = func(_ context.Context, cfg Config, plan app.Plan) (app.RunResult, error) {
		executed = true
		if cfg.PRNumber != 123 {
			t.Fatalf("PR number mismatch: %d", cfg.PRNumber)
		}
		return app.RunResult{Plan: plan, CommitSHAs: []string{"abc123"}}, nil
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := RunWithIO(context.Background(), []string{"123", "--no-color"}, strings.NewReader("yes\n"), &stdout, &stderr)
	if err != nil {
		t.Fatalf("RunWithIO returned error: %v", err)
	}
	if !executed {
		t.Fatalf("expected execution")
	}
	if !strings.Contains(stderr.String(), "Apply this PR? (y/N): ") {
		t.Fatalf("expected confirmation prompt, got %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "Applied PR #123") {
		t.Fatalf("expected result output, got %q", stdout.String())
	}
}

func TestRunApplyCancelDoesNotExecute(t *testing.T) {
	restore := stubRunHooks(t)
	defer restore()
	restoreTerminal := stubTerminalInput(t, true)
	defer restoreTerminal()

	buildPlanFunc = func(context.Context, Config) (app.Plan, error) {
		return testPlan(), nil
	}
	var executed bool
	executePlanFunc = func(context.Context, Config, app.Plan) (app.RunResult, error) {
		executed = true
		return app.RunResult{}, nil
	}

	err := RunWithIO(context.Background(), []string{"123"}, strings.NewReader("\n"), &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("RunWithIO returned error: %v", err)
	}
	if executed {
		t.Fatalf("did not expect execution")
	}
}

func TestRunApplyYesJSONOutputsOnlyResultJSON(t *testing.T) {
	restore := stubRunHooks(t)
	defer restore()

	buildPlanFunc = func(context.Context, Config) (app.Plan, error) {
		return testPlan(), nil
	}
	executePlanFunc = func(_ context.Context, _ Config, plan app.Plan) (app.RunResult, error) {
		return app.RunResult{Plan: plan, CommitSHAs: []string{"abc123"}}, nil
	}

	var stdout bytes.Buffer
	err := RunWithIO(context.Background(), []string{"123", "--yes", "--json"}, strings.NewReader(""), &stdout, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("RunWithIO returned error: %v", err)
	}
	out := stdout.String()
	if strings.Count(out, "\n") != 1 || !strings.Contains(out, `"commitShas":["abc123"]`) {
		t.Fatalf("expected single JSON object, got %q", out)
	}
}

func TestRunApplyJSONRequiresYes(t *testing.T) {
	restore := stubRunHooks(t)
	defer restore()
	restoreTerminal := stubTerminalInput(t, true)
	defer restoreTerminal()

	err := RunWithIO(context.Background(), []string{"123", "--json"}, strings.NewReader("yes\n"), &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "JSON output requires --yes") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseValidatesMethod(t *testing.T) {
	if _, err := Parse([]string{"123", "--method", "bogus"}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestParseRequiresPRNumber(t *testing.T) {
	if _, err := Parse([]string{}); err == nil {
		t.Fatalf("expected error")
	}
}

func stubRunHooks(t *testing.T) func() {
	t.Helper()
	oldBuild := buildPlanFunc
	oldExecute := executePlanFunc
	oldTerminal := isTerminalReaderFunc
	return func() {
		buildPlanFunc = oldBuild
		executePlanFunc = oldExecute
		isTerminalReaderFunc = oldTerminal
	}
}

func stubTerminalInput(t *testing.T, isTerminal bool) func() {
	t.Helper()
	old := isTerminalReaderFunc
	isTerminalReaderFunc = func(_ io.Reader) bool {
		return isTerminal
	}
	return func() {
		isTerminalReaderFunc = old
	}
}

func testPlan() app.Plan {
	pr := app.Plan{
		RequestedMethod: app.MethodAuto,
		Method:          app.MethodPatch,
		Destination: app.Destination{
			Branch: "feature",
			Head:   "abcdef123456",
		},
		Commands: []app.Command{
			{Program: "gh", Args: []string{"pr", "diff", "123", "--patch"}},
			{Program: "git", Args: []string{"am", "-3"}},
		},
	}
	pr.PR.Number = 123
	pr.PR.Title = "Fix bug"
	pr.PR.State = "MERGED"
	return pr
}
