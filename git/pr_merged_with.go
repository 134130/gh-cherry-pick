package git

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
)

type MergeStrategy string

const (
	MergeStrategyRebase MergeStrategy = "rebase"
	MergeStrategySquash MergeStrategy = "squash"
	MergeStrategyAuto   MergeStrategy = "auto"
)

func PRMergedWith(ctx context.Context, prNumber int) (MergeStrategy, error) {
	stdout := &bytes.Buffer{}
	if err := NewCommand("gh", "pr", "view", strconv.Itoa(prNumber), "--json", "mergeCommit", "--jq", ".mergeCommit.oid").Run(ctx, WithStdout(stdout)); err != nil {
		return "", fmt.Errorf("failed to get merge commit SHA for PR #%d: %w", prNumber, err)
	}

	mergeCommitSHA := strings.TrimSpace(stdout.String())
	if len(mergeCommitSHA) == 0 {
		return "", fmt.Errorf("failed to get merge commit SHA for PR #%d: PR not merged", prNumber)
	}

	return inspectMergeStrategy(ctx, prNumber, mergeCommitSHA)
}

func inspectMergeStrategy(ctx context.Context, prNumber int, mergeCommitSHA string) (MergeStrategy, error) {
	nameWithOwner, err := GetNameWithOwner(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get repository name with owner: %w", err)
	}

	endpoint := fmt.Sprintf("repos/%v/commits/%v~1", nameWithOwner, mergeCommitSHA)
	args := []string{"api", endpoint, "--jq", ".sha"}

	stdout := &bytes.Buffer{}
	if err = NewCommand("gh", args...).Run(ctx, WithStdout(stdout)); err != nil {
		return "", fmt.Errorf("failed to get previous commit SHA for merge commit %s: %w", mergeCommitSHA, err)
	}

	prevCommitSHA := strings.TrimSpace(stdout.String())

	endpoint = fmt.Sprintf("repos/%v/commits/%v/pulls", nameWithOwner, prevCommitSHA)
	args = []string{"api"}
	args = append(args, "-H", "Accept: application/vnd.github+json")
	args = append(args, "-H", "X-GitHub-Api-Version: 2022-11-28")
	args = append(args, endpoint, "--jq", ".[].number")

	stdout = &bytes.Buffer{}
	if err = NewCommand("gh", args...).Run(ctx, WithStdout(stdout)); err != nil {
		return "", fmt.Errorf("failed to get related PR numbers for commit %s: %w", prevCommitSHA, err)
	}

	prevCommitRelatedPRNumbers := strings.TrimSpace(stdout.String())
	if len(prevCommitRelatedPRNumbers) == 0 {
		return "", fmt.Errorf("failed to get related PR numbers for commit %s: no related PRs", prevCommitSHA)
	}

	if strings.Contains(prevCommitRelatedPRNumbers, strconv.Itoa(prNumber)) {
		return MergeStrategyRebase, nil
	} else {
		return MergeStrategySquash, nil
	}
}
