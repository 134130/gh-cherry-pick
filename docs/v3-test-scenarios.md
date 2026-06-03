# gh-cherry-pick v3 Test Scenarios

This document turns the v3 CLI interface into implementation-oriented test
scenarios. v3 is greenfield and focuses on the current branch as the destination.

## Current Test Baseline

The v3 suite uses local integration-style fixtures in `test/scenario_test.go`.
Those tests validate real Git behavior without depending on GitHub state.

`git/cherry_pick_test.go` is a legacy v2 remote fixture test and is not the
source of truth for v3.

## External Fixture Repository

Manual end-to-end demos use `https://github.com/134130/test-cherry-pick`.
Run `./init all` in that repository to reset it and print the current PR
manifest.

| Logical ID | Shape | Demo Target | Expected |
| --- | --- | --- |
| `basic` | Squash-merged PR | `target-basic` | Applies cleanly to the current branch. |
| `squash` | Squash-merged PR | `target-squash` | `auto` resolves to `squash`. |
| `rebase` | Rebase-merged PR | `target-rebase` | `auto` resolves to `patch`. |
| `merge` | Merge-commit PR | `target-merge` | `auto` resolves to `merge`. |
| `multi` | Multi-commit PR | `target-multi` | `--method commits` cherry-picks commits in order. |
| `conflict` | Squash-merged conflicting PR | `target-conflict` | Leaves native conflict state. |
| `open` | Open PR | `target-open` | Blocks before mutation. |

## Test Layers

| Layer | Purpose |
| --- | --- |
| CLI parse/run tests | Validate flags, confirmation policy, dry-run, and JSON mode without running Git. |
| Planner tests | Validate PR metadata to plan conversion. |
| Executor tests | Validate Git command sequence and failure guidance. |
| Integration tests | Validate real Git behavior against fixtures. |

## CLI Matrix

| ID | Command | Context | Expected |
| --- | --- | --- | --- |
| CLI-001 | `gh cherry-pick 123 --dry-run` | Any | Builds and prints a human plan; no mutation. |
| CLI-002 | `gh cherry-pick 123 --dry-run --json` | Any | Prints a single JSON plan object; no mutation. |
| CLI-003 | `gh cherry-pick 123` | TTY stdin | Prints plan, asks for confirmation, applies on yes. |
| CLI-004 | `gh cherry-pick 123` | TTY stdin | Prints plan, exits without mutation on no/empty answer. |
| CLI-005 | `gh cherry-pick 123` | Non-TTY stdin | Fails before building a plan. |
| CLI-006 | `gh cherry-pick 123 --yes` | Non-TTY stdin | Builds plan and applies without prompt. |
| CLI-007 | `gh cherry-pick 123 --json` | TTY stdin | Fails unless `--yes` is also set. |
| CLI-008 | `gh cherry-pick 123 --yes --json` | Any | Prints a single JSON result object. |
| CLI-009 | `gh cherry-pick 123 --method patch` | Any | Uses explicit `patch` method. |
| CLI-010 | `gh cherry-pick 123 --method invalid` | Any | Fails validation before mutation. |

## Planner Matrix

| ID | Input | Expected |
| --- | --- | --- |
| PLN-001 | Squash-merged PR with `--method auto` | Concrete method is `squash`. |
| PLN-002 | Rebase-merged PR with `--method auto` | Concrete method is `patch`. |
| PLN-003 | Merge-commit PR with `--method auto` | Concrete method is `merge`. |
| PLN-004 | Any merged PR with explicit `--method commits` | Explicit method is used without auto lookup. |
| PLN-005 | Dirty worktree/index | Plan is blocked with `blockedReason`; no commands. |
| PLN-006 | Rebase/am/merge/cherry-pick in progress | Plan is blocked with `blockedReason`; no commands. |
| PLN-007 | Open PR | Plan is blocked; first skeleton only applies merged PRs. |
| PLN-008 | Missing PR | Planner returns lookup error. |
| PLN-009 | Detached HEAD | Planner returns current-branch error. |

## Executor Matrix

| ID | Method | Expected Commands |
| --- | --- | --- |
| EXE-001 | `patch` | `gh pr diff <pr> --patch`, then `git am -3`. |
| EXE-002 | `commits` | `git cherry-pick <sha>` for each PR commit in order. |
| EXE-003 | `squash` | `gh pr diff <pr>`, `git apply --3way --index`, then `git commit -m`. |
| EXE-004 | `merge` | `git cherry-pick -m 1 <merge-commit>`. |
| EXE-005 | blocked plan | Executor returns blocked error without running commands. |
| EXE-006 | patch conflict | Error includes `git am --continue` and `git am --abort`. |
| EXE-007 | cherry-pick conflict | Error includes `git cherry-pick --continue` and `git cherry-pick --abort`. |
| EXE-008 | squash apply conflict | Error tells the user to resolve, stage, and commit manually. |

## Output Matrix

| ID | Command | Expected |
| --- | --- | --- |
| OUT-001 | `gh cherry-pick 123 --dry-run --json` | stdout is valid JSON with `pr`, `destination`, `method`, and `commands`. |
| OUT-002 | `gh cherry-pick 123 --yes --json` | stdout is one JSON result object with `commitShas`. |
| OUT-003 | `gh cherry-pick 123 --dry-run` | Human plan includes PR, destination, method, and commands. |
| OUT-004 | `gh cherry-pick 123` | Human apply result includes created commit SHAs. |
| OUT-005 | blocked plan in dry-run | Human/JSON output includes the blocked reason. |

## Integration Matrix

Use the manifest printed by `test-cherry-pick ./init all` to map logical IDs to
the current PR numbers.

| ID | Command | Fixture | Expected |
| --- | --- | --- | --- |
| INT-001 | `gh cherry-pick <basic-pr> --yes` | `basic` | Applies to current branch and keeps current branch checked out. |
| INT-002 | `gh cherry-pick <squash-pr> --dry-run --json` | `squash` | Emits stable non-mutating JSON with method `squash`. |
| INT-003 | `gh cherry-pick <rebase-pr> --yes` | `rebase` | Applies to current branch and keeps current branch checked out. |
| INT-004 | `gh cherry-pick <merge-pr> --yes` | `merge` | Applies merge commit content to current branch. |
| INT-005 | `gh cherry-pick <multi-pr> --method commits --yes` | `multi` | Applies PR commits in order. |
| INT-006 | `gh cherry-pick <conflict-pr> --yes` | `conflict` | Leaves native conflict state and prints recovery guidance. |
| INT-007 | `gh cherry-pick <open-pr> --dry-run` | `open` | Blocks because the PR is not merged. |
| INT-008 | `gh cherry-pick <basic-pr> --yes` | Dirty worktree | Fails without mutation. |
| INT-009 | `gh cherry-pick <basic-pr> --yes` | Clean branch | Does not switch branches, create branches, or push. |

## Minimum v3 Acceptance Set

| ID | Scenario |
| --- | --- |
| MVP-001 | `gh cherry-pick <squash-pr> --dry-run --json` emits stable non-mutating JSON. |
| MVP-002 | `gh cherry-pick <squash-pr> --yes` applies squash fixture PR to current branch. |
| MVP-003 | `gh cherry-pick <rebase-pr> --yes` applies rebase fixture PR to current branch. |
| MVP-004 | `gh cherry-pick <conflict-pr> --yes` reports conflict with native recovery guidance. |
| MVP-005 | Dirty worktree preflight fails without mutation. |
| MVP-006 | Successful apply does not switch branches, create branches, or push. |
