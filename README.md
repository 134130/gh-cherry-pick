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

- `gh cherry-pick -pr <pr_number> -onto <target_branch> [-merge auto|squash|rebase] [-push]` to cherry-pick a PR based on target branch. It determines the merge strategy based on the original PR's merge strategy.
- `gh cherry-pick -pr <pr_number> -onto <target_branch> -merge squash` to cherry-pick a PR's merged commit based on target branch.
- `gh cherry-pick -pr <pr_number> -onto <target_branch> -merge rebase` to cherry-pick all the commits from a PR based on target branch.

## Related

- [gh-domino](https://github.com/134130/gh-domino) - A GitHub CLI extension to rebase stacked pull requests
- [gh-poi](https://github.com/seachicken/gh-poi) - A GitHub CLI extension to safely clean up local branches you no longer need

## Alterantives

- In case one wants to base on a branch being [squash-merged](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/configuring-pull-request-merges/about-merge-methods-on-github#squashing-your-merge-commits) into main and avoid git showing conflicts, one can use [magic-merge-commit](https://github.com/koppor/magic-merge-commit/tree/main) to create a merge commit satisfying git and enabling a clean merge of `main` again.
