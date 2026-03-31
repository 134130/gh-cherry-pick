package git

import (
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

func (m MergeStrategy) Validate() error {
	switch m {
	case MergeStrategyRebase, MergeStrategySquash, MergeStrategyAuto:
		return nil
	default:
		return fmt.Errorf("invalid merge strategy %q: must be one of rebase, squash, auto", m)
	}
}

func PRMergedWith(ctx context.Context, prNumber int, mergeCommitSHA string) (MergeStrategy, error) {
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

	hostname, err := GetGHHostname(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get GH hostname: %w", err)
	}

	prevCommitSHA, err := ghAPIQuery(ctx, hostname,
		fmt.Sprintf("repos/%s/commits/%s", nameWithOwner, mergeCommitSHA),
		".parents[0].sha",
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("failed to get previous commit SHA for merge commit %s: %w", mergeCommitSHA, err)
	}

	prNumbersStr, err := ghAPIQuery(ctx, hostname,
		fmt.Sprintf("repos/%s/commits/%s/pulls", nameWithOwner, prevCommitSHA),
		".[].number",
		map[string]string{"Accept": "application/vnd.github+json"},
	)
	if err != nil {
		return "", fmt.Errorf("failed to get related PR numbers for commit %s: %w", prevCommitSHA, err)
	}

	targetPRStr := strconv.Itoa(prNumber)
	for _, line := range strings.Split(prNumbersStr, "\n") {
		if strings.TrimSpace(line) == targetPRStr {
			return MergeStrategyRebase, nil
		}
	}
	return MergeStrategySquash, nil
}
