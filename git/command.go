package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/cli/safeexec"
)

type Command struct {
	cmd  string
	args []string
}

func NewCommand(cmd string, args ...string) *Command {
	return &Command{
		cmd:  cmd,
		args: args,
	}
}

func (c *Command) Run(ctx context.Context, mods ...CommandModifier) error {
	exe, err := path(c.cmd)
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return &NotInstalledError{
				message: fmt.Sprintf("unabled to find %s executable in PATH; please install %s before retrying", c.cmd, c.cmd),
				err:     err,
			}
		}
		return err
	}

	cmd := exec.CommandContext(ctx, exe, c.args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	for _, mod := range mods {
		mod(cmd)
	}

	err = cmd.Run()
	if err != nil {
		switch c.cmd {
		case "git":
			ge := GitError{err: err}
			var exitError *exec.ExitError
			if errors.As(err, &exitError) {
				ge.Stderr = stderr.String()
				ge.ExitCode = exitError.ExitCode()
			}
			return &ge

		case "gh":
			ge := GHError{err: err}
			var exitError *exec.ExitError
			if errors.As(err, &exitError) {
				ge.Stderr = stderr.String()
				ge.ExitCode = exitError.ExitCode()
			}
			return &ge
		default:
			panic(fmt.Sprintf("unsupported command: %s", c.cmd))
		}
	}

	return nil
}

type CommandModifier func(c *exec.Cmd)

func WithStdout(stdout io.Writer) CommandModifier {
	return func(c *exec.Cmd) {
		c.Stdout = stdout
	}
}

func WithStdin(stdin io.Reader) CommandModifier {
	return func(c *exec.Cmd) {
		c.Stdin = stdin
	}
}

func path(cmd string) (string, error) {
	switch cmd {
	case "git":
		return safeexec.LookPath("git")
	case "gh":
		if ghExe := os.Getenv("GH_PATH"); ghExe != "" {
			return ghExe, nil
		}
		return safeexec.LookPath("gh")
	}

	return "", fmt.Errorf("unsupported command: %s", cmd)
}
