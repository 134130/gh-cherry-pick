package app

import (
	"context"
	"reflect"
	"testing"

	"github.com/134130/gh-cherry-pick/gitobj"
)

func TestPlannerSelectsAutoMethods(t *testing.T) {
	tests := []struct {
		name  string
		shape string
		want  ApplyMethod
	}{{
		name:  "squash",
		shape: "squash",
		want:  MethodSquash,
	}, {
		name:  "rebase",
		shape: "rebase",
		want:  MethodPatch,
	}, {
		name:  "merge commit",
		shape: "merge_commit",
		want:  MethodMerge,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newFakeStore()
			store.mergeShape = tt.shape
			plan, err := NewPlanner(store).BuildPlan(context.Background(), PlanOptions{PRNumber: 123, Method: MethodAuto})
			if err != nil {
				t.Fatalf("BuildPlan returned error: %v", err)
			}
			if plan.Method != tt.want {
				t.Fatalf("method mismatch: want %s, got %s", tt.want, plan.Method)
			}
		})
	}
}

func TestPlannerExplicitMethodOverridesAuto(t *testing.T) {
	store := newFakeStore()
	store.mergeShape = "rebase"

	plan, err := NewPlanner(store).BuildPlan(context.Background(), PlanOptions{PRNumber: 123, Method: MethodCommits})
	if err != nil {
		t.Fatalf("BuildPlan returned error: %v", err)
	}
	if plan.Method != MethodCommits {
		t.Fatalf("method mismatch: %s", plan.Method)
	}
	if store.mergeShapeCalled {
		t.Fatalf("did not expect merge shape lookup")
	}
}

func TestPlannerBlocksDirtyRepository(t *testing.T) {
	store := newFakeStore()
	store.state.Dirty = true

	plan, err := NewPlanner(store).BuildPlan(context.Background(), PlanOptions{PRNumber: 123})
	if err != nil {
		t.Fatalf("BuildPlan returned error: %v", err)
	}
	if !plan.Blocked() || plan.BlockedReason == "" {
		t.Fatalf("expected blocked plan")
	}
	if len(plan.Commands) != 0 {
		t.Fatalf("expected no commands for blocked plan")
	}
}

func TestPlannerBuildsCommitCommandsInOrder(t *testing.T) {
	store := newFakeStore()
	store.pr.Commits = []struct {
		Oid string `json:"oid"`
	}{{Oid: "aaa111"}, {Oid: "bbb222"}}

	plan, err := NewPlanner(store).BuildPlan(context.Background(), PlanOptions{PRNumber: 123, Method: MethodCommits})
	if err != nil {
		t.Fatalf("BuildPlan returned error: %v", err)
	}
	got := commandStrings(plan.Commands)
	want := []string{"git cherry-pick aaa111", "git cherry-pick bbb222"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("commands mismatch\nwant: %#v\n got: %#v", want, got)
	}
}

func TestPlannerBlocksOpenPR(t *testing.T) {
	store := newFakeStore()
	store.pr.State = gitobj.PullRequestStateOpen

	plan, err := NewPlanner(store).BuildPlan(context.Background(), PlanOptions{PRNumber: 123})
	if err != nil {
		t.Fatalf("BuildPlan returned error: %v", err)
	}
	if !plan.Blocked() {
		t.Fatalf("expected open PR to be blocked")
	}
}

type fakeStore struct {
	branch           string
	head             string
	state            RepositoryState
	pr               gitobj.PullRequest
	mergeShape       string
	mergeShapeCalled bool
}

func newFakeStore() *fakeStore {
	store := &fakeStore{
		branch:     "feature",
		head:       "abc123",
		mergeShape: "squash",
	}
	store.pr.Number = 123
	store.pr.Title = "Fix bug"
	store.pr.State = gitobj.PullRequestStateMerged
	store.pr.MergeCommit.Sha = "merge123"
	return store
}

func (s *fakeStore) CurrentBranch(context.Context) (string, error) {
	return s.branch, nil
}

func (s *fakeStore) HeadSHA(context.Context) (string, error) {
	return s.head, nil
}

func (s *fakeStore) State(context.Context) (RepositoryState, error) {
	return s.state, nil
}

func (s *fakeStore) PullRequest(context.Context, int) (gitobj.PullRequest, error) {
	return s.pr, nil
}

func (s *fakeStore) MergeShape(context.Context, gitobj.PullRequest) (string, error) {
	s.mergeShapeCalled = true
	return s.mergeShape, nil
}

func commandStrings(commands []Command) []string {
	out := make([]string, 0, len(commands))
	for _, command := range commands {
		line := command.Program
		for _, arg := range command.Args {
			line += " " + arg
		}
		out = append(out, line)
	}
	return out
}
