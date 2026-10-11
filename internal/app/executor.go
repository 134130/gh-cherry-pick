package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"

	"github.com/134130/gitkit/ghcli"
	"github.com/134130/gitkit/gitcmd"
)

type GitKitExecutor struct {
	git    gitcmd.Client
	gh     ghcli.Client
	output io.Writer
}

type synchronizedWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (w *synchronizedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.w.Write(p)
}

func NewGitKitExecutor(r gitcmd.Runner, output io.Writer) GitKitExecutor {
	if output != nil {
		output = &synchronizedWriter{w: output}
	}
	return GitKitExecutor{
		git:    gitcmd.Git(r),
		gh:     ghcli.New(r),
		output: output,
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
		err = e.applyPatch(ctx, plan)
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

func (e GitKitExecutor) applyPatch(ctx context.Context, plan Plan) error {
	diff, err := e.gh.PRDiff(ctx, plan.PR.Number)
	if err != nil {
		return fmt.Errorf("get PR patch: %w", err)
	}
	if _, err := e.git.Stream(e.output, e.output).WithStdin(bytes.NewReader(diff)).Run(ctx, "am", "-3"); err != nil {
		return e.applyFailure(ctx, plan, err, "Resolve the conflict, then run `git am --continue` or abort with `git am --abort`.")
	}
	return nil
}

func (e GitKitExecutor) applyCommits(ctx context.Context, plan Plan) error {
	if len(plan.PR.Commits) == 0 {
		return fmt.Errorf("PR #%d has no commits to cherry-pick", plan.PR.Number)
	}
	for _, commit := range plan.PR.Commits {
		if _, err := e.git.Stream(e.output, e.output).Run(ctx, "cherry-pick", commit.Oid); err != nil {
			return e.applyFailure(ctx, plan, err, "Resolve the conflict, then run `git cherry-pick --continue` or abort with `git cherry-pick --abort`.")
		}
	}
	return nil
}

func (e GitKitExecutor) applySquash(ctx context.Context, plan Plan) error {
	diff, err := e.gh.OutputBytes(ctx, "pr", "diff", strconv.Itoa(plan.PR.Number))
	if err != nil {
		return fmt.Errorf("get PR diff: %w", err)
	}
	if _, err := e.git.Stream(e.output, e.output).WithStdin(bytes.NewReader(diff)).Run(ctx, "apply", "--3way", "--index"); err != nil {
		return e.applyFailure(ctx, plan, err, "Resolve the conflict, stage the files, then commit manually.")
	}
	if _, err := e.git.Stream(e.output, e.output).Run(ctx, "commit", "-m", squashCommitMessage(plan.PR.Title, plan.PR.Number)); err != nil {
		return &ApplyFailure{
			Summary: fmt.Sprintf("Failed to commit PR #%d.", plan.PR.Number),
			Detail:  err.Error(),
			Err:     err,
		}
	}
	return nil
}

func (e GitKitExecutor) applyMerge(ctx context.Context, plan Plan) error {
	if plan.PR.MergeCommit.Sha == "" {
		return fmt.Errorf("PR #%d has no merge commit", plan.PR.Number)
	}
	if _, err := e.git.Stream(e.output, e.output).Run(ctx, "cherry-pick", "-m", "1", plan.PR.MergeCommit.Sha); err != nil {
		return e.applyFailure(ctx, plan, err, "Resolve the conflict, then run `git cherry-pick --continue` or abort with `git cherry-pick --abort`.")
	}
	return nil
}

func (e GitKitExecutor) applyFailure(ctx context.Context, plan Plan, err error, guidance string) error {
	conflictedFiles := e.conflictedFiles(ctx)
	if len(conflictedFiles) == 0 {
		return &ApplyFailure{
			Summary: fmt.Sprintf("Failed to apply PR #%d.", plan.PR.Number),
			Detail:  err.Error(),
			Err:     err,
		}
	}
	return &ApplyFailure{
		Summary:         fmt.Sprintf("Conflict while applying PR #%d.", plan.PR.Number),
		ConflictedFiles: conflictedFiles,
		Guidance:        guidance,
		Detail:          err.Error(),
		Err:             err,
	}
}

func (e GitKitExecutor) conflictedFiles(ctx context.Context) []string {
	out, err := e.git.Output(ctx, "diff", "--name-only", "--diff-filter=U")
	if err != nil || out == "" {
		return nil
	}
	return strings.Fields(out)
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

var _ Executor = GitKitExecutor{}
