package harness

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/134130/gitkit/gitcmd"
)

type Runner struct {
	workDir string
	state   *State

	mu       sync.Mutex
	commands []gitcmd.Command
}

func NewRunner(workDir string, state *State) *Runner {
	return &Runner{
		workDir: workDir,
		state:   state,
	}
}

var _ gitcmd.Runner = (*Runner)(nil)

func (r *Runner) Run(ctx context.Context, cmd gitcmd.Command) (gitcmd.Result, error) {
	r.record(cmd)
	cmd = r.withDefaultDir(cmd)
	if cmd.Program == gitcmd.ProgramGH {
		return r.runGH(ctx, cmd)
	}
	if isRemoteGetURL(cmd) {
		return gitcmd.Result{Command: cmd, Stdout: []byte("https://example.invalid/owner/repo.git\n")}, nil
	}
	return runGitCommand(ctx, cmd)
}

func (r *Runner) Start(ctx context.Context, cmd gitcmd.Command) (gitcmd.Process, error) {
	result, err := r.Run(ctx, cmd)
	return completedProcess{result: result, err: err}, nil
}

func (r *Runner) Commands() []gitcmd.Command {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]gitcmd.Command(nil), r.commands...)
}

func (r *Runner) CommandStrings() []string {
	commands := r.Commands()
	out := make([]string, 0, len(commands))
	for _, command := range commands {
		out = append(out, command.String())
	}
	return out
}

func (r *Runner) record(cmd gitcmd.Command) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.commands = append(r.commands, cmd)
}

func (r *Runner) withDefaultDir(cmd gitcmd.Command) gitcmd.Command {
	if cmd.Dir == "" {
		cmd.Dir = r.workDir
	}
	return cmd
}

func (r *Runner) runGH(ctx context.Context, cmd gitcmd.Command) (gitcmd.Result, error) {
	args := cmd.Args
	switch {
	case len(args) >= 4 && args[0] == "pr" && args[1] == "view":
		return r.ghPRView(cmd, args)
	case len(args) >= 3 && args[0] == "pr" && args[1] == "diff":
		return r.ghPRDiff(ctx, cmd, args)
	case len(args) >= 2 && args[0] == "repo" && args[1] == "view":
		return r.ghRepoView(cmd, args)
	case len(args) >= 2 && args[0] == "api":
		return r.ghAPI(ctx, cmd, args)
	default:
		return exitResult(cmd, fmt.Errorf("unsupported gh command: gh %s", strings.Join(args, " ")))
	}
}

func (r *Runner) ghPRView(cmd gitcmd.Command, args []string) (gitcmd.Result, error) {
	number, err := strconv.Atoi(args[2])
	if err != nil {
		return exitResult(cmd, fmt.Errorf("parse PR number %q: %w", args[2], err))
	}
	payload, err := json.Marshal(r.state.Must(number).GitObject())
	if err != nil {
		return exitResult(cmd, fmt.Errorf("marshal PR #%d: %w", number, err))
	}
	payload = append(payload, '\n')
	return gitcmd.Result{Command: cmd, Stdout: payload}, nil
}

func (r *Runner) ghPRDiff(ctx context.Context, cmd gitcmd.Command, args []string) (gitcmd.Result, error) {
	number, err := strconv.Atoi(args[2])
	if err != nil {
		return exitResult(cmd, fmt.Errorf("parse PR number %q: %w", args[2], err))
	}
	pr := r.state.Must(number)
	gitArgs := []string{"diff", pr.BaseSHA + ".." + pr.HeadSHA}
	if len(args) == 4 && args[3] == "--patch" {
		gitArgs = []string{"format-patch", "--stdout", pr.BaseSHA + ".." + pr.HeadSHA}
	}
	result, err := runGitCommand(ctx, gitcmd.GitCommand(gitArgs...).InDir(cmd.Dir))
	result.Command = cmd
	return result, err
}

func (r *Runner) ghRepoView(cmd gitcmd.Command, args []string) (gitcmd.Result, error) {
	if hasArg(args, "--jq", ".nameWithOwner") {
		return gitcmd.Result{Command: cmd, Stdout: []byte("owner/repo\n")}, nil
	}
	return gitcmd.Result{Command: cmd, Stdout: []byte(`{"nameWithOwner":"owner/repo"}` + "\n")}, nil
}

func (r *Runner) ghAPI(ctx context.Context, cmd gitcmd.Command, args []string) (gitcmd.Result, error) {
	endpoint := apiEndpoint(args)
	jq := valueAfter(args, "--jq")
	if endpoint == "" {
		return exitResult(cmd, fmt.Errorf("missing API endpoint"))
	}

	if strings.HasSuffix(endpoint, "/pulls") {
		commit := path.Base(strings.TrimSuffix(endpoint, "/pulls"))
		var out strings.Builder
		for _, number := range r.state.RelatedPRNumbers(commit) {
			out.WriteString(strconv.Itoa(number))
			out.WriteByte('\n')
		}
		return gitcmd.Result{Command: cmd, Stdout: []byte(out.String())}, nil
	}

	commit := path.Base(endpoint)
	parents, err := gitParents(ctx, cmd.Dir, commit)
	if err != nil {
		return exitResult(cmd, err)
	}
	switch jq {
	case ".parents | length":
		return gitcmd.Result{Command: cmd, Stdout: []byte(fmt.Sprintf("%d\n", len(parents)))}, nil
	case ".parents[0].sha":
		if len(parents) == 0 {
			return exitResult(cmd, fmt.Errorf("commit %s has no parents", commit))
		}
		return gitcmd.Result{Command: cmd, Stdout: []byte(parents[0] + "\n")}, nil
	default:
		return exitResult(cmd, fmt.Errorf("unsupported gh api jq %q for %s", jq, endpoint))
	}
}

func apiEndpoint(args []string) string {
	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--hostname", "--jq", "-H":
			i++
			continue
		}
		if !strings.HasPrefix(arg, "-") {
			return arg
		}
	}
	return ""
}

func runGitCommand(ctx context.Context, cmd gitcmd.Command) (gitcmd.Result, error) {
	execCmd := exec.CommandContext(ctx, "git", cmd.Args...)
	execCmd.Dir = cmd.Dir
	execCmd.Env = sanitizedEnv(cmd.Env...)
	execCmd.Stdin = cmd.Stdin

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	execCmd.Stdout = stdout
	execCmd.Stderr = stderr
	if cmd.Stdout != nil {
		execCmd.Stdout = io.MultiWriter(stdout, cmd.Stdout)
	}
	if cmd.Stderr != nil {
		execCmd.Stderr = io.MultiWriter(stderr, cmd.Stderr)
	}

	err := execCmd.Run()
	result := gitcmd.Result{
		Command: cmd,
		Stdout:  stdout.Bytes(),
		Stderr:  stderr.Bytes(),
	}
	if err == nil {
		return result, nil
	}
	result.ExitCode = 1
	if exitError, ok := err.(*exec.ExitError); ok {
		result.ExitCode = exitError.ExitCode()
	}
	return result, &gitcmd.ExitError{Result: result, Err: err}
}

func gitParents(ctx context.Context, dir, commit string) ([]string, error) {
	result, err := runGitCommand(ctx, gitcmd.GitCommand("rev-list", "--parents", "-n", "1", commit).InDir(dir))
	if err != nil {
		return nil, err
	}
	fields := strings.Fields(result.StdoutString())
	if len(fields) <= 1 {
		return nil, nil
	}
	return fields[1:], nil
}

func sanitizedEnv(extra ...string) []string {
	blocked := map[string]struct{}{
		"GIT_DIR":        {},
		"GIT_INDEX_FILE": {},
		"GIT_WORK_TREE":  {},
	}
	env := make([]string, 0, len(os.Environ())+len(extra))
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if _, ok := blocked[key]; ok {
			continue
		}
		env = append(env, entry)
	}
	return append(env, extra...)
}

func exitResult(cmd gitcmd.Command, err error) (gitcmd.Result, error) {
	result := gitcmd.Result{
		Command:  cmd,
		Stderr:   []byte(err.Error()),
		ExitCode: 1,
	}
	return result, &gitcmd.ExitError{Result: result, Err: err}
}

type completedProcess struct {
	result gitcmd.Result
	err    error
}

func (p completedProcess) Wait() (gitcmd.Result, error) {
	return p.result, p.err
}

func isRemoteGetURL(cmd gitcmd.Command) bool {
	return cmd.Program == gitcmd.ProgramGit &&
		len(cmd.Args) == 3 &&
		cmd.Args[0] == "remote" &&
		cmd.Args[1] == "get-url" &&
		cmd.Args[2] == defaultRemote
}

func hasArg(args []string, flag, value string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == flag && args[i+1] == value {
			return true
		}
	}
	return false
}

func valueAfter(args []string, flag string) string {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == flag {
			return args[i+1]
		}
	}
	return ""
}
