package git

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

type MergeStrategy string

const (
	Rebase MergeStrategy = "rebase"
	Squash MergeStrategy = "squash"
)

func PRMergedWith(ctx context.Context, prNumber int) (MergeStrategy, error) {
	stdout, stderr, err := ExecContext(ctx, "gh", "pr", "view", strconv.Itoa(prNumber), "--json", "mergeCommit", "--jq", ".mergeCommit.oid")
	if err != nil {
		return "", fmt.Errorf("failed to get merge commit SHA for PR #%d: %w: %s", prNumber, err, stderr.String())
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

	stdout, stderr, err := ExecContext(ctx, "gh", "api", fmt.Sprintf("repos/%s/commits/%s~1", nameWithOwner, mergeCommitSHA), "--jq", ".sha")
	if err != nil {
		return "", fmt.Errorf("failed to get previous commit SHA for merge commit %s: %w: %s", mergeCommitSHA, err, stderr.String())
	}

	prevCommitSHA := strings.TrimSpace(stdout.String())
	stdout, stderr, err = ExecContext(ctx, "gh", "api",
		"-H", "Accept: application/vnd.github+json",
		"-H", "X-GitHub-Api-Version: 2022-11-28",
		fmt.Sprintf("repos/%s/commits/%s/pulls", nameWithOwner, prevCommitSHA), "--jq", ".[].number")
	if err != nil {
		return "", fmt.Errorf("failed to get related PR numbers for commit %s: %w: %s", prevCommitSHA, err, stderr.String())
	}

	prevCommitRelatedPRNumbers := strings.TrimSpace(stdout.String())
	if len(prevCommitRelatedPRNumbers) == 0 {
		return "", fmt.Errorf("failed to get related PR numbers for commit %s: no related PRs", prevCommitSHA)
	}

	if strings.Contains(prevCommitRelatedPRNumbers, strconv.Itoa(prNumber)) {
		return Rebase, nil
	} else {
		return Squash, nil
	}
}
