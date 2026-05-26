# gh-cherry-pick v3 CLI Interface

This document defines the proposed v3 command surface. v3 is greenfield: it does
not preserve the v2 target-branch/backport workflow.

## Goals

- Make `gh cherry-pick <pr>` mean: apply the PR onto the current branch.
- Never create a new branch unless a future command explicitly says so.
- Never push by default.
- Provide a previewable plan that can be rendered as stable JSON for tests and
  automation.
- Keep Git conflict recovery understandable and close to native Git behavior.

## Non-Goals

- No `--onto` target branch workflow.
- No backport branch creation workflow.
- No `--push` workflow.
- No temporary `--worktree` workflow in the initial v3 surface.
- No compatibility aliases for v2 flags such as `-pr`, `-onto`, or `-merge`.

## Mental Model

`gh cherry-pick` is a PR-to-current-branch applicator.

The current branch is the destination. The selected PR is the source. The tool
resolves the PR metadata, chooses or validates an apply method, performs a
preflight check, then applies the PR without changing branches.

## Commands

| Command | Mutates Git? | Purpose |
| --- | --- | --- |
| `gh cherry-pick` | TUI dependent | Open interactive TUI when stdout is a TTY. Error in non-TTY mode. |
| `gh cherry-pick <pr>` | Yes | Apply PR to the current branch. |
| `gh cherry-pick plan <pr>` | No | Show the exact planned operation. |
| `gh cherry-pick inspect <pr>` | No | Show PR metadata used to choose an apply method. |
| `gh cherry-pick status` | No | Show an in-progress v3 operation, if one exists. |
| `gh cherry-pick continue` | Yes | Continue an interrupted v3 operation after conflicts are resolved. |
| `gh cherry-pick abort` | Yes | Abort an interrupted v3 operation and restore the pre-apply state when possible. |

## Primary Usage

```sh
gh cherry-pick 123
gh cherry-pick 123 --method patch
gh cherry-pick 123 --allow-open
gh cherry-pick plan 123
gh cherry-pick plan 123 --format json
gh cherry-pick inspect 123
```

## Apply Methods

`--method` describes how the PR content is applied. It does not describe how the
PR was merged on GitHub, although `auto` may infer that information.

| Method | Expected Git Operation | Primary Use |
| --- | --- | --- |
| `auto` | Infer the safest supported method from PR metadata. | Default behavior. |
| `patch` | Apply `gh pr diff --patch` with `git am -3`. | Rebase-merged PRs and open PRs. |
| `commits` | Cherry-pick the PR commits in order. | Preserve individual PR commits. |
| `squash` | Apply the PR diff as one new commit. | Collapse PR content intentionally. |
| `merge` | Cherry-pick a merge commit with mainline parent 1. | GitHub merge-commit PRs. |

### Auto Method

`auto` should choose a method using explicit PR and commit metadata.

| PR Shape | Auto Method |
| --- | --- |
| Merged by squash | `squash` or direct cherry-pick of the squash merge commit when equivalent. |
| Merged by rebase | `patch`. |
| Merged by merge commit | `merge`. |
| Open PR with `--allow-open` | `patch`. |
| Open PR without `--allow-open` | Fail before mutation. |

The implementation may use a more precise internal representation than these
method names, but the visible plan must explain the selected method and the
commands it intends to run.

## Flags

| Flag | Applies To | Default | Meaning |
| --- | --- | --- | --- |
| `--method auto\|patch\|commits\|squash\|merge` | apply, plan | `auto` | Select how to apply the PR. |
| `--allow-open` | apply, plan, inspect | `false` | Allow applying an open PR. |
| `--no-commit` | apply, plan | `false` | Apply changes to index/worktree without creating commits, when supported. |
| `--format human\|json` | plan, inspect, status | `human` | Select output format. |
| `--json` | plan, inspect, status | `false` | Alias for `--format json`. |
| `--no-color` | all | `false` | Disable ANSI color. |
| `--verbose`, `-v` | all | `false` | Print lower-level command/progress details. |
| `--yes`, `-y` | TUI/confirming flows | `false` | Skip confirmation where a command would otherwise ask. |

## Default Apply Behavior

`gh cherry-pick <pr>` should:

1. Resolve repository context from the current working directory.
2. Resolve the current branch and HEAD.
3. Fail if the worktree or index is dirty.
4. Fail if a rebase, am, merge, or cherry-pick operation is already in progress.
5. Load PR metadata.
6. Fail if the PR is open unless `--allow-open` is set.
7. Build and print a concise plan.
8. Apply the PR to the current branch without switching branches.
9. Print the created commit SHAs or the resulting worktree state.

The command should not push, create a remote branch, create a local branch, or
change the PR on GitHub.

## Plan Output

Human `plan` output should answer:

- Which PR is the source?
- Which branch and HEAD are the destination?
- Which method was selected?
- Which Git commands will run?
- What preflight checks must pass?

JSON `plan` output should be stable enough for tests. A representative shape:

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
  "method": "patch",
  "allowOpen": false,
  "noCommit": false,
  "commands": [
    ["gh", "pr", "diff", "123", "--patch"],
    ["git", "am", "-3"]
  ],
  "preflight": [
    "repository",
    "clean_worktree",
    "no_git_operation_in_progress",
    "pr_applicable"
  ]
}
```

## Inspect Output

`inspect` is for explaining method selection without planning a destination
mutation. It should include:

- PR number, title, URL, author, state, draft state.
- Base and head branch names.
- Merge commit SHA, if present.
- PR commit SHAs in order.
- Inferred GitHub merge shape.
- Recommended apply method.
- Any reason the PR is not applicable by default.

## Conflict Lifecycle

When a conflict occurs, the tool should leave the repository in the native Git
conflict state and write enough v3 operation state to support `status`,
`continue`, and `abort`.

Expected messages:

- `gh cherry-pick status` shows the PR, method, current Git operation, and next
  command.
- `gh cherry-pick continue` delegates to the native continuation command, then
  finishes the v3 operation state.
- `gh cherry-pick abort` delegates to the native abort command, then removes the
  v3 operation state.

Native command guidance should still be printed for users who prefer Git:

- `git am --continue` / `git am --abort`
- `git cherry-pick --continue` / `git cherry-pick --abort`

## TUI Shape

Running `gh cherry-pick` with a TTY should open a TUI that:

1. Lists recently merged PRs by default.
2. Allows toggling open PRs when the user opts into `--allow-open` behavior.
3. Shows the inferred method and destination branch for the selected PR.
4. Shows a plan preview before mutation.
5. Executes the same engine as the CLI path.

Running `gh cherry-pick` without a TTY should fail with usage instead of opening
an interactive flow.

## Removed v2 Surface

The following v2 concepts should not exist in v3:

| v2 Surface | v3 Replacement |
| --- | --- |
| `-pr <number>` | Positional `<pr>`. |
| `-onto <branch>` | None. Destination is always current branch. |
| `-merge auto\|squash\|rebase` | `--method auto\|patch\|commits\|squash\|merge`. |
| `-push` | None. Users can push with Git explicitly. |
| `-worktree` | None in initial v3. |
| Generated `cherry-pick-pr-...` branch | None. |
