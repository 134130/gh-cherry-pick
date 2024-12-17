package gitobj

import (
	"fmt"

	"github.com/134130/gh-cherry-pick/internal/tui"
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
	State   PullRequestState `json:"state"`
	IsDraft bool             `json:"isDraft"`
}

func (pr PullRequest) StateString() string {
	switch pr.State {
	case PullRequestStateOpen:
		return tui.Green("open")
	case PullRequestStateClosed:
		return tui.Red("closed")
	case PullRequestStateMerged:
		return tui.Purple("merged")
	default:
		if pr.IsDraft {
			return tui.Grey("draft")
		}
		return "UNKNOWN"
	}
}

func (pr PullRequest) PRNumberString() string {
	return tui.Cyan(fmt.Sprintf("#%d", pr.Number))
}
