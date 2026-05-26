package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/134130/gitkit/ghcli"
	"github.com/134130/gitkit/gitcmd"
)

type GitKitExecutor struct {
	git gitcmd.Client
	gh  ghcli.Client
}

func NewGitKitExecutor(r gitcmd.Runner) GitKitExecutor {
	return GitKitExecutor{
		git: gitcmd.Git(r),
		gh:  ghcli.New(r),
	}
}

func (e GitKitExecutor) Execute(ctx context.Context, plan Plan) (RunResult, error) {
	if plan.Blocked() {
		return RunResult{Plan: plan}, errors.New(plan.BlockedReason)
	}
	before, err := e.git.Output(ctx, "rev-parse", "HEAD")
	if err != nil {
		return RunResult{Plan: plan}, fmt.Errorf("get current HEAD: %w", err)
	}

	switch plan.Method {
	case MethodPatch:
		err = e.applyPatch(ctx, plan.PR.Number)
	case MethodCommits:
		err = e.applyCommits(ctx, plan)
	case MethodSquash:
		err = e.applySquash(ctx, plan)
	case MethodMerge:
		err = e.applyMerge(ctx, plan)
	default:
		err = fmt.Errorf("unsupported method: %s", plan.Method)
	}
	if err != nil {
		return RunResult{Plan: plan}, err
	}

	commits, err := e.createdCommits(ctx, before)
	if err != nil {
		return RunResult{Plan: plan}, fmt.Errorf("list created commits: %w", err)
	}
	return RunResult{Plan: plan, CommitSHAs: commits}, nil
}

func (e GitKitExecutor) applyPatch(ctx context.Context, prNumber int) error {
	diff, err := e.gh.PRDiff(ctx, prNumber)
	if err != nil {
		return fmt.Errorf("get PR patch: %w", err)
	}
	if _, err := e.git.WithStdin(bytes.NewReader(diff)).Run(ctx, "am", "-3"); err != nil {
		return fmt.Errorf("apply PR patch: %w\n\nresolve conflicts, then run `git am --continue` or abort with `git am --abort`", err)
	}
	return nil
}

func (e GitKitExecutor) applyCommits(ctx context.Context, plan Plan) error {
	if len(plan.PR.Commits) == 0 {
		return fmt.Errorf("PR #%d has no commits to cherry-pick", plan.PR.Number)
	}
	for _, commit := range plan.PR.Commits {
		if _, err := e.git.Run(ctx, "cherry-pick", commit.Oid); err != nil {
			return fmt.Errorf("cherry-pick commit %s: %w\n\nresolve conflicts, then run `git cherry-pick --continue` or abort with `git cherry-pick --abort`", shortSHA(commit.Oid), err)
		}
	}
	return nil
}

func (e GitKitExecutor) applySquash(ctx context.Context, plan Plan) error {
	diff, err := e.gh.OutputBytes(ctx, "pr", "diff", strconv.Itoa(plan.PR.Number))
	if err != nil {
		return fmt.Errorf("get PR diff: %w", err)
	}
	if _, err := e.git.WithStdin(bytes.NewReader(diff)).Run(ctx, "apply", "--3way", "--index"); err != nil {
		return fmt.Errorf("apply PR diff: %w\n\nresolve conflicts, then stage the files and commit manually", err)
	}
	if _, err := e.git.Run(ctx, "commit", "-m", squashCommitMessage(plan.PR.Title, plan.PR.Number)); err != nil {
		return fmt.Errorf("commit squashed PR changes: %w", err)
	}
	return nil
}

func (e GitKitExecutor) applyMerge(ctx context.Context, plan Plan) error {
	if plan.PR.MergeCommit.Sha == "" {
		return fmt.Errorf("PR #%d has no merge commit", plan.PR.Number)
	}
	if _, err := e.git.Run(ctx, "cherry-pick", "-m", "1", plan.PR.MergeCommit.Sha); err != nil {
		return fmt.Errorf("cherry-pick merge commit %s: %w\n\nresolve conflicts, then run `git cherry-pick --continue` or abort with `git cherry-pick --abort`", shortSHA(plan.PR.MergeCommit.Sha), err)
	}
	return nil
}

func (e GitKitExecutor) createdCommits(ctx context.Context, before string) ([]string, error) {
	out, err := e.git.Output(ctx, "rev-list", "--reverse", before+"..HEAD")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	return strings.Fields(out), nil
}

func shortSHA(sha string) string {
	if len(sha) <= 7 {
		return sha
	}
	return sha[:7]
}

var _ Executor = GitKitExecutor{}
