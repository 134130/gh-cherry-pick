package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/134130/gh-cherry-pick/gitobj"
	"github.com/134130/gh-cherry-pick/internal/color"
	"github.com/134130/gh-cherry-pick/internal/log"
	"github.com/134130/gh-cherry-pick/internal/tui"
)

type CherryPick struct {
	PRNumber      int
	OnTo          string
	MergeStrategy MergeStrategy
	Push          bool
	Worktree      bool
}

func (cherryPick *CherryPick) RunWithContext(ctx context.Context) error {
	logger := log.LoggerFromCtx(ctx)

	logger.Infof("🍒 %s", color.Bold("starting cherry-picker\n"))

	if cherryPick.Worktree {
		err := tui.WithStep(ctx, "setting up worktree cache", func(ctx context.Context, logger log.Logger) error {
			nameWithOwner, err := GetNameWithOwner(ctx)
			if err != nil {
				return fmt.Errorf("error getting repository name: %w", err)
			}

			remoteURL, err := GetRemoteURL(ctx)
			if err != nil {
				return fmt.Errorf("error getting remote URL: %w", err)
			}

			parts := strings.SplitN(nameWithOwner, "/", 2)
			ownerDir := filepath.Join(os.TempDir(), "gh-cherry-pick", parts[0])
			cacheDir := filepath.Join(ownerDir, parts[1])

			if err := os.MkdirAll(ownerDir, 0755); err != nil {
				return fmt.Errorf("error creating cache directory: %w", err)
			}

			if _, statErr := os.Stat(filepath.Join(cacheDir, ".git")); os.IsNotExist(statErr) {
				logger.Infof("cloning repository to cache: %s", cacheDir)
				if err := Clone(ctx, remoteURL, cacheDir); err != nil {
					return fmt.Errorf("error cloning repository: %w", err)
				}
			} else {
				logger.Infof("using cached repository: %s", cacheDir)
			}

			return os.Chdir(cacheDir)
		})
		if err != nil {
			return err
		}
	}

	err := tui.WithStep(ctx, "checking is repository ready", func(ctx context.Context, logger log.Logger) error {
		if !cherryPick.Worktree {
			logger.Infof("checking is repository dirty")
			if dirty, err := IsDirty(ctx); err != nil {
				return fmt.Errorf("error checking if the repository is dirty: %w", err)
			} else if dirty {
				return fmt.Errorf("the repository is dirty. please commit your changes before continuing")
			}

			logger.Infof("checking is repository in a rebase or am")
			if rebaseOrAm, err := IsInRebaseOrAm(ctx); err != nil {
				return fmt.Errorf("error checking if the repository is in a rebase or am: %w", err)
			} else if rebaseOrAm {
				return fmt.Errorf("the repository is in a rebase or am. please resolve the rebase or am before continuing")
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	var pr *gitobj.PullRequest
	err = tui.WithStep(ctx, "validating the pull request", func(ctx context.Context, logger log.Logger) error {
		logger.WithField("pr", cherryPick.PRNumber).Infof("fetching the pull request")
		if pr, err = GetPullRequest(ctx, cherryPick.PRNumber); err != nil {
			return fmt.Errorf("error getting the pull request: %w", err)
		}

		logger.Successf("%s  %s %s", pr.PRNumberString(), pr.Url, color.Grey(pr.Author.Login))

		if pr.State != gitobj.PullRequestStateMerged {
			return fmt.Errorf("PR is not merged (current state: %s). please ensure the PR is merged before continuing", pr.StateString())
		}

		return nil
	})
	if err != nil {
		return err
	}

	var mergeStrategy MergeStrategy
	err = tui.WithStep(ctx, "determining merge strategy", func(ctx context.Context, logger log.Logger) error {
		if cherryPick.MergeStrategy == MergeStrategyAuto {
			logger.Infof("no merge strategy given, determining merge strategy")

			if mergeStrategy, err = PRMergedWith(ctx, cherryPick.PRNumber, pr.MergeCommit.Sha); err != nil {
				return fmt.Errorf("error determining merge strategy: %w", err)
			}

			logger.Successf("determined merge strategy as %s", color.Cyan(mergeStrategy))
		} else {
			mergeStrategy = cherryPick.MergeStrategy
			logger.Infof("use merge strategy %s with given flag", color.Cyan(cherryPick.MergeStrategy))
		}

		return nil
	})
	if err != nil {
		return err
	}

	var cherryPickBranchName = fmt.Sprintf("cherry-pick-pr-%d-onto-%s-%d", cherryPick.PRNumber, strings.ReplaceAll(cherryPick.OnTo, "/", "-"), time.Now().Unix())
	err = tui.WithStep(ctx, "checking out branch", func(ctx context.Context, logger log.Logger) error {
		logger.WithField("branch", pr.BaseRefName).Infof("fetching the branch")
		if err = Fetch(ctx, "origin", pr.BaseRefName); err != nil {
			return fmt.Errorf("error fetching the branch '%s': %w", cherryPick.OnTo, err)
		}

		if cherryPick.OnTo != pr.BaseRefName {
			logger.WithField("branch", cherryPick.OnTo).Infof("fetching the branch")
			if err = Fetch(ctx, "origin", cherryPick.OnTo); err != nil {
				return fmt.Errorf("error fetching the branch '%s': %w", cherryPick.OnTo, err)
			}
		}

		logger.WithField("branch", cherryPickBranchName).
			WithField("base", cherryPick.OnTo).
			Infof("checking out to new branch")
		if err = CheckoutNewBranch(ctx, cherryPickBranchName, "origin", cherryPick.OnTo); err != nil {
			return fmt.Errorf("error checking out to new branch '%s': %w", cherryPickBranchName, err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	switch mergeStrategy {
	case MergeStrategyRebase:
		err = tui.WithStep(ctx, "rebasing PR", func(ctx context.Context, logger log.Logger) error {
			logger.WithField("pr", cherryPick.PRNumber).Infof("fetching diff")
			var prDiff bytes.Buffer
			if err = NewCommand("gh", "pr", "diff", strconv.Itoa(cherryPick.PRNumber), "--patch").Run(ctx, WithStdout(&prDiff)); err != nil {
				return fmt.Errorf("error getting PR diff: %w", err)
			}

			logger.Infof("applying diff")
			if err = NewCommand("git", "am", "-3").Run(ctx, WithStdin(&prDiff)); err != nil {
				helpMsg := fmt.Sprintf("run %s after resolve the conflicts\nrun %s if you want to abort the rebase", color.Green("`git am --continue`"), color.Yellow("`git am --abort`"))

				var gitError *GitError
				if errors.As(err, &gitError) && gitError.ExitCode == 1 && strings.Contains(gitError.Stderr, "error: Failed to merge in the changes") {
					return fmt.Errorf("error applying PR diff\n%s", helpMsg)
				}
				return fmt.Errorf("error applying PR diff\n%s\n\n%w", helpMsg, err)
			}

			return nil
		})
		if err != nil {
			return err
		}
		logger.Successf("rebased branch %s onto %s", color.Cyan(cherryPickBranchName), color.Cyan(cherryPick.OnTo))

	case MergeStrategySquash:
		err = tui.WithStep(ctx, "cherry-picking PR merge commit", func(ctx context.Context, logger log.Logger) error {
			logger.WithField("merge_commit", pr.MergeCommit.Sha[:7]).Infof("cherry-picking")
			if err = NewCommand("git", "cherry-pick", "--keep-redundant-commits", pr.MergeCommit.Sha).Run(ctx); err != nil {
				helpMsg := fmt.Sprintf("run %v after resolve the conflicts\nrun %v if you want to abort the cherry-pick", color.Green("`git cherry-pick --continue`"), color.Yellow("`git cherry-pick --abort`"))

				var gitError *GitError
				if errors.As(err, &gitError) && gitError.ExitCode == 1 && strings.Contains(gitError.Stderr, "error: could not apply") {
					return fmt.Errorf("error cherry-picking PR merge commit\n%s", helpMsg)
				}
				return fmt.Errorf("error cherry-picking PR merge commit\n%s\n\n%w", helpMsg, err)
			}

			return nil
		})
		if err != nil {
			return err
		}
		logger.Successf("cherry-picked branch %s onto %s", color.Cyan(cherryPickBranchName), color.Cyan(cherryPick.OnTo))
	}

	if cherryPick.Push || cherryPick.Worktree {
		err = tui.WithStep(ctx, "pushing branch", func(ctx context.Context, logger log.Logger) error {
			logger.WithField("branch", cherryPickBranchName).Infof("pushing")
			if err = Push(ctx, "origin", cherryPickBranchName); err != nil {
				return fmt.Errorf("error pushing branch %s: %w", cherryPickBranchName, err)
			}

			repoWebURL, repoURLErr := GetRepoWebURL(ctx)
			if repoURLErr == nil {
				logger.Successf("pushed branch %s\ncreate a pull request by visiting:\n    %s",
					color.Cyan(cherryPickBranchName),
					fmt.Sprintf("%s/compare/%s...%s", repoWebURL, cherryPick.OnTo, cherryPickBranchName),
				)
			} else {
				logger.Successf("pushed branch %s", color.Cyan(cherryPickBranchName))
			}

			return nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}
