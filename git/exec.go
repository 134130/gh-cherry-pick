package git

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/cli/safeexec"
)

func ExecContext(ctx context.Context, cmd string, args ...string) (stdout, stderr bytes.Buffer, err error) {
	exe, err := Path(cmd)
	if err != nil {
		return
	}
	err = run(ctx, exe, nil, nil, &stdout, &stderr, args)
	return
}

func ExecContextWithStdin(ctx context.Context, stdin io.Reader, cmd string, args ...string) (stdout, stderr bytes.Buffer, err error) {
	exe, err := Path(cmd)
	if err != nil {
		return
	}
	err = run(ctx, exe, nil, stdin, &stdout, &stderr, args)
	return
}

func Path(cmd string) (string, error) {
	switch cmd {
	case "git":
		return safeexec.LookPath("git")
	case "gh":
		if ghExe := os.Getenv("GH_PATH"); ghExe != "" {
			return ghExe, nil
		}
		return safeexec.LookPath("gh")
	}

	return "", fmt.Errorf("unknown command: %s", cmd)
}

func run(ctx context.Context, exe string, env []string, stdin io.Reader, stdout, stderr io.Writer, args []string) error {
	cmd := exec.CommandContext(ctx, exe, args...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if env != nil {
		cmd.Env = env
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s execution failed: %w", exe, err)
	}
	return nil
}
