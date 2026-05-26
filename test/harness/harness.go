package harness

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/134130/gh-cherry-pick/internal/app"
	"github.com/134130/gh-cherry-pick/internal/app/gitkitstore"
)

const (
	defaultBranch = "main"
	defaultRemote = "origin"
)

type Harness struct {
	t         testing.TB
	ctx       context.Context
	state     *State
	runner    *Runner
	RootDir   string
	RemoteDir string
	WorkDir   string
}

func New(t testing.TB) *Harness {
	t.Helper()

	ctx := context.Background()
	if tb, ok := t.(interface{ Context() context.Context }); ok {
		ctx = tb.Context()
	}

	root := t.TempDir()
	h := &Harness{
		t:         t,
		ctx:       ctx,
		state:     NewState(),
		RootDir:   root,
		RemoteDir: filepath.Join(root, "origin.git"),
		WorkDir:   filepath.Join(root, "work"),
	}

	h.git("", "init", "--bare", "--initial-branch="+defaultBranch, h.RemoteDir)
	h.git("", "clone", h.RemoteDir, h.WorkDir)
	h.git(h.WorkDir, "config", "user.name", "gh-cherry-pick test")
	h.git(h.WorkDir, "config", "user.email", "gh-cherry-pick@example.invalid")
	h.writeFile("README.md", "# test repository\n")
	h.git(h.WorkDir, "add", "README.md")
	h.git(h.WorkDir, "commit", "-m", "Initial commit")
	h.git(h.WorkDir, "push", "--set-upstream", defaultRemote, defaultBranch)
	h.Fetch()

	return h
}

func (h *Harness) Runner() *Runner {
	h.t.Helper()
	if h.runner == nil {
		h.runner = NewRunner(h.WorkDir, h.state)
	}
	return h.runner
}

func (h *Harness) Plan(opts app.PlanOptions) app.Plan {
	h.t.Helper()
	plan, err := app.NewPlanner(gitkitstore.New(h.Runner())).BuildPlan(h.ctx, opts)
	if err != nil {
		h.t.Fatalf("build plan: %v", err)
	}
	return plan
}

func (h *Harness) Execute(plan app.Plan) (app.RunResult, error) {
	h.t.Helper()
	return app.NewGitKitExecutor(h.Runner()).Execute(h.ctx, plan)
}

func (h *Harness) Branch(name, startPoint string) *Harness {
	h.t.Helper()
	h.Fetch()
	h.git(h.WorkDir, "switch", "-C", name, startPoint)
	return h
}

func (h *Harness) Checkout(branch string) *Harness {
	h.t.Helper()
	h.git(h.WorkDir, "switch", branch)
	return h
}

func (h *Harness) Commit(branch, path, content string) string {
	h.t.Helper()
	h.git(h.WorkDir, "switch", branch)
	h.writeFile(path, content)
	h.git(h.WorkDir, "add", path)
	h.git(h.WorkDir, "commit", "-m", fmt.Sprintf("%s: %s", branch, path))
	return h.Ref("HEAD")
}

func (h *Harness) Push(branch string) *Harness {
	h.t.Helper()
	h.git(h.WorkDir, "push", "--set-upstream", defaultRemote, branch)
	h.Fetch()
	return h
}

func (h *Harness) Fetch() *Harness {
	h.t.Helper()
	h.git(h.WorkDir, "fetch", defaultRemote)
	return h
}

func (h *Harness) OpenPR(number int, title, base, head string) *Harness {
	h.t.Helper()
	h.Fetch()
	baseRef := remoteRef(base)
	headRef := remoteRef(head)
	pr := PR{
		Number:  number,
		Title:   title,
		State:   "OPEN",
		Base:    base,
		Head:    head,
		BaseSHA: h.Ref(baseRef),
		HeadSHA: h.Ref(headRef),
		Commits: h.RevList(baseRef + ".." + headRef),
	}
	h.state.Upsert(pr)
	return h
}

func (h *Harness) SquashMerge(number int) *Harness {
	h.t.Helper()
	pr := h.state.Must(number)
	h.checkoutRemote(pr.Base)
	h.git(h.WorkDir, "merge", "--squash", remoteRef(pr.Head))
	h.git(h.WorkDir, "commit", "-m", fmt.Sprintf("Squash PR #%d", number))
	mergeSHA := h.Ref("HEAD")
	h.git(h.WorkDir, "push", defaultRemote, pr.Base)
	h.Fetch()
	h.state.MarkMerged(number, mergeSHA, MergeShapeSquash, nil)
	return h
}

func (h *Harness) RebaseMerge(number int) *Harness {
	h.t.Helper()
	pr := h.state.Must(number)
	h.checkoutRemote(pr.Base)
	rebased := make([]string, 0, len(pr.Commits))
	for _, commit := range pr.Commits {
		h.git(h.WorkDir, "cherry-pick", commit)
		rebased = append(rebased, h.Ref("HEAD"))
	}
	mergeSHA := h.Ref("HEAD")
	h.git(h.WorkDir, "push", defaultRemote, pr.Base)
	h.Fetch()
	h.state.MarkMerged(number, mergeSHA, MergeShapeRebase, rebased)
	return h
}

func (h *Harness) MergeCommit(number int) *Harness {
	h.t.Helper()
	pr := h.state.Must(number)
	h.checkoutRemote(pr.Base)
	h.git(h.WorkDir, "merge", "--no-ff", "--no-edit", remoteRef(pr.Head))
	mergeSHA := h.Ref("HEAD")
	h.git(h.WorkDir, "push", defaultRemote, pr.Base)
	h.Fetch()
	h.state.MarkMerged(number, mergeSHA, MergeShapeMergeCommit, nil)
	return h
}

func (h *Harness) CurrentBranch() string {
	h.t.Helper()
	return h.gitOutput(h.WorkDir, "branch", "--show-current")
}

func (h *Harness) Ref(ref string) string {
	h.t.Helper()
	return h.gitOutput(h.WorkDir, "rev-parse", ref)
}

func (h *Harness) RevList(revRange string) []string {
	h.t.Helper()
	out := h.gitOutput(h.WorkDir, "rev-list", "--reverse", revRange)
	if out == "" {
		return nil
	}
	return strings.Fields(out)
}

func (h *Harness) File(path string) string {
	h.t.Helper()
	content, err := os.ReadFile(filepath.Join(h.WorkDir, path))
	if err != nil {
		h.t.Fatalf("read %s: %v", path, err)
	}
	return string(content)
}

func (h *Harness) DirtyFile(path, content string) *Harness {
	h.t.Helper()
	h.writeFile(path, content)
	return h
}

func (h *Harness) StatusPorcelain() string {
	h.t.Helper()
	return h.gitOutput(h.WorkDir, "status", "--porcelain")
}

func (h *Harness) RemoteRefs() string {
	h.t.Helper()
	return h.gitOutput(h.WorkDir, "ls-remote", defaultRemote)
}

func (h *Harness) checkoutRemote(branch string) {
	h.t.Helper()
	h.Fetch()
	h.git(h.WorkDir, "switch", "-C", branch, remoteRef(branch))
}

func (h *Harness) writeFile(path, content string) {
	h.t.Helper()
	fullPath := filepath.Join(h.WorkDir, path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		h.t.Fatalf("mkdir %s: %v", filepath.Dir(fullPath), err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		h.t.Fatalf("write %s: %v", path, err)
	}
}

func (h *Harness) gitOutput(dir string, args ...string) string {
	h.t.Helper()
	stdout, stderr, err := runGit(h.ctx, dir, args...)
	if err != nil {
		h.t.Fatalf("git %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout, stderr)
	}
	return strings.TrimSpace(stdout)
}

func (h *Harness) git(dir string, args ...string) {
	h.t.Helper()
	stdout, stderr, err := runGit(h.ctx, dir, args...)
	if err != nil {
		h.t.Fatalf("git %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout, stderr)
	}
}

func runGit(ctx context.Context, dir string, args ...string) (string, string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = sanitizedEnv()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func remoteRef(branch string) string {
	if strings.HasPrefix(branch, defaultRemote+"/") {
		return branch
	}
	return defaultRemote + "/" + branch
}
