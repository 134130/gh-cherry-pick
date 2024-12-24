package git

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func setup(ctx context.Context) func() {
	if _, err := os.Stat(".test"); errors.Is(err, os.ErrNotExist) {
		if err = os.Mkdir(".test", os.ModePerm); err != nil {
			panic(err)
		}
	} else if err != nil {
		panic(err)
	}

	if err := os.Chdir(".test"); err != nil {
		panic(err)
	}

	if _, err := os.Stat("test-cherry-pick"); !errors.Is(err, os.ErrNotExist) {
		if err = os.RemoveAll("test-cherry-pick"); err != nil {
			panic(err)
		}
	}

	args := []string{"clone", "https://github.com/134130/test-cherry-pick.git"}
	if err := exec.CommandContext(ctx, "git", args...).Run(); err != nil {
		panic(err)
	}

	if err := os.Chdir("test-cherry-pick"); err != nil {
		panic(err)
	}

	return func() {
		if err := os.Chdir("../.."); err != nil {
			panic(err)
		}

		if err := os.RemoveAll(".test"); err != nil {
			panic(err)
		}
	}
}

func ptr[T any](s T) *T {
	return &s
}

func TestRunWithContext(t *testing.T) {
	ctx := context.Background()

	testcases := []struct {
		name     string
		prNumber int
		onTo     string
		error    *string
	}{{
		name:     "squash merged PR",
		prNumber: 4,
		onTo:     "release/10.0",
		error:    nil,
	}, {
		name:     "rebase merged PR",
		prNumber: 5,
		onTo:     "release/10.0",
		error:    nil,
	}, {
		name:     "will be conflicted PR",
		prNumber: 7,
		onTo:     "release/10.0",
		error:    ptr("resolve the conflicts"),
	}}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			d := setup(ctx)
			defer d()

			cherryPick := CherryPick{
				PRNumber:      tc.prNumber,
				OnTo:          tc.onTo,
				MergeStrategy: MergeStrategyAuto,
				Push:          false,
			}

			err := cherryPick.RunWithContext(ctx)
			if tc.error == nil {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error: %s", *tc.error)
				} else if !strings.Contains(err.Error(), *tc.error) {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}
