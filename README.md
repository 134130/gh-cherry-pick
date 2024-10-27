## Description
- A GitHub CLI extension to cherry-pick a PR's merged commit based on target branch.
- It will be useful when you are using to cherry-pick a PR, which is merged to the main branch, to the release branch.

## Installation
```shell
gh extension install 134130/gh-cherry-pick
```

## Usage
- `gh cherry-pick --pr <pr_number> --to <target_branch>` to cherry-pick a PR's merged commit based on target branch.
- `gh cherry-pick --pr <pr_number> --to <target_branch> --rebase` to cherry-pick all the commits from a PR based on target branch. 
