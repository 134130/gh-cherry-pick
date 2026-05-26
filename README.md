# gh-cherry-pick

A GitHub CLI extension to apply a pull request to the current branch.

## Installation

```shell
gh extension install 134130/gh-cherry-pick
```

## Usage

Apply a merged pull request to the branch you are currently on:

```shell
gh cherry-pick 123
```

By default, the command prints a plan and asks for confirmation before mutating
Git. For automation, pass `--yes`:

```shell
gh cherry-pick 123 --yes
```

Preview the plan without applying it:

```shell
gh cherry-pick 123 --dry-run
gh cherry-pick 123 --dry-run --json
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--method` | `auto` | Apply method: `auto`, `patch`, `commits`, `squash`, or `merge` |
| `--dry-run` | `false` | Print the plan without applying it |
| `--yes`, `-y` | `false` | Apply without interactive confirmation |
| `--format` | `human` | Output format: `human` or `json` |
| `--json` | `false` | Shortcut for `--format json` |
| `--no-color` | `false` | Disable ANSI color |
| `--verbose`, `-v` | `false` | Print command/progress details |

### Apply Methods

| Method | Behavior |
|--------|----------|
| `auto` | Infer a concrete method from PR metadata |
| `patch` | Apply `gh pr diff <pr> --patch` with `git am -3` |
| `commits` | Cherry-pick each PR commit in order |
| `squash` | Apply the PR diff and create one commit |
| `merge` | Cherry-pick a merge commit with `-m 1` |

## Related

- [gh-domino](https://github.com/134130/gh-domino) - A GitHub CLI extension to rebase stacked pull requests
- [gh-poi](https://github.com/seachicken/gh-poi) - A GitHub CLI extension to safely clean up local branches you no longer need
