package app

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/134130/gh-cherry-pick/gitobj"
	"github.com/134130/gitkit/gitcmd"
)

func TestExecutorPatchMethodRunsPRDiffAndGitAM(t *testing.T) {
	runner := newFakeRunner()
	runner.outputs["git rev-parse HEAD"] = "base\n"
	runner.outputs["gh pr diff 123 --patch"] = "patch"
	runner.outputs["git rev-list --reverse base..HEAD"] = "newsha\n"

	result, err := NewGitKitExecutor(runner, nil).Execute(context.Background(), executorPlan(MethodPatch))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	wantCommands := []string{
		"git rev-parse HEAD",
		"gh pr diff 123 --patch",
		"git am -3",
		"git rev-list --reverse base..HEAD",
	}
	if !reflect.DeepEqual(runner.commands, wantCommands) {
		t.Fatalf("commands mismatch\nwant: %#v\n got: %#v", wantCommands, runner.commands)
	}
	if !reflect.DeepEqual(result.CommitSHAs, []string{"newsha"}) {
		t.Fatalf("commit SHAs mismatch: %#v", result.CommitSHAs)
	}
}

func TestExecutorCommitsMethodRunsCherryPicksInOrder(t *testing.T) {
	runner := newFakeRunner()
	runner.outputs["git rev-parse HEAD"] = "base\n"
	runner.outputs["git rev-list --reverse base..HEAD"] = "one\ntwo\n"

	plan := executorPlan(MethodCommits)
	plan.PR.Commits = []struct {
		Oid string `json:"oid"`
	}{{Oid: "one"}, {Oid: "two"}}

	_, err := NewGitKitExecutor(runner, nil).Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	wantCommands := []string{
		"git rev-parse HEAD",
		"git cherry-pick one",
		"git cherry-pick two",
		"git rev-list --reverse base..HEAD",
	}
	if !reflect.DeepEqual(runner.commands, wantCommands) {
		t.Fatalf("commands mismatch\nwant: %#v\n got: %#v", wantCommands, runner.commands)
	}
}

func TestExecutorMergeMethodUsesMainlineParent(t *testing.T) {
	runner := newFakeRunner()
	runner.outputs["git rev-parse HEAD"] = "base\n"
	runner.outputs["git rev-list --reverse base..HEAD"] = "merge\n"

	_, err := NewGitKitExecutor(runner, nil).Execute(context.Background(), executorPlan(MethodMerge))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if got, want := runner.commands[1], "git cherry-pick -m 1 mergesha"; got != want {
		t.Fatalf("command mismatch: want %q, got %q", want, got)
	}
}

func TestExecutorBlockedPlanDoesNotRunCommands(t *testing.T) {
	runner := newFakeRunner()
	plan := executorPlan(MethodPatch)
	plan.BlockedReason = "working tree or index is dirty"

	_, err := NewGitKitExecutor(runner, nil).Execute(context.Background(), plan)
	if err == nil {
		t.Fatalf("expected error")
	}
	if len(runner.commands) != 0 {
		t.Fatalf("expected no commands, got %#v", runner.commands)
	}
}

func TestExecutorPatchConflictIncludesNativeGuidance(t *testing.T) {
	runner := newFakeRunner()
	runner.outputs["git rev-parse HEAD"] = "base\n"
	runner.outputs["gh pr diff 123 --patch"] = "patch"
	runner.outputs["git diff --name-only --diff-filter=U"] = "file.txt\n"
	runner.errors["git am -3"] = exitError("git am -3", "conflict")

	_, err := NewGitKitExecutor(runner, nil).Execute(context.Background(), executorPlan(MethodPatch))
	if err == nil {
		t.Fatalf("expected error")
	}
	for _, want := range []string{"git am --continue", "git am --abort"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected error to contain %q, got %v", want, err)
		}
	}
}

func TestExecutorStreamsMutationOutputWithoutStreamingQueries(t *testing.T) {
	runner := newFakeRunner()
	runner.outputs["git rev-parse HEAD"] = "base\n"
	runner.outputs["git cherry-pick one"] = "hook stdout\n"
	runner.stderrs["git cherry-pick one"] = "hook stderr\n"

	plan := executorPlan(MethodCommits)
	plan.PR.Commits = []struct {
		Oid string `json:"oid"`
	}{{Oid: "one"}}

	var output bytes.Buffer
	_, err := NewGitKitExecutor(runner, &output).Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if got, want := output.String(), "hook stdout\nhook stderr\n"; got != want {
		t.Fatalf("streamed output mismatch: want %q, got %q", want, got)
	}
}

func TestExecutorDoesNotReportHookFailureAsConflict(t *testing.T) {
	runner := newFakeRunner()
	runner.outputs["git rev-parse HEAD"] = "base\n"
	runner.outputs["gh pr diff 123"] = "patch"
	runner.stderrs["git commit -m Fix bug (#123)"] = "lefthook failed\n"
	runner.errors["git commit -m Fix bug (#123)"] = exitError("git commit -m Fix bug (#123)", "lefthook failed")

	var output bytes.Buffer
	_, err := NewGitKitExecutor(runner, &output).Execute(context.Background(), executorPlan(MethodSquash))
	if err == nil {
		t.Fatalf("expected error")
	}
	if strings.Contains(err.Error(), "Conflict") || strings.Contains(err.Error(), "--continue") {
		t.Fatalf("hook failure was reported as conflict: %v", err)
	}
	if !strings.Contains(err.Error(), "lefthook failed") || !strings.Contains(output.String(), "lefthook failed") {
		t.Fatalf("expected hook failure detail, err=%q output=%q", err, output.String())
	}
}

type fakeRunner struct {
	commands []string
	outputs  map[string]string
	stderrs  map[string]string
	errors   map[string]error
}

func newFakeRunner() *fakeRunner {
	return &fakeRunner{
		outputs: map[string]string{},
		stderrs: map[string]string{},
		errors:  map[string]error{},
	}
}

func (r *fakeRunner) Run(_ context.Context, cmd gitcmd.Command) (gitcmd.Result, error) {
	line := cmd.String()
	r.commands = append(r.commands, line)
	result := gitcmd.Result{
		Command: cmd,
		Stdout:  []byte(r.outputs[line]),
		Stderr:  []byte(r.stderrs[line]),
	}
	if cmd.Stdout != nil {
		_, _ = cmd.Stdout.Write(result.Stdout)
	}
	if cmd.Stderr != nil {
		_, _ = cmd.Stderr.Write(result.Stderr)
	}
	if err := r.errors[line]; err != nil {
		result.ExitCode = 1
		return result, err
	}
	return result, nil
}

func (r *fakeRunner) Start(context.Context, gitcmd.Command) (gitcmd.Process, error) {
	return nil, errors.New("Start is not implemented in fakeRunner")
}

func executorPlan(method ApplyMethod) Plan {
	var pr gitobj.PullRequest
	pr.Number = 123
	pr.Title = "Fix bug"
	pr.State = gitobj.PullRequestStateMerged
	pr.MergeCommit.Sha = "mergesha"
	return Plan{
		PR:              pr,
		RequestedMethod: method,
		Method:          method,
	}
}

func exitError(command, stderr string) error {
	cmd := gitcmd.GitCommand(strings.Fields(strings.TrimPrefix(command, "git "))...)
	return &gitcmd.ExitError{
		Result: gitcmd.Result{
			Command:  cmd,
			Stderr:   []byte(stderr),
			ExitCode: 1,
		},
		Err: errors.New("exit status 1"),
	}
}
