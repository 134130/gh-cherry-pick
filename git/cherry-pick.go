package git

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/134130/gh-cherry-pick/internal/tui"
)

type CherryPick struct {
	PRNumber int
	To       string
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

	var cherryPickBranchName = fmt.Sprintf("cherry-pick-pr-%d-to-%s-%d", cherryPick.PRNumber, cherryPick.To, time.Now().Unix())
	tui.WithSpinner(ctx, fmt.Sprintf("Fetching PR #%d to branch %s", cherryPick.PRNumber, cherryPickBranchName), func(ctx context.Context) (string, error) {

		if _, err := (&Git{}).RunWithContext(ctx, "fetch", "--recurse-submodules", "origin", fmt.Sprintf("pull/%d/head:%s", cherryPick.PRNumber, cherryPickBranchName)); err != nil {
			return "", fmt.Errorf("error fetching PR branch: %w", err)
		}

		return fmt.Sprintf("Fetched PR #%d to branch %s", cherryPick.PRNumber, cherryPickBranchName), nil
	})

	tui.WithSpinner(ctx, fmt.Sprintf("Checking out branch %s", cherryPickBranchName), func(ctx context.Context) (string, error) {
		if _, err := (&Git{}).RunWithContext(ctx, "checkout", cherryPickBranchName); err != nil {
			return "", fmt.Errorf("error checking out branch %s: %w", cherryPickBranchName, err)
		}

		return fmt.Sprintf("Checked out branch %s", cherryPickBranchName), nil
	})

	if cherryPick.Rebase {
		tui.WithSpinner(ctx, fmt.Sprintf("Rebasing branch %s onto %s", cherryPickBranchName, cherryPick.To), func(ctx context.Context) (string, error) {
			prDiff, err := (&GH{}).RunWithContext(ctx, "pr", "diff", strconv.Itoa(cherryPick.PRNumber), "--patch")
			if err != nil {
				return "", fmt.Errorf("error getting PR diff: %w", err)
			}

			if _, err = (&Git{Stdin: bytes.NewBufferString(prDiff)}).RunWithContext(ctx, "am", "-3"); err != nil {
				return "", fmt.Errorf("error applying PR diff: %w", err)
			}

			return fmt.Sprintf("Rebased branch %s onto %s", cherryPickBranchName, cherryPick.To), nil
		})
	} else {
		tui.WithSpinner(ctx, fmt.Sprintf("Cherry-picking branch %s onto %s", cherryPickBranchName, cherryPick.To), func(ctx context.Context) (string, error) {
			mergeCommit, err := (&GH{}).RunWithContext(ctx, "pr", "view", strconv.Itoa(cherryPick.PRNumber), "--json", "--mergeCommit", "--jq", ".mergeCommit.oid")
			if err != nil {
				return "", fmt.Errorf("error getting PR merge commit: %w", err)
			}

			if len(mergeCommit) == 0 {
				return "", fmt.Errorf("error getting PR merge commit: please ensure the PR has been merged")
			}

			if _, err = (&Git{}).RunWithContext(ctx, "cherry-pick", "--keep-redundant-commits", mergeCommit); err != nil {
				return "", fmt.Errorf("error cherry-picking PR merge commit: %w", err)
			}

			return fmt.Sprintf("Cherry-picked branch %s onto %s", cherryPickBranchName, cherryPick.To), nil
		})
	}

	return nil
}
