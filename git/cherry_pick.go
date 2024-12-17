package git

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cli/go-gh/v2"

	"github.com/134130/gh-cherry-pick/gitobj"
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

	var pr *gitobj.PullRequest
	err := tui.WithSpinner(ctx, fmt.Sprintf("fetching the pull request %s", tui.Cyan(fmt.Sprintf("#%d", cherryPick.PRNumber))), func(ctx context.Context) (string, error) {
		pullReq, err := GetPullRequest(ctx, cherryPick.PRNumber)
		if err != nil {
			return "", fmt.Errorf("error getting the pull request: %w", err)
		}

		pr = pullReq

		return fmt.Sprintf("🍒 %s (%s) by %s", pr.Title, pr.PRNumberString(), tui.Grey(pr.Author.Login)), nil
	})
	if err != nil {
		return err
	}

	if pr.State != gitobj.PullRequestStateMerged {
		return fmt.Errorf("PR %s is not merged (current state: %s). please ensure the PR is merged before continuing", pr.PRNumberString(), pr.StateString())
	}

	err = tui.WithSpinner(ctx, fmt.Sprintf("fetching branch: %s", tui.Cyan(cherryPick.OnTo)), func(ctx context.Context) (string, error) {
		if _, stderr, err := ExecContext(ctx, "git", "fetch", "--recurse-submodules", "origin", cherryPick.OnTo); err != nil {
			return "", fmt.Errorf("error fetching branch %s: %w: %s", tui.Cyan(cherryPick.OnTo), err, stderr.String())
		}

		return fmt.Sprintf("fetched branch: %s", tui.Cyan(cherryPick.OnTo)), nil
	})
	if err != nil {
		return err
	}

	var cherryPickBranchName = fmt.Sprintf("cherry-pick-pr-%d-onto-%s-%d", cherryPick.PRNumber, cherryPick.OnTo, time.Now().Unix())
	err = tui.WithSpinner(ctx, fmt.Sprintf("Checking out branch %s based on %s", tui.Cyan(cherryPickBranchName), tui.Cyan(cherryPick.OnTo)), func(ctx context.Context) (string, error) {
		if _, stderr, err := ExecContext(ctx, "git", "switch", "-c", cherryPickBranchName, fmt.Sprintf("origin/%s", cherryPick.OnTo)); err != nil {
			return "", fmt.Errorf("error checking out branch %s: %w: %s", tui.Cyan(cherryPickBranchName), err, stderr.String())
		}

		return fmt.Sprintf("checked out branch %s based on %s", tui.Cyan(cherryPickBranchName), tui.Cyan(cherryPick.OnTo)), nil
	})
	if err != nil {
		return err
	}

	mergeStrategy := cherryPick.MergeStrategy
	if cherryPick.MergeStrategy == MergeStrategyAuto {
		err = tui.WithSpinner(ctx, fmt.Sprintf("Determining merge strategy"), func(ctx context.Context) (string, error) {
			mergeStrategy, err = PRMergedWith(ctx, cherryPick.PRNumber)
			if err != nil {
				return "", fmt.Errorf("error determining merge strategy: %w", err)
			}

			return fmt.Sprintf("determined merge strategy as %s", tui.Cyan(mergeStrategy)), nil
		})
		if err != nil {
			return err
		}
	} else {
		tui.PrintSuccess(fmt.Sprintf("using %s merge strategy with given flag", tui.Cyan(mergeStrategy)))
	}

	switch mergeStrategy {
	case MergeStrategyRebase:
		err = tui.WithSpinner(ctx, fmt.Sprintf("Rebasing branch %s onto %s", tui.Cyan(cherryPickBranchName), tui.Cyan(cherryPick.OnTo)), func(ctx context.Context) (string, error) {
			var prDiff bytes.Buffer
			prDiff, stderr, err := gh.ExecContext(ctx, "pr", "diff", strconv.Itoa(cherryPick.PRNumber), "--patch")
			if err != nil {
				return "", fmt.Errorf("error getting PR diff: %w: %s", err, stderr.String())
			}

			if _, _, err = ExecContextWithStdin(ctx, &prDiff, "git", "am", "-3"); err != nil {
				return "", fmt.Errorf("error applying PR diff. please resolve the conflicts and run %s. if you want to abort the rebase, run %s", tui.Green("`git am --continue`"), tui.Yellow("`git am --abort`"))
			}

			return fmt.Sprintf("rebased branch %s onto %s", cherryPickBranchName, cherryPick.OnTo), nil
		})
		if err != nil {
			return err
		}

	case MergeStrategySquash:
		err = tui.WithSpinner(ctx, fmt.Sprintf("Cherry-picking branch %s onto %s", tui.Cyan(cherryPickBranchName), tui.Cyan(cherryPick.OnTo)), func(ctx context.Context) (string, error) {
			stdout, stderr, err := ExecContext(ctx, "gh", "pr", "view", strconv.Itoa(cherryPick.PRNumber), "--json", "mergeCommit", "--jq", ".mergeCommit.oid")
			if err != nil {
				return "", fmt.Errorf("error getting PR merge commit: %w: %s", err, stderr.String())
			}

			mergeCommit := strings.TrimSpace(stdout.String())
			if _, stderr, err = ExecContext(ctx, "git", "cherry-pick", "--keep-redundant-commits", mergeCommit); err != nil {
				return "", fmt.Errorf("error cherry-picking PR merge commit: %s. please resolve the conflicts and run %s. if you want to abort the cherry-pick, run %s", stderr.String(), tui.Green("`git cherry-pick --continue`"), tui.Yellow("`git cherry-pick --abort`"))
			}

			return fmt.Sprintf("cherry-picked branch %s onto %s", tui.Cyan(cherryPickBranchName), tui.Cyan(cherryPick.OnTo)), nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}
