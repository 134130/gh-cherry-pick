# gh-cherry-pick v3 CLI Interface

This document defines the v3 command surface. v3 is greenfield and does not
preserve the v2 target-branch workflow.

## Goals

- Make `gh cherry-pick <pr>` mean: apply the PR onto the current branch.
- Separate interactive CLI from machine CLI without adding a TUI.
- Never create a branch or push.
- Use `--dry-run` for preview and `--yes` for noninteractive mutation.
- Keep conflict recovery close to native Git behavior.

## Mental Model

`gh cherry-pick` is a PR-to-current-branch applicator.

The current branch is the destination. The selected PR is the source. The tool
resolves PR metadata, builds a plan, optionally asks for confirmation, then
applies the PR without switching branches.

## Command Surface

| Command | Mutates Git? | Purpose |
| --- | --- | --- |
| `gh cherry-pick <pr>` | Yes after confirmation | Show a plan, ask for confirmation, then apply. |
| `gh cherry-pick <pr> --yes` | Yes | Apply without interactive confirmation. Required for non-TTY mutation. |
| `gh cherry-pick <pr> --dry-run` | No | Show the plan and exit. |
| `gh cherry-pick <pr> --dry-run --json` | No | Show the plan as stable JSON. |

There are no v3 subcommands in the first skeleton. Preview is handled by
`--dry-run`.

## Flags

| Flag | Default | Meaning |
| --- | --- | --- |
| `--method auto\|patch\|commits\|squash\|merge` | `auto` | Select how to apply the PR. |
| `--dry-run` | `false` | Print the plan without applying it. |
| `--yes`, `-y` | `false` | Skip interactive confirmation. |
| `--format human\|json` | `human` | Select output format. |
| `--json` | `false` | Alias for `--format json`. |
| `--no-color` | `false` | Disable ANSI color. |
| `--verbose`, `-v` | `false` | Print command/progress details. |

## Apply Methods

`--method` describes how PR content is applied. It does not directly describe
how the PR was merged on GitHub, although `auto` infers that information.

| Method | Expected Operation |
| --- | --- |
| `auto` | Infer a concrete method from PR metadata. |
| `patch` | `gh pr diff <pr> --patch` piped into `git am -3`. |
| `commits` | `git cherry-pick` each PR commit in order. |
| `squash` | Apply the PR diff to the index and create one commit. |
| `merge` | `git cherry-pick -m 1 <merge-commit>`. |

### Auto Method

| PR Shape | Auto Method |
| --- | --- |
| Merged by squash | `squash` |
| Merged by rebase | `patch` |
| Merged by merge commit | `merge` |
| Open or closed-unmerged PR | Block before mutation |

## Interactive CLI

Interactive CLI means plain terminal prompts, not a TUI.

`gh cherry-pick <pr>` should:

1. Build a plan for the current branch.
2. Print the human plan.
3. If the plan is blocked, fail before asking for confirmation.
4. Ask `Apply this PR? (y/N):`.
5. Apply only when the user answers `y` or `yes`.

If stdin is not a terminal, mutation requires `--yes`.

## Machine CLI

Machine CLI is explicit:

- Use `--dry-run --json` for a non-mutating plan.
- Use `--yes --json` for a noninteractive mutation result.
- JSON output must be a single JSON object on stdout.
- Progress, prompts, and diagnostics must not be mixed into JSON stdout.

## Default Apply Behavior

`gh cherry-pick <pr>` should:

1. Resolve repository context from the current working directory.
2. Resolve the current branch and HEAD.
3. Fail if the worktree or index is dirty.
4. Fail if a rebase, am, merge, or cherry-pick operation is already in progress.
5. Load PR metadata including merge commit and commit list.
6. Fail if the PR is not merged.
7. Build a concrete method plan.
8. Apply the PR to the current branch without switching branches.
9. Print the created commit SHAs.

The command should not push, create a branch, or modify GitHub PR metadata.

## JSON Plan Shape

Representative `gh cherry-pick 123 --dry-run --json` output:

```json
{
  "pr": {
    "number": 123,
    "state": "MERGED",
    "title": "Fix retry behavior",
    "url": "https://github.com/owner/repo/pull/123"
  },
  "destination": {
    "branch": "feature/current",
    "head": "abc1234"
  },
  "requestedMethod": "auto",
  "method": "patch",
  "mergeShape": "rebase",
  "commands": [
    {"program": "gh", "args": ["pr", "diff", "123", "--patch"]},
    {"program": "git", "args": ["am", "-3"]}
  ]
}
```

Blocked plans include `blockedReason`.

## Conflict Handling

The first v3 skeleton does not add lifecycle subcommands. On conflict, leave the
repository in native Git state and print the relevant native recovery command:

- `git am --continue` / `git am --abort`
- `git cherry-pick --continue` / `git cherry-pick --abort`
- for squash diff apply, resolve files, stage them, and commit manually
