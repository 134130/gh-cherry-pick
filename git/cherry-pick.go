package git

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cli/go-gh/v2"

	"github.com/134130/gh-cherry-pick/internal/tui"
)

type CherryPick struct {
	PRNumber      int
	OnTo          string
	MergeStrategy MergeStrategy
}

func (cherryPick *CherryPick) RunWithContext(ctx context.Context) error {
	if dirty, err := IsDirty(ctx); err != nil {
		return fmt.Errorf("error checking if the repository is dirty: %w", err)
	} else if dirty {
		return fmt.Errorf("the repository is dirty. please commit your changes before continuing")
	}

	if rebaseOrAm, err := IsInRebaseOrAm(ctx); err != nil {
		return fmt.Errorf("error checking if the repository is in a rebase or am: %w", err)
	} else if rebaseOrAm {
		return fmt.Errorf("the repository is in a rebase or am. please resolve the rebase or am before continuing")
	}

	stdout, stderr, err := ExecContext(ctx, "gh", "pr", "view", strconv.Itoa(cherryPick.PRNumber), "--json", "state", "--jq", ".state")
	if err != nil {
		return fmt.Errorf("error getting PR #%d: %w: %s", cherryPick.PRNumber, err, stderr.String())
	}

	state := strings.TrimSpace(stdout.String())
	if state != "MERGED" {
		return fmt.Errorf("PR #%d is not merged (current state: %s). please ensure the PR is merged before continuing", cherryPick.PRNumber, state)
	}

	var cherryPickBranchName = fmt.Sprintf("cherry-pick-pr-%d-onto-%s-%d", cherryPick.PRNumber, cherryPick.OnTo, time.Now().Unix())
	tui.WithSpinner(ctx, fmt.Sprintf("Fetching %s to branch %s", cherryPick.OnTo, cherryPickBranchName), func(ctx context.Context) (string, error) {
		if _, stderr, err = ExecContext(ctx, "git", "fetch", "--recurse-submodules", "origin", cherryPick.OnTo); err != nil {
			return "", fmt.Errorf("error fetching PR branch: %w: %s", err, stderr.String())
		}

		return fmt.Sprintf("Fetched %s to branch %s", cherryPick.OnTo, cherryPickBranchName), nil
	})

	tui.WithSpinner(ctx, fmt.Sprintf("Checking out branch %s", cherryPickBranchName), func(ctx context.Context) (string, error) {
		if _, stderr, err = ExecContext(ctx, "git", "switch", cherryPickBranchName); err != nil {
			return "", fmt.Errorf("error checking out branch %s: %w: %s", cherryPickBranchName, err, stderr.String())
		}

		return fmt.Sprintf("Checked out branch %s", cherryPickBranchName), nil
	})

	mergeStrategy := cherryPick.MergeStrategy
	tui.WithSpinner(ctx, fmt.Sprintf("Determining merge strategy for PR #%d", cherryPick.PRNumber), func(ctx context.Context) (string, error) {
		if cherryPick.MergeStrategy == MergeStrategyAuto {
			mergeStrategy, err = PRMergedWith(ctx, cherryPick.PRNumber)
			if err != nil {
				return "", fmt.Errorf("error determining merge strategy: %w", err)
			}
		}

		return fmt.Sprintf("Determined merge strategy for PR #%d: %s", cherryPick.PRNumber, mergeStrategy), nil
	})

	switch mergeStrategy {
	case MergeStrategyRebase:
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

	case MergeStrategySquash:
		tui.WithSpinner(ctx, fmt.Sprintf("Cherry-picking branch %s onto %s", cherryPickBranchName, cherryPick.OnTo), func(ctx context.Context) (string, error) {
			stdout, stderr, err = ExecContext(ctx, "gh", "pr", "view", strconv.Itoa(cherryPick.PRNumber), "--json", "mergeCommit", "--jq", ".mergeCommit.oid")
			if err != nil {
				return "", fmt.Errorf("error getting PR merge commit: %w: %s", err, stderr.String())
			}

			mergeCommit := strings.TrimSpace(stdout.String())
			if _, stderr, err = ExecContext(ctx, "git", "cherry-pick", "--keep-redundant-commits", mergeCommit); err != nil {
				return "", fmt.Errorf("error cherry-picking PR merge commit: %w: %s", err, stderr.String())
			}

			return fmt.Sprintf("Cherry-picked branch %s onto %s", cherryPickBranchName, cherryPick.OnTo), nil
		})
	}

	return nil
}
