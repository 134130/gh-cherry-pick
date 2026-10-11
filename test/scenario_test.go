package test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/134130/gh-cherry-pick/internal/app"
	"github.com/134130/gh-cherry-pick/test/harness"
)

func TestHarnessAppliesSquashMergedPR(t *testing.T) {
	h := harness.New(t)
	h.Branch("feature", "origin/main")
	h.Commit("feature", "feature.txt", "feature\n")
	h.Push("feature")
	h.OpenPR(1, "feature", "main", "feature")
	h.Branch("target", "origin/main")
	h.SquashMerge(1)
	h.Checkout("target")
	remoteRefs := h.RemoteRefs()

	plan := h.Plan(app.PlanOptions{PRNumber: 1})
	if plan.Method != app.MethodSquash {
		t.Fatalf("method mismatch: want %s, got %s", app.MethodSquash, plan.Method)
	}
	result, err := h.Execute(plan)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	assertCurrentBranch(t, h, "target")
	assertClean(t, h)
	assertFile(t, h, "feature.txt", "feature\n")
	assertRemoteRefs(t, h, remoteRefs)
	if len(result.CommitSHAs) != 1 {
		t.Fatalf("expected 1 created commit, got %#v", result.CommitSHAs)
	}
}

func TestHarnessStreamsCommitHookOutput(t *testing.T) {
	h := harness.New(t)
	h.Branch("feature", "origin/main")
	h.Commit("feature", "feature.txt", "feature\n")
	h.Push("feature")
	h.OpenPR(6, "feature", "main", "feature")
	h.Branch("target", "origin/main")
	h.SquashMerge(6)
	h.Checkout("target")

	hook := filepath.Join(h.WorkDir, ".git", "hooks", "pre-commit")
	if err := os.WriteFile(hook, []byte("#!/bin/sh\necho hook-stdout\necho hook-stderr >&2\n"), 0755); err != nil {
		t.Fatalf("write pre-commit hook: %v", err)
	}

	plan := h.Plan(app.PlanOptions{PRNumber: 6})
	var output bytes.Buffer
	if _, err := app.NewGitKitExecutor(h.Runner(), &output).Execute(t.Context(), plan); err != nil {
		t.Fatalf("execute: %v", err)
	}
	for _, want := range []string{"hook-stdout", "hook-stderr"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("expected %q in streamed hook output %q", want, output.String())
		}
	}
}

func TestHarnessAppliesRebaseMergedPR(t *testing.T) {
	h := harness.New(t)
	h.Branch("feature", "origin/main")
	h.Commit("feature", "one.txt", "one\n")
	h.Commit("feature", "two.txt", "two\n")
	h.Push("feature")
	h.OpenPR(2, "feature", "main", "feature")
	h.Branch("target", "origin/main")
	h.RebaseMerge(2)
	h.Checkout("target")

	plan := h.Plan(app.PlanOptions{PRNumber: 2})
	if plan.Method != app.MethodPatch {
		t.Fatalf("method mismatch: want %s, got %s", app.MethodPatch, plan.Method)
	}
	result, err := h.Execute(plan)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	assertCurrentBranch(t, h, "target")
	assertClean(t, h)
	assertFile(t, h, "one.txt", "one\n")
	assertFile(t, h, "two.txt", "two\n")
	if len(result.CommitSHAs) != 2 {
		t.Fatalf("expected 2 created commits, got %#v", result.CommitSHAs)
	}
}

func TestHarnessAppliesMergeCommitPR(t *testing.T) {
	h := harness.New(t)
	h.Branch("feature", "origin/main")
	h.Commit("feature", "feature.txt", "feature\n")
	h.Push("feature")
	h.OpenPR(3, "feature", "main", "feature")
	h.Branch("target", "origin/main")
	h.MergeCommit(3)
	h.Checkout("target")

	plan := h.Plan(app.PlanOptions{PRNumber: 3})
	if plan.Method != app.MethodMerge {
		t.Fatalf("method mismatch: want %s, got %s", app.MethodMerge, plan.Method)
	}
	result, err := h.Execute(plan)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	assertCurrentBranch(t, h, "target")
	assertClean(t, h)
	assertFile(t, h, "feature.txt", "feature\n")
	if len(result.CommitSHAs) != 1 {
		t.Fatalf("expected 1 created commit, got %#v", result.CommitSHAs)
	}
}

func TestHarnessReportsConflict(t *testing.T) {
	h := harness.New(t)
	h.Commit("main", "conflict.txt", "base\n")
	h.Push("main")
	h.Branch("feature", "origin/main")
	h.Commit("feature", "conflict.txt", "feature\n")
	h.Push("feature")
	h.OpenPR(4, "feature", "main", "feature")
	h.Branch("target", "origin/main")
	h.Commit("target", "conflict.txt", "target\n")
	h.SquashMerge(4)
	h.Checkout("target")
	remoteRefs := h.RemoteRefs()

	_, err := h.Execute(h.Plan(app.PlanOptions{PRNumber: 4}))
	if err == nil {
		t.Fatalf("expected conflict")
	}
	if !strings.Contains(err.Error(), "Resolve the conflict") {
		t.Fatalf("expected conflict guidance, got %v", err)
	}
	assertCurrentBranch(t, h, "target")
	assertRemoteRefs(t, h, remoteRefs)
	if status := h.StatusPorcelain(); !strings.Contains(status, "conflict.txt") {
		t.Fatalf("expected conflict status for conflict.txt, got %q", status)
	}
}

func TestHarnessBlocksDirtyWorktree(t *testing.T) {
	h := harness.New(t)
	h.Branch("feature", "origin/main")
	h.Commit("feature", "feature.txt", "feature\n")
	h.Push("feature")
	h.OpenPR(5, "feature", "main", "feature")
	h.Branch("target", "origin/main")
	h.SquashMerge(5)
	h.Checkout("target")
	h.DirtyFile("dirty.txt", "dirty\n")

	plan := h.Plan(app.PlanOptions{PRNumber: 5})
	if !plan.Blocked() {
		t.Fatalf("expected blocked plan")
	}
	if !strings.Contains(plan.BlockedReason, "dirty") {
		t.Fatalf("unexpected blocked reason: %s", plan.BlockedReason)
	}
	_, err := h.Execute(plan)
	if err == nil {
		t.Fatalf("expected executor to reject blocked plan")
	}
}

func assertCurrentBranch(t *testing.T, h *harness.Harness, want string) {
	t.Helper()
	if got := h.CurrentBranch(); got != want {
		t.Fatalf("branch mismatch: want %q, got %q", want, got)
	}
}

func assertClean(t *testing.T, h *harness.Harness) {
	t.Helper()
	if status := h.StatusPorcelain(); status != "" {
		t.Fatalf("expected clean worktree, got %q", status)
	}
}

func assertFile(t *testing.T, h *harness.Harness, path, want string) {
	t.Helper()
	if got := h.File(path); got != want {
		t.Fatalf("%s mismatch: want %q, got %q", path, want, got)
	}
}

func assertRemoteRefs(t *testing.T, h *harness.Harness, want string) {
	t.Helper()
	if got := h.RemoteRefs(); got != want {
		t.Fatalf("remote refs changed\nwant:\n%s\n got:\n%s", want, got)
	}
}
