# gh-cherry-pick

A GitHub CLI extension to cherry-pick a PR's merged commit based on target branch.

## Description

- It will be useful when you are using to cherry-pick a PR, which is merged to the main branch, to the release branch.

![image](https://github.com/user-attachments/assets/bd95beeb-3366-46a4-b1de-c4825c7f6fc5)


## Installation

```shell
gh extension install 134130/gh-cherry-pick
```

## Usage

- `gh cherry-pick -pr <pr_number> -onto <target_branch> [-merge auto|squash|rebase] [-push] [-worktree]` to cherry-pick a PR based on target branch. It determines the merge strategy based on the original PR's merge strategy.
- `gh cherry-pick -pr <pr_number> -onto <target_branch> -merge squash` to cherry-pick a PR's merged commit based on target branch.
- `gh cherry-pick -pr <pr_number> -onto <target_branch> -merge rebase` to cherry-pick all the commits from a PR based on target branch.

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-pr` | (required) | PR number to cherry-pick |
| `-onto` | (required) | Target branch to cherry-pick onto |
| `-merge` | `auto` | Merge strategy: `auto`, `squash`, or `rebase` |
| `-push` | `false` | Push the cherry-picked branch to the remote |
| `-worktree` | `false` | Use a temporary worktree cached in the OS temp directory |

### `--worktree` option

The `--worktree` flag lets you run cherry-pick without a clean local working tree. Instead of operating on your current repository, it clones the repository to an OS temp directory (`$TMPDIR/gh-cherry-pick/<owner>/<repo>`) and runs all operations there. On subsequent runs, the cached clone is reused.

This is useful when:
- Your working tree has uncommitted changes you don't want to stash
- You want to cherry-pick from any directory, not just inside the repository

```shell
# First run: clones the repository to the temp cache
gh cherry-pick -pr 123 -onto release/1.0 --worktree

# Subsequent runs: reuses the cached clone
gh cherry-pick -pr 456 -onto release/1.0 --worktree --push
```

## Related

- [gh-domino](https://github.com/134130/gh-domino) - A GitHub CLI extension to rebase stacked pull requests
- [gh-poi](https://github.com/seachicken/gh-poi) - A GitHub CLI extension to safely clean up local branches you no longer need

## Alterantives

- In case one wants to base on a branch being [squash-merged](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/configuring-pull-request-merges/about-merge-methods-on-github#squashing-your-merge-commits) into main and avoid git showing conflicts, one can use [magic-merge-commit](https://github.com/koppor/magic-merge-commit/tree/main) to create a merge commit satisfying git and enabling a clean merge of `main` again.
