package gitkitstore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/134130/gh-cherry-pick/gitobj"
	"github.com/134130/gh-cherry-pick/internal/app"
	"github.com/134130/gitkit/ghcli"
	"github.com/134130/gitkit/ghpr"
	"github.com/134130/gitkit/gitcmd"
	"github.com/134130/gitkit/gitrepo"
)

const pullRequestFields = "number,title,url,author,state,isDraft,mergeCommit,baseRefName,headRefName,commits"

type Store struct {
	git gitrepo.Client
	gh  ghcli.Client
	pr  ghpr.Client
}

func New(r gitcmd.Runner) Store {
	git := gitrepo.New(r)
	gh := ghcli.New(r)
	return Store{
		git: git,
		gh:  gh,
		pr:  ghpr.New(gh, git),
	}
}

func (s Store) CurrentBranch(ctx context.Context) (string, error) {
	return s.git.CurrentBranch(ctx)
}

func (s Store) HeadSHA(ctx context.Context) (string, error) {
	return s.git.RevParse(ctx, "HEAD")
}

func (s Store) State(ctx context.Context) (app.RepositoryState, error) {
	state, err := s.git.State(ctx)
	if err != nil {
		return app.RepositoryState{}, err
	}
	return app.RepositoryState{
		Dirty:         state.Dirty,
		Rebasing:      state.Rebasing,
		ApplyingPatch: state.ApplyingPatch,
		CherryPicking: state.CherryPicking,
		Merging:       state.Merging,
	}, nil
}

func (s Store) PullRequest(ctx context.Context, number int) (gitobj.PullRequest, error) {
	out, err := s.gh.OutputBytes(ctx, "pr", "view", fmt.Sprint(number), "--json", pullRequestFields)
	if err != nil {
		return gitobj.PullRequest{}, err
	}
	var pr gitobj.PullRequest
	if err := json.NewDecoder(bytes.NewReader(out)).Decode(&pr); err != nil {
		return gitobj.PullRequest{}, fmt.Errorf("decode pull request: %w", err)
	}
	return pr, nil
}

func (s Store) MergeShape(ctx context.Context, pr gitobj.PullRequest) (string, error) {
	shape, err := s.pr.MergeStrategy(ctx, pr.Number, pr.MergeCommit.Sha)
	if err != nil {
		return "", err
	}
	return string(shape), nil
}

var _ app.RepositoryStore = Store{}
