gh_utils::is_dirty() {
  if git_status=$(git status --porcelain --untracked=no 2>/dev/null) && [[ -n "$git_status" ]]; then
    return 0
  fi
  return 1
}

gh_utils::is_in_rebase_or_am() {
  if repo_root=$(git rev-parse --show-toplevel) && rebase_magic="${repo_root}/.git/rebase-apply" && [[ -e "${rebase_magic}" ]]; then
    return 0
  fi
  return 1
}

gh_utils::is_pr_merged_with_rebase_strategy() {
  local pr_number=$1

  local merge_commit_sha
  merge_commit_sha=$(gh pr view "${pr_number}" --json mergeCommit --jq .mergeCommit.oid)

  local prev_commit_sha
  prev_commit_sha=$(git rev-parse "${merge_commit_sha}~1")

  local prev_commit_related_pr_numbers
  prev_commit_related_pr_numbers=$(gh api \
    -H "Accept: application/vnd.github+json" \
    -H "X-GitHub-Api-Version: 2022-11-28" \
    "/repos/$(gh repo view --json nameWithOwner)/commits/${prev_commit_sha}/pulls" \
      | jq -c '.[].number'
  )

  if echo "${prev_commit_related_pr_numbers}" | grep -q "${pr_number}"; then
    return 1
  fi

  return 0
}
