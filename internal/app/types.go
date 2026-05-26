package app

import (
	"context"
	"fmt"

	"github.com/134130/gh-cherry-pick/gitobj"
)

type ApplyMethod string

const (
	MethodAuto    ApplyMethod = "auto"
	MethodPatch   ApplyMethod = "patch"
	MethodCommits ApplyMethod = "commits"
	MethodSquash  ApplyMethod = "squash"
	MethodMerge   ApplyMethod = "merge"
)

func (m ApplyMethod) Validate() error {
	switch m {
	case MethodAuto, MethodPatch, MethodCommits, MethodSquash, MethodMerge:
		return nil
	default:
		return fmt.Errorf("unsupported method: %s", m)
	}
}

type RepositoryState struct {
	Dirty         bool
	Rebasing      bool
	ApplyingPatch bool
	CherryPicking bool
	Merging       bool
}

func (s RepositoryState) Busy() bool {
	return s.Rebasing || s.ApplyingPatch || s.CherryPicking || s.Merging
}

type Destination struct {
	Branch string
	Head   string
}

type Command struct {
	Program string
	Args    []string
}

type Plan struct {
	PR              gitobj.PullRequest
	Destination     Destination
	RequestedMethod ApplyMethod
	Method          ApplyMethod
	MergeShape      string
	Commands        []Command
	State           RepositoryState
	BlockedReason   string
}

func (p Plan) Blocked() bool {
	return p.BlockedReason != ""
}

type PlanOptions struct {
	PRNumber int
	Method   ApplyMethod
}

func (o PlanOptions) normalized() PlanOptions {
	if o.Method == "" {
		o.Method = MethodAuto
	}
	return o
}

type RunResult struct {
	Plan       Plan
	CommitSHAs []string
}

type RepositoryStore interface {
	CurrentBranch(ctx context.Context) (string, error)
	HeadSHA(ctx context.Context) (string, error)
	State(ctx context.Context) (RepositoryState, error)
	PullRequest(ctx context.Context, number int) (gitobj.PullRequest, error)
	MergeShape(ctx context.Context, pr gitobj.PullRequest) (string, error)
}

type Executor interface {
	Execute(ctx context.Context, plan Plan) (RunResult, error)
}
