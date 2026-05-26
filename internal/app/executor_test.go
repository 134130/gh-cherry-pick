package app

import (
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

	result, err := NewGitKitExecutor(runner).Execute(context.Background(), executorPlan(MethodPatch))
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

	_, err := NewGitKitExecutor(runner).Execute(context.Background(), plan)
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

	_, err := NewGitKitExecutor(runner).Execute(context.Background(), executorPlan(MethodMerge))
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

	_, err := NewGitKitExecutor(runner).Execute(context.Background(), plan)
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
	runner.errors["git am -3"] = exitError("git am -3", "conflict")

	_, err := NewGitKitExecutor(runner).Execute(context.Background(), executorPlan(MethodPatch))
	if err == nil {
		t.Fatalf("expected error")
	}
	for _, want := range []string{"git am --continue", "git am --abort"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected error to contain %q, got %v", want, err)
		}
	}
}

type fakeRunner struct {
	commands []string
	outputs  map[string]string
	errors   map[string]error
}

func newFakeRunner() *fakeRunner {
	return &fakeRunner{
		outputs: map[string]string{},
		errors:  map[string]error{},
	}
}

func (r *fakeRunner) Run(_ context.Context, cmd gitcmd.Command) (gitcmd.Result, error) {
	line := cmd.String()
	r.commands = append(r.commands, line)
	if err := r.errors[line]; err != nil {
		return gitcmd.Result{Command: cmd, ExitCode: 1}, err
	}
	return gitcmd.Result{
		Command: cmd,
		Stdout:  []byte(r.outputs[line]),
	}, nil
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
