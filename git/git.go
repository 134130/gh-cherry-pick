package git

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/134130/gh-cherry-pick/gitobj"
)

func GetNameWithOwner(ctx context.Context) (string, error) {
	stdout, stderr, err := ExecContext(ctx, "gh", "repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner")
	if err != nil {
		return "", fmt.Errorf("failed to get repository name with owner: %w: %s", err, stderr.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}

func GetRepoRoot(ctx context.Context) (string, error) {
	stdout, stderr, err := ExecContext(ctx, "git", "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("failed to resolve the repository root: %w: %s", err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

func GetPullRequest(ctx context.Context, number int) (*gitobj.PullRequest, error) {
	stdout, stderr, err := ExecContext(ctx, "gh", "pr", "view", strconv.Itoa(number), "--json", "number,title,url,author,state,isDraft")
	if err != nil {
		return nil, fmt.Errorf("failed to get the pull request: %w: %s", err, stderr.String())
	}

	var pr gitobj.PullRequest
	if err = json.Unmarshal(stdout.Bytes(), &pr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal the pull request: %w", err)
	}
	return &pr, nil
}

func IsDirty(ctx context.Context) (bool, error) {
	stdout, stderr, err := ExecContext(ctx, "git", "status", "--porcelain")
	if err != nil {
		return false, fmt.Errorf("failed to check if the repository is dirty: %w: %s", err, stderr.String())
	}

	return len(strings.TrimSpace(stdout.String())) > 0, nil
}

func IsInRebaseOrAm(ctx context.Context) (bool, error) {
	repoRoot, err := GetRepoRoot(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to check if the repository is in a rebase or am: %w", err)
	}

	var rebaseMagicFile = fmt.Sprintf("%s/.git/rebase-apply", repoRoot)
	if _, err := os.Stat(rebaseMagicFile); err == nil {
		return true, nil
	} else if !os.IsNotExist(err) {
		return false, fmt.Errorf("failed to check if the repository is in a rebase or am: %w", err)
	}

	return false, nil
}
