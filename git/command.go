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

	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

func NewCommand(cmd string, args ...string) *Command {
	return &Command{
		cmd:    cmd,
		args:   args,
		stdin:  &bytes.Buffer{},
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
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
	cmd.Stdin = c.stdin
	cmd.Stdout = c.stdout
	cmd.Stderr = c.stderr

	for _, mod := range mods {
		mod(c)
	}

	err = cmd.Run()
	if err != nil {
		switch c.cmd {
		case "git":
			ge := GitError{err: err}
			var exitError *exec.ExitError
			if errors.As(err, &exitError) {
				ge.Stderr = string(exitError.Stderr)
				ge.ExitCode = exitError.ExitCode()
			}
			return &ge

		case "gh":
			ge := GHError{err: err}
			var exitError *exec.ExitError
			if errors.As(err, &exitError) {
				ge.Stderr = string(exitError.Stderr)
				ge.ExitCode = exitError.ExitCode()
			}
			return &ge
		default:
			panic(fmt.Sprintf("unsupported command: %s", c.cmd))
		}
	}

	return nil
}

type CommandModifier func(*Command)

func WithStdout(stdout io.Writer) CommandModifier {
	return func(c *Command) {
		c.stdout = stdout
	}
}

func WithStdin(stdin io.Reader) CommandModifier {
	return func(c *Command) {
		c.stdin = stdin
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
