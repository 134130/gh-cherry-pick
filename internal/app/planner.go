package app

import (
	"context"
	"fmt"
	"strconv"
)

type Planner struct {
	store RepositoryStore
}

func NewPlanner(store RepositoryStore) Planner {
	return Planner{store: store}
}

func (p Planner) BuildPlan(ctx context.Context, opts PlanOptions) (Plan, error) {
	opts = opts.normalized()
	if opts.PRNumber < 1 {
		return Plan{}, fmt.Errorf("PR number must be greater than 0")
	}
	if err := opts.Method.Validate(); err != nil {
		return Plan{}, err
	}

	branch, err := p.store.CurrentBranch(ctx)
	if err != nil {
		return Plan{}, fmt.Errorf("get current branch: %w", err)
	}
	head, err := p.store.HeadSHA(ctx)
	if err != nil {
		return Plan{}, fmt.Errorf("get current HEAD: %w", err)
	}
	state, err := p.store.State(ctx)
	if err != nil {
		return Plan{}, fmt.Errorf("get repository state: %w", err)
	}
	pr, err := p.store.PullRequest(ctx, opts.PRNumber)
	if err != nil {
		return Plan{}, fmt.Errorf("get pull request #%d: %w", opts.PRNumber, err)
	}

	plan := Plan{
		PR: pr,
		Destination: Destination{
			Branch: branch,
			Head:   head,
		},
		RequestedMethod: opts.Method,
		State:           state,
	}

	if state.Dirty {
		plan.BlockedReason = "working tree or index is dirty"
		return plan, nil
	}
	if state.Busy() {
		plan.BlockedReason = "another Git operation is already in progress"
		return plan, nil
	}
	if pr.State != "MERGED" {
		plan.BlockedReason = fmt.Sprintf("PR is not merged: %s", pr.State)
		return plan, nil
	}

	method := opts.Method
	if method == MethodAuto {
		shape, err := p.store.MergeShape(ctx, pr)
		if err != nil {
			return Plan{}, fmt.Errorf("determine merge shape: %w", err)
		}
		plan.MergeShape = shape
		method, err = methodFromMergeShape(shape)
		if err != nil {
			return Plan{}, err
		}
	}
	plan.Method = method
	plan.Commands = commandsForPlan(plan)
	return plan, nil
}

func methodFromMergeShape(shape string) (ApplyMethod, error) {
	switch shape {
	case "rebase":
		return MethodPatch, nil
	case "squash":
		return MethodSquash, nil
	case "merge_commit":
		return MethodMerge, nil
	default:
		return "", fmt.Errorf("unsupported merge shape: %s", shape)
	}
}

func commandsForPlan(plan Plan) []Command {
	prNumber := strconv.Itoa(plan.PR.Number)
	switch plan.Method {
	case MethodPatch:
		return []Command{
			{Program: "gh", Args: []string{"pr", "diff", prNumber, "--patch"}},
			{Program: "git", Args: []string{"am", "-3"}},
		}
	case MethodCommits:
		commands := make([]Command, 0, len(plan.PR.Commits))
		for _, commit := range plan.PR.Commits {
			commands = append(commands, Command{Program: "git", Args: []string{"cherry-pick", commit.Oid}})
		}
		return commands
	case MethodSquash:
		return []Command{
			{Program: "gh", Args: []string{"pr", "diff", prNumber}},
			{Program: "git", Args: []string{"apply", "--3way", "--index"}},
			{Program: "git", Args: []string{"commit", "-m", squashCommitMessage(plan.PR.Title, plan.PR.Number)}},
		}
	case MethodMerge:
		return []Command{{Program: "git", Args: []string{"cherry-pick", "-m", "1", plan.PR.MergeCommit.Sha}}}
	default:
		return nil
	}
}

func squashCommitMessage(title string, number int) string {
	return fmt.Sprintf("%s (#%d)", title, number)
}
