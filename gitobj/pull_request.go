package gitobj

import (
	"fmt"

	"github.com/134130/gh-cherry-pick/internal/color"
)

type PullRequestState string

const (
	PullRequestStateOpen   PullRequestState = "OPEN"
	PullRequestStateClosed PullRequestState = "CLOSED"
	PullRequestStateMerged PullRequestState = "MERGED"
)

type PullRequest struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Url    string `json:"url"`
	Author struct {
		Login string `json:"login"`
	} `json:"author"`
	State       PullRequestState `json:"state"`
	IsDraft     bool             `json:"isDraft"`
	MergeCommit struct {
		Sha string `json:"oid"`
	} `json:"mergeCommit"`
	BaseRefName string `json:"baseRefName"`
	HeadRefName string `json:"headRefName"`
}

func (pr PullRequest) StateString() string {
	switch pr.State {
	case PullRequestStateOpen:
		return color.Green("open")
	case PullRequestStateClosed:
		return color.Red("closed")
	case PullRequestStateMerged:
		return color.Purple("merged")
	default:
		if pr.IsDraft {
			return color.Grey("draft")
		}
		return "UNKNOWN"
	}
}

func (pr PullRequest) PRNumberString() string {
	return color.Cyan(fmt.Sprintf("#%d", pr.Number))
}
