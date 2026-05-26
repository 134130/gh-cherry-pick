# gh-cherry-pick v3 Test Scenarios

This document turns the v3 CLI interface into implementation-oriented test
scenarios. The scenarios are intentionally written as a matrix so they can later
be translated into unit tests, integration tests, TUI model tests, and fixture
tests.

## Existing Test Baseline

The current test suite has one integration-style test table in
`git/cherry_pick_test.go`.

| Current Case | Fixture PR | Current Behavior |
| --- | --- | --- |
| Squash merged PR | `#4` | Succeeds when applied onto `release/10.0`. |
| Rebase merged PR | `#5` | Succeeds when applied onto `release/10.0`. |
| Conflicting PR | `#7` | Fails with conflict guidance. |

v3 should reuse the same fixture concepts, but the destination must become the
current branch rather than an `--onto` branch.

## Test Layers

| Layer | Purpose | Example Target |
| --- | --- | --- |
| CLI parse tests | Validate command and flag surface without running Git. | `internal/cli` |
| Planner tests | Validate PR metadata to plan conversion. | `internal/app` |
| Executor tests | Validate Git command sequence and failure handling. | `internal/app` |
| TUI model tests | Validate interactive state transitions. | `internal/tui` |
| Integration tests | Validate real Git behavior against fixtures. | `test` or `git` |

## CLI Surface Matrix

| ID | Command | Context | Expected |
| --- | --- | --- | --- |
| CLI-001 | `gh cherry-pick 123` | Non-TTY or TTY | Parses as apply command for PR `123`. |
| CLI-002 | `gh cherry-pick` | TTY | Opens TUI. |
| CLI-003 | `gh cherry-pick` | Non-TTY | Fails with usage and no mutation. |
| CLI-004 | `gh cherry-pick plan 123` | Any | Parses as non-mutating plan command. |
| CLI-005 | `gh cherry-pick inspect 123` | Any | Parses as non-mutating inspect command. |
| CLI-006 | `gh cherry-pick status` | Any | Parses without PR number. |
| CLI-007 | `gh cherry-pick continue` | Any | Parses without PR number. |
| CLI-008 | `gh cherry-pick abort` | Any | Parses without PR number. |
| CLI-009 | `gh cherry-pick 123 --method patch` | Any | Method is `patch`. |
| CLI-010 | `gh cherry-pick 123 --method invalid` | Any | Fails validation before mutation. |
| CLI-011 | `gh cherry-pick 123 --allow-open` | Any | Allows open PR planning. |
| CLI-012 | `gh cherry-pick plan 123 --json` | Any | Equivalent to `--format json`. |
| CLI-013 | `gh cherry-pick -pr 123 -onto release/1.0` | Any | Fails as unsupported v2 surface. |
| CLI-014 | `gh cherry-pick 123 --push` | Any | Fails as unsupported v2 surface. |
| CLI-015 | `gh cherry-pick 123 --worktree` | Any | Fails as unsupported v2 surface. |

## Preflight Matrix

| ID | Command | Repo State | Expected |
| --- | --- | --- | --- |
| PRE-001 | `gh cherry-pick 4` | Clean worktree on named branch | Continues to PR validation. |
| PRE-002 | `gh cherry-pick 4` | Dirty worktree | Fails before loading/applying patch. |
| PRE-003 | `gh cherry-pick 4` | Dirty index | Fails before loading/applying patch. |
| PRE-004 | `gh cherry-pick 4` | Rebase in progress | Fails before mutation. |
| PRE-005 | `gh cherry-pick 4` | `git am` in progress | Fails before mutation. |
| PRE-006 | `gh cherry-pick 4` | Cherry-pick in progress | Fails before mutation. |
| PRE-007 | `gh cherry-pick 4` | Merge in progress | Fails before mutation. |
| PRE-008 | `gh cherry-pick 4` | Detached HEAD | Fails unless v3 later defines detached support. |
| PRE-009 | `gh cherry-pick 4` | Not a Git repository | Fails with repository error. |
| PRE-010 | `gh cherry-pick 4` | `git` executable missing | Fails with install guidance. |
| PRE-011 | `gh cherry-pick 4` | `gh` executable missing | Fails with install guidance. |
| PRE-012 | `gh cherry-pick 4` | `gh` not authenticated | Fails before Git mutation. |

## PR Applicability Matrix

| ID | Command | PR State | PR Shape | Expected |
| --- | --- | --- | --- | --- |
| PR-001 | `gh cherry-pick 4` | Merged | Squash merge | Applicable. |
| PR-002 | `gh cherry-pick 5` | Merged | Rebase merge | Applicable. |
| PR-003 | `gh cherry-pick <merge-pr>` | Merged | Merge commit | Applicable. |
| PR-004 | `gh cherry-pick <open-pr>` | Open | Any | Fails before mutation. |
| PR-005 | `gh cherry-pick <open-pr> --allow-open` | Open | Any | Applicable with `patch` method. |
| PR-006 | `gh cherry-pick <draft-pr> --allow-open` | Open draft | Any | Fails unless draft support is explicitly added. |
| PR-007 | `gh cherry-pick <closed-pr>` | Closed unmerged | Any | Fails before mutation. |
| PR-008 | `gh cherry-pick <missing-pr>` | Missing | None | Fails before mutation. |
| PR-009 | `gh cherry-pick <fork-pr>` | Merged or open allowed | Fork head | Applicable if patch can be resolved. |
| PR-010 | `gh cherry-pick <deleted-head-pr> --allow-open` | Open | Missing head ref | Fails with source unavailable. |

## Method Selection Matrix

| ID | Command | PR Shape | Expected Method | Expected Plan |
| --- | --- | --- | --- | --- |
| MET-001 | `gh cherry-pick plan 4` | Squash merge | `squash` | Plan explains single-commit apply. |
| MET-002 | `gh cherry-pick plan 5` | Rebase merge | `patch` | Plan includes `gh pr diff --patch` and `git am -3`. |
| MET-003 | `gh cherry-pick plan <merge-pr>` | Merge commit | `merge` | Plan includes `git cherry-pick -m 1 <sha>`. |
| MET-004 | `gh cherry-pick plan <open-pr> --allow-open` | Open | `patch` | Plan includes PR diff application. |
| MET-005 | `gh cherry-pick plan 4 --method patch` | Squash merge | `patch` | Explicit method overrides auto. |
| MET-006 | `gh cherry-pick plan 5 --method commits` | Rebase merge | `commits` | Plan lists commit cherry-picks in order. |
| MET-007 | `gh cherry-pick plan 4 --method squash` | Squash merge | `squash` | Plan creates one commit. |
| MET-008 | `gh cherry-pick plan <merge-pr> --method merge` | Merge commit | `merge` | Plan validates mainline parent. |
| MET-009 | `gh cherry-pick plan 4 --method merge` | Non-merge commit | None | Fails method validation. |
| MET-010 | `gh cherry-pick plan <open-pr> --method merge --allow-open` | Open | None | Fails method validation. |

## Apply Execution Matrix

| ID | Command | Fixture | Expected |
| --- | --- | --- | --- |
| APP-001 | `gh cherry-pick 4` | Squash merged PR | Applies to current branch and keeps current branch checked out. |
| APP-002 | `gh cherry-pick 5` | Rebase merged PR | Applies to current branch and keeps current branch checked out. |
| APP-003 | `gh cherry-pick <merge-pr>` | Merge commit PR | Applies with mainline parent 1. |
| APP-004 | `gh cherry-pick 7` | Conflicting PR | Leaves conflict state and prints continue/abort guidance. |
| APP-005 | `gh cherry-pick 4 --method patch` | Squash merged PR | Applies via patch path. |
| APP-006 | `gh cherry-pick 5 --method commits` | Rebase merged PR | Creates one commit per PR commit. |
| APP-007 | `gh cherry-pick 4 --method squash` | Squash merged PR | Creates exactly one new commit. |
| APP-008 | `gh cherry-pick 4 --no-commit` | Squash merged PR | Leaves changes staged/worktree-applied without creating a commit. |
| APP-009 | `gh cherry-pick 4` | PR already applied | Fails with already-applied or empty-change guidance. |
| APP-010 | `gh cherry-pick 4` | Clean branch | Does not push or create local/remote branches. |

## Plan and Inspect Matrix

| ID | Command | Expected |
| --- | --- | --- |
| PLN-001 | `gh cherry-pick plan 4` | Prints human-readable plan and does not mutate Git. |
| PLN-002 | `gh cherry-pick plan 4 --format json` | Prints stable JSON plan and does not mutate Git. |
| PLN-003 | `gh cherry-pick plan 4 --method patch` | JSON/human output reflects explicit method. |
| PLN-004 | `gh cherry-pick plan <open-pr>` | Fails by default and does not mutate Git. |
| PLN-005 | `gh cherry-pick plan <open-pr> --allow-open` | Succeeds with `patch` plan and does not mutate Git. |
| INS-001 | `gh cherry-pick inspect 4` | Prints PR metadata, merge shape, and recommended method. |
| INS-002 | `gh cherry-pick inspect 4 --format json` | Prints stable JSON metadata. |
| INS-003 | `gh cherry-pick inspect <missing-pr>` | Fails with PR lookup error. |
| INS-004 | `gh cherry-pick inspect <open-pr>` | Succeeds but marks default apply as blocked unless `--allow-open`. |

## Conflict Lifecycle Matrix

| ID | Command Sequence | Initial State | Expected |
| --- | --- | --- | --- |
| CON-001 | `gh cherry-pick 7` | Clean branch | Conflict occurs and v3 operation state is recorded. |
| CON-002 | `gh cherry-pick status` | Conflict from `CON-001` | Shows PR, method, native Git state, and next commands. |
| CON-003 | Resolve files, `gh cherry-pick continue` | Conflict from `CON-001` | Completes native operation and clears v3 state. |
| CON-004 | `gh cherry-pick abort` | Conflict from `CON-001` | Aborts native operation and clears v3 state. |
| CON-005 | `gh cherry-pick continue` | No v3 operation | Fails with no operation in progress. |
| CON-006 | `gh cherry-pick abort` | No v3 operation | Fails with no operation in progress. |
| CON-007 | Native `git am --continue` | Conflict from patch method | Git can continue; later `gh cherry-pick status` detects completion or stale state. |
| CON-008 | Native `git cherry-pick --abort` | Conflict from commit method | Git can abort; later `gh cherry-pick status` detects completion or stale state. |

## TUI Matrix

| ID | Interaction | Expected |
| --- | --- | --- |
| TUI-001 | Start `gh cherry-pick` in TTY | Loads recently merged PRs. |
| TUI-002 | Select PR | Shows destination as current branch. |
| TUI-003 | Select PR | Shows inferred method. |
| TUI-004 | Toggle open PR visibility | Open PRs appear only after explicit opt-in. |
| TUI-005 | Preview plan | Shows same plan as `gh cherry-pick plan <pr>`. |
| TUI-006 | Confirm execution | Runs same executor as CLI apply. |
| TUI-007 | Cancel before execution | Exits with no Git mutation. |
| TUI-008 | Execution conflict | Shows conflict status and next commands. |
| TUI-009 | Window resize | Layout remains readable. |
| TUI-010 | Ctrl-C before mutation | Cancels cleanly with no Git mutation. |

## Output Matrix

| ID | Command | Expected |
| --- | --- | --- |
| OUT-001 | `gh cherry-pick plan 4 --format json` | stdout is valid JSON; stderr contains progress only when verbose. |
| OUT-002 | `gh cherry-pick inspect 4 --format json` | stdout is valid JSON; no ANSI color. |
| OUT-003 | `gh cherry-pick 4 --no-color` | Human output has no ANSI color. |
| OUT-004 | `gh cherry-pick 4 --verbose` | Includes lower-level command/progress details. |
| OUT-005 | `gh cherry-pick 4` | Success output includes PR number, method, destination branch, and new commit SHA(s). |
| OUT-006 | `gh cherry-pick 7` | Failure output includes native continue/abort guidance. |

## Compatibility Removal Matrix

These tests protect the greenfield contract.

| ID | Command | Expected |
| --- | --- | --- |
| REM-001 | `gh cherry-pick -pr 4` | Fails; use positional PR. |
| REM-002 | `gh cherry-pick -pr 4 -onto release/10.0` | Fails; v3 has no target branch mode. |
| REM-003 | `gh cherry-pick 4 --onto release/10.0` | Fails; v3 has no target branch mode. |
| REM-004 | `gh cherry-pick 4 --push` | Fails; v3 never pushes. |
| REM-005 | `gh cherry-pick 4 --worktree` | Fails; no initial v3 worktree mode. |
| REM-006 | `gh cherry-pick 4 --merge rebase` | Fails; use `--method`. |

## Suggested Fixture Additions

The current fixture coverage maps to squash, rebase, and conflict cases. v3
would benefit from additional fixture PRs or local synthetic repos:

| Fixture Need | Why |
| --- | --- |
| Merge commit PR | Validate `--method merge` and auto merge detection. |
| Open PR | Validate default rejection and `--allow-open`. |
| Draft PR | Validate explicit draft behavior. |
| Already applied PR | Validate empty-change handling. |
| Multi-commit PR | Validate `--method commits` commit ordering. |
| Fork PR | Validate PR diff path independent of local head branch refs. |
| Deleted head PR | Validate source-unavailable errors. |

## Minimum v3 Acceptance Set

Before v3 can replace the current implementation, these scenarios should pass:

| ID | Scenario |
| --- | --- |
| MVP-001 | `gh cherry-pick 4` applies squash fixture PR to current branch. |
| MVP-002 | `gh cherry-pick 5` applies rebase fixture PR to current branch. |
| MVP-003 | `gh cherry-pick 7` reports conflict and leaves recoverable state. |
| MVP-004 | `gh cherry-pick plan 4 --format json` emits stable non-mutating JSON. |
| MVP-005 | Dirty worktree preflight fails without mutation. |
| MVP-006 | Old `-pr/-onto` interface is rejected. |
| MVP-007 | Successful apply does not switch branches, create branches, or push. |
