package git

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cli/go-gh/v2"

	"github.com/134130/gh-cherry-pick/internal/tui"
)

type CherryPick struct {
	PRNumber int
	OnTo     string
	Rebase   bool
}

func (cherryPick *CherryPick) RunWithContext(ctx context.Context) error {
	if dirty, err := IsDirty(ctx); err != nil {
		if _, err = fmt.Fprintf(os.Stderr, "error checking if the repository is dirty: %v", err); err != nil {
			return err
		}
	} else if dirty {
		if _, err = fmt.Fprintf(os.Stderr, "the repository is dirty. Please commit your changes before continuing"); err != nil {
			return err
		}
	}

	if rebaseOrAm, err := IsInRebaseOrAm(ctx); err != nil {
		if _, err = fmt.Fprintf(os.Stderr, "error checking if the repository is in a rebase or am: %v", err); err != nil {
			return err
		}
	} else if rebaseOrAm {
		if _, err = fmt.Fprintf(os.Stderr, "the repository is in a rebase or am. Please resolve the rebase or am before continuing"); err != nil {
			return err
		}
	}

	var cherryPickBranchName = fmt.Sprintf("cherry-pick-pr-%d-onto-%s-%d", cherryPick.PRNumber, cherryPick.OnTo, time.Now().Unix())
	tui.WithSpinner(ctx, fmt.Sprintf("Fetching PR #%d to branch %s", cherryPick.PRNumber, cherryPickBranchName), func(ctx context.Context) (string, error) {

		if _, stderr, err := ExecContext(ctx, "git", "fetch", "--recurse-submodules", "origin", fmt.Sprintf("pull/%d/head:%s", cherryPick.PRNumber, cherryPickBranchName)); err != nil {
			return "", fmt.Errorf("error fetching PR branch: %w: %s", err, stderr.String())
		}

		return fmt.Sprintf("Fetched PR #%d to branch %s", cherryPick.PRNumber, cherryPickBranchName), nil
	})

	tui.WithSpinner(ctx, fmt.Sprintf("Checking out branch %s", cherryPickBranchName), func(ctx context.Context) (string, error) {
		if _, stderr, err := ExecContext(ctx, "git", "checkout", cherryPickBranchName); err != nil {
			return "", fmt.Errorf("error checking out branch %s: %w: %s", cherryPickBranchName, err, stderr.String())
		}

		return fmt.Sprintf("Checked out branch %s", cherryPickBranchName), nil
	})

	if cherryPick.Rebase {
		tui.WithSpinner(ctx, fmt.Sprintf("Rebasing branch %s onto %s", cherryPickBranchName, cherryPick.OnTo), func(ctx context.Context) (string, error) {
			prDiff, stderr, err := gh.ExecContext(ctx, "pr", "diff", strconv.Itoa(cherryPick.PRNumber), "--patch")
			if err != nil {
				return "", fmt.Errorf("error getting PR diff: %w: %s", err, stderr.String())
			}

			if _, stderr, err = ExecContextWithStdin(ctx, &prDiff, "git", "am", "-3"); err != nil {
				return "", fmt.Errorf("error applying PR diff: %w: %s", err, stderr.String())
			}

			return fmt.Sprintf("Rebased branch %s onto %s", cherryPickBranchName, cherryPick.OnTo), nil
		})
	} else {
		tui.WithSpinner(ctx, fmt.Sprintf("Cherry-picking branch %s onto %s", cherryPickBranchName, cherryPick.OnTo), func(ctx context.Context) (string, error) {
			stdout, stderr, err := ExecContext(ctx, "gh", "pr", "view", strconv.Itoa(cherryPick.PRNumber), "--json", "--mergeCommit", "--jq", ".mergeCommit.oid")
			if err != nil {
				return "", fmt.Errorf("error getting PR merge commit: %w: %s", err, stderr.String())
			}

			mergeCommit := strings.TrimSpace(stdout.String())
			if len(mergeCommit) == 0 {
				return "", fmt.Errorf("error getting PR merge commit: please ensure the PR has been merged")
			}

			if _, stderr, err = ExecContext(ctx, "git", "cherry-pick", "--keep-redundant-commits", mergeCommit); err != nil {
				return "", fmt.Errorf("error cherry-picking PR merge commit: %w: %s", err, stderr.String())
			}

			return fmt.Sprintf("Cherry-picked branch %s onto %s", cherryPickBranchName, cherryPick.OnTo), nil
		})
	}

	return nil
}
