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
    "/repos/${gh_repo_name_with_owner}/commits/${prev_commit_sha}/pulls" \
      | jq -c '.[].number'
  )

  if echo "${prev_commit_related_pr_numbers}" | grep -q "${pr_number}"; then
    return 1
  fi

  return 0
}
