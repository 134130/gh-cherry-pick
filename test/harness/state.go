package harness

import (
	"fmt"
	"sort"
	"sync"

	"github.com/134130/gh-cherry-pick/gitobj"
)

const (
	MergeShapeSquash      = "squash"
	MergeShapeRebase      = "rebase"
	MergeShapeMergeCommit = "merge_commit"
)

type State struct {
	mu      sync.Mutex
	prs     map[int]PR
	related map[string][]int
}

func NewState() *State {
	return &State{
		prs:     map[int]PR{},
		related: map[string][]int{},
	}
}

func (s *State) Upsert(pr PR) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prs[pr.Number] = pr.Clone()
	for _, commit := range pr.Commits {
		s.related[commit] = appendUnique(s.related[commit], pr.Number)
	}
}

func (s *State) MarkMerged(number int, mergeCommit, shape string, relatedCommits []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pr, ok := s.prs[number]
	if !ok {
		panic(fmt.Sprintf("missing PR #%d", number))
	}
	pr.State = gitobj.PullRequestStateMerged
	pr.MergeCommit = mergeCommit
	pr.MergeShape = shape
	pr.RelatedCommits = append([]string(nil), relatedCommits...)
	s.prs[number] = pr
	for _, commit := range relatedCommits {
		s.related[commit] = appendUnique(s.related[commit], number)
	}
}

func (s *State) Must(number int) PR {
	s.mu.Lock()
	defer s.mu.Unlock()
	pr, ok := s.prs[number]
	if !ok {
		panic(fmt.Sprintf("missing PR #%d", number))
	}
	return pr.Clone()
}

func (s *State) RelatedPRNumbers(commit string) []int {
	s.mu.Lock()
	defer s.mu.Unlock()
	numbers := append([]int(nil), s.related[commit]...)
	sort.Ints(numbers)
	return numbers
}

type PR struct {
	Number         int
	Title          string
	State          gitobj.PullRequestState
	Base           string
	Head           string
	BaseSHA        string
	HeadSHA        string
	Commits        []string
	MergeCommit    string
	MergeShape     string
	RelatedCommits []string
}

func (pr PR) Clone() PR {
	pr.Commits = append([]string(nil), pr.Commits...)
	pr.RelatedCommits = append([]string(nil), pr.RelatedCommits...)
	return pr
}

func (pr PR) GitObject() gitobj.PullRequest {
	out := gitobj.PullRequest{
		Number:      pr.Number,
		Title:       pr.Title,
		Url:         fmt.Sprintf("https://example.invalid/owner/repo/pull/%d", pr.Number),
		State:       pr.State,
		BaseRefName: pr.Base,
		HeadRefName: pr.Head,
	}
	out.Author.Login = "test-user"
	out.MergeCommit.Sha = pr.MergeCommit
	for _, commit := range pr.Commits {
		out.Commits = append(out.Commits, struct {
			Oid string `json:"oid"`
		}{Oid: commit})
	}
	return out
}

func appendUnique(items []int, item int) []int {
	for _, existing := range items {
		if existing == item {
			return items
		}
	}
	return append(items, item)
}
