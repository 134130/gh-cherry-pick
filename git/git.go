package git

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/134130/gh-cherry-pick/gitobj"
)

func GetNameWithOwner(ctx context.Context) (string, error) {
	stdout := &bytes.Buffer{}
	args := []string{"repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner"}
	if err := NewCommand("gh", args...).Run(ctx, WithStdout(stdout)); err != nil {
		return "", fmt.Errorf("failed to get repository name with owner: %w", err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func GetRepoRoot(ctx context.Context) (string, error) {
	stdout := &bytes.Buffer{}
	if err := NewCommand("git", "rev-parse", "--show-toplevel").Run(ctx, WithStdout(stdout)); err != nil {
		return "", fmt.Errorf("failed to resolve the repository root: %w", err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func GetPullRequest(ctx context.Context, number int) (*gitobj.PullRequest, error) {
	stdout := &bytes.Buffer{}
	args := []string{"pr", "view", strconv.Itoa(number), "--json", "number,title,url,author,state,isDraft"}
	if err := NewCommand("gh", args...).Run(ctx, WithStdout(stdout)); err != nil {
		return nil, fmt.Errorf("failed to get the pull request: %w", err)
	}

	var pr gitobj.PullRequest
	if err := json.NewDecoder(stdout).Decode(&pr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal the pull request: %w", err)
	}
	return &pr, nil
}

func CheckoutNewBranch(ctx context.Context, newBranch, startPoint string) error {
	if err := NewCommand("git", "checkout", "-b", newBranch, startPoint).Run(ctx); err != nil {
		return fmt.Errorf("failed to checkout a new branch: %w", err)
	}
	return nil
}

func Push(ctx context.Context, remote, ref string) error {
	if err := NewCommand("git", "push", "--set-upstream", remote, ref).Run(ctx); err != nil {
		return fmt.Errorf("failed to push the branch: %w", err)
	}
	return nil
}

func Fetch(ctx context.Context, remote, refspec string) error {
	if err := NewCommand("git", "fetch", "--recurse-submodules", remote, refspec).Run(ctx); err != nil {
		return fmt.Errorf("failed to fetch the branch: %w", err)
	}
	return nil
}

func IsDirty(ctx context.Context) (bool, error) {
	stdout := &bytes.Buffer{}
	if err := NewCommand("git", "status", "--porcelain").Run(ctx, WithStdout(stdout)); err != nil {
		return false, fmt.Errorf("failed to check if the repository is dirty: %w", err)
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
