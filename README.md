## Description
- A GitHub CLI extension to cherry-pick a PR's merged commit based on target branch.
- It will be useful when you are using to cherry-pick a PR, which is merged to the main branch, to the release branch.

## Installation
```shell
gh extension install 134130/gh-cherry-pick
```

## Usage
- `gh cherry-pick --pr <pr_number> --onto <target_branch> [--merge auto]` to cherry-pick a PR based on target branch. It determines the merge strategy based on the original PR's merge strategy.
- `gh cherry-pick --pr <pr_number> --onto <target_branch> --merge squash` to cherry-pick a PR's merged commit based on target branch.
- `gh cherry-pick --pr <pr_number> --onto <target_branch> --merge rebase` to cherry-pick all the commits from a PR based on target branch. 
