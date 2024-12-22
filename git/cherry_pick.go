package git

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/134130/gh-cherry-pick/gitobj"
	"github.com/134130/gh-cherry-pick/internal/log"
	"github.com/134130/gh-cherry-pick/internal/tui"
)

type CherryPick struct {
	PRNumber      int
	OnTo          string
	MergeStrategy MergeStrategy
	Push          bool
}

func (cherryPick *CherryPick) RunWithContext(ctx context.Context) error {
	logger := log.LoggerFromCtx(ctx)
	err := tui.WithSpinner(ctx, "checking repository is ready ...", func(ctx context.Context, logger log.Logger) error {
		logger.Infof("checking repository is dirty")
		if dirty, err := IsDirty(ctx); err != nil {
			return fmt.Errorf("error checking if the repository is dirty: %w", err)
		} else if dirty {
			return fmt.Errorf("the repository is dirty. please commit your changes before continuing")
		}

		logger.Infof("checking repository is in a rebase or am")
		if rebaseOrAm, err := IsInRebaseOrAm(ctx); err != nil {
			return fmt.Errorf("error checking if the repository is in a rebase or am: %w", err)
		} else if rebaseOrAm {
			return fmt.Errorf("the repository is in a rebase or am. please resolve the rebase or am before continuing")
		}

		return nil
	})
	if err != nil {
		return err
	}
	logger.Successf("repository is available for cherry-pick")

	var pr *gitobj.PullRequest
	title := fmt.Sprintf("fetching the pull request %s ...", tui.Cyan(fmt.Sprintf("#%d", cherryPick.PRNumber)))
	err = tui.WithSpinner(ctx, title, func(ctx context.Context, logger log.Logger) error {
		logger.Infof("getting the pull request %s ...", tui.Cyan(fmt.Sprintf("#%d", cherryPick.PRNumber)))
		if pr, err = GetPullRequest(ctx, cherryPick.PRNumber); err != nil {
			return fmt.Errorf("error getting the pull request: %w", err)
		}

		if pr.State != gitobj.PullRequestStateMerged {
			return fmt.Errorf("PR %s is not merged (current state: %s). please ensure the PR is merged before continuing", pr.PRNumberString(), pr.StateString())
		}

		return nil
	})
	if err != nil {
		return err
	}
	logger.Successf("fetched the pull request - %v (%v) by %v", tui.Cyan(pr.Title), pr.PRNumberString(), tui.Cyan(pr.Author.Login))

	var mergeStrategy MergeStrategy
	err = tui.WithSpinner(ctx, "determining merge strategy ...", func(ctx context.Context, logger log.Logger) error {
		if cherryPick.MergeStrategy == MergeStrategyAuto {
			logger.Infof("determining merge strategy automatically")

			if mergeStrategy, err = PRMergedWith(ctx, cherryPick.PRNumber); err != nil {
				return fmt.Errorf("error determining merge strategy: %w", err)
			}
		} else {
			logger.Infof("using merge strategy %s with given flag", tui.Cyan(cherryPick.MergeStrategy))
		}

		return nil
	})
	if err != nil {
		return err
	}
	logger.Successf("determined merge strategy as %s", tui.Cyan(mergeStrategy))

	err = tui.WithSpinner(ctx, "fetching the commits between the PR and the target branch ...", func(ctx context.Context, logger log.Logger) error {
		logger.Infof("getting the merge base between %v and %v", tui.Cyan(pr.MergeCommit.Sha), tui.Cyan(fmt.Sprintf("origin/%s", cherryPick.OnTo)))
		mergeBase, err := GetMergeBase(ctx, pr.MergeCommit.Sha, fmt.Sprintf("origin/%s", cherryPick.OnTo))
		if err != nil {
			return fmt.Errorf("error getting the merge base: %w", err)
		}

		logger.Infof("fetching the commits between %v and %v", tui.Cyan(mergeBase), tui.Cyan(pr.MergeCommit.Sha))
		if err = Fetch(ctx, "origin", fmt.Sprintf("%s..%s", mergeBase, pr.MergeCommit.Sha)); err != nil {
			return fmt.Errorf("error fetching the commits between the PR and the target branch: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}
	logger.Successf("fetched the commits between the PR and the target branch")

	var cherryPickBranchName = fmt.Sprintf("cherry-pick-pr-%d-onto-%s-%d", cherryPick.PRNumber, strings.ReplaceAll(cherryPick.OnTo, "/", "-"), time.Now().Unix())
	err = tui.WithSpinner(ctx, "checking out branch ...", func(ctx context.Context, logger log.Logger) error {
		logger.Infof("branch name:    %v", tui.Cyan(cherryPickBranchName))
		logger.Infof("starting point: %v", tui.Cyan(fmt.Sprintf("origin/%s", cherryPick.OnTo)))

		logger.Infof("fetching the branch %v", tui.Cyan(cherryPick.OnTo))
		if err = Fetch(ctx, "origin", cherryPick.OnTo); err != nil {
			return fmt.Errorf("error fetching the branch '%s': %w", cherryPick.OnTo, err)
		}

		logger.Infof("checking out a new branch %v based on %v", tui.Cyan(cherryPickBranchName), tui.Cyan(cherryPick.OnTo))
		if err = CheckoutNewBranch(ctx, cherryPickBranchName, fmt.Sprintf("origin/%s", cherryPick.OnTo)); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}
	logger.Successf("checked out to %s based on %s", tui.Cyan(cherryPickBranchName), tui.Cyan(cherryPick.OnTo))

	switch mergeStrategy {
	case MergeStrategyRebase:
		err = tui.WithSpinner(ctx, "rebasing ...", func(ctx context.Context, logger log.Logger) error {
			logger.Infof("fetching diff")
			var prDiff bytes.Buffer
			if err = NewCommand("gh", "pr", "diff", strconv.Itoa(cherryPick.PRNumber), "--patch").Run(ctx, WithStdout(&prDiff)); err != nil {
				return fmt.Errorf("error getting PR diff: %w", err)
			}

			logger.Infof("applying diff")
			if err = NewCommand("git", "am", "-3").Run(ctx, WithStdin(&prDiff)); err != nil {
				return fmt.Errorf("error applying PR diff\nplease resolve the conflicts and run %s. if you want to abort the rebase, run %s", tui.Green("`git am --continue`"), tui.Yellow("`git am --abort`"))
			}

			return nil
		})
		if err != nil {
			return err
		}
		logger.Successf("rebased branch %s onto %s", tui.Cyan(cherryPickBranchName), tui.Cyan(cherryPick.OnTo))

	case MergeStrategySquash:
		err = tui.WithSpinner(ctx, "cherry-picking ...", func(ctx context.Context, logger log.Logger) error {
			logger.Infof("cherry-picking PR merge commit %v", tui.Cyan(pr.MergeCommit.Sha))
			if err = NewCommand("git", "cherry-pick", "--keep-redundant-commits", pr.MergeCommit.Sha).Run(ctx); err != nil {
				// TODO: handle conflict message
				return fmt.Errorf("error cherry-picking PR merge commit\nplease resolve the conflicts and run %v. if you want to abort the cherry-pick, run %v\n\n%v", tui.Green("`git cherry-pick --continue`"), tui.Yellow("`git cherry-pick --abort`"), err)
			}

			return nil
		})
		if err != nil {
			return err
		}
		logger.Successf("cherry-picked branch %s onto %s", tui.Cyan(cherryPickBranchName), tui.Cyan(cherryPick.OnTo))
	}

	return nil
}
