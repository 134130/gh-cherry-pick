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
	"github.com/134130/gh-cherry-pick/internal/once"
)

var nameWithOwnerOnce = once.OnceValue[string]{}

func GetNameWithOwner(ctx context.Context) (string, error) {
	return nameWithOwnerOnce.Do(ctx, func(ctx context.Context) (string, error) {
		stdout := &bytes.Buffer{}
		args := []string{"repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner"}
		if err := NewCommand("gh", args...).Run(ctx, WithStdout(stdout)); err != nil {
			return "", err
		}
		return strings.TrimSpace(stdout.String()), nil
	})
}

func GetRepoRoot(ctx context.Context) (string, error) {
	stdout := &bytes.Buffer{}
	if err := NewCommand("git", "rev-parse", "--show-toplevel").Run(ctx, WithStdout(stdout)); err != nil {
		return "", err
	}
	return strings.TrimSpace(stdout.String()), nil
}

func GetPullRequest(ctx context.Context, number int) (*gitobj.PullRequest, error) {
	stdout := &bytes.Buffer{}
	args := []string{"pr", "view", strconv.Itoa(number), "--json", "number,title,url,author,state,isDraft,mergeCommit,baseRefName,headRefName"}
	if err := NewCommand("gh", args...).Run(ctx, WithStdout(stdout)); err != nil {
		return nil, fmt.Errorf("failed to get the pull request: %w", err)
	}

	var pr gitobj.PullRequest
	if err := json.NewDecoder(stdout).Decode(&pr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal the pull request: %w", err)
	}
	return &pr, nil
}

func CheckoutNewBranch(ctx context.Context, newBranch, remote, startPoint string) error {
	remoteStartPoint := fmt.Sprintf("%s/%s", remote, startPoint)
	return NewCommand("git", "switch", "-c", newBranch, "-t", remoteStartPoint, remoteStartPoint).Run(ctx)
}

func Push(ctx context.Context, remote, ref string) error {
	return NewCommand("git", "push", "--set-upstream", remote, ref).Run(ctx)
}

func Fetch(ctx context.Context, remote, refspec string) error {
	return NewCommand("git", "fetch", "--recurse-submodules", remote, refspec).Run(ctx)
}

func IsDirty(ctx context.Context) (bool, error) {
	stdout := &bytes.Buffer{}
	if err := NewCommand("git", "status", "--porcelain").Run(ctx, WithStdout(stdout)); err != nil {
		return false, err
	}
	return len(strings.TrimSpace(stdout.String())) > 0, nil
}

func IsInRebaseOrAm(ctx context.Context) (bool, error) {
	repoRoot, err := GetRepoRoot(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get the repository root: %w", err)
	}

	var rebaseMagicFile = fmt.Sprintf("%s/.git/rebase-apply", repoRoot)
	if _, err = os.Stat(rebaseMagicFile); err == nil {
		return true, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}

	return false, nil
}
