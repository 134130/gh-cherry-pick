package git

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/cli/safeexec"
)

type Git struct {
	Stdin *bytes.Buffer
}

func (git *Git) RunWithContext(ctx context.Context, arg ...string) (string, error) {
	gitPath, err := safeexec.LookPath("git")
	if err != nil {
		return "", fmt.Errorf("git not found in PATH: %w", err)
	}

	var stdout bytes.Buffer
	cmd := exec.CommandContext(ctx, gitPath, arg...)
	if git.Stdin != nil {
		cmd.Stdin = git.Stdin
	}
	cmd.Stdout = &stdout

	if err = cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to run git %v: %w", arg, err)
	}

	return strings.TrimSpace(stdout.String()), nil
}

func IsDirty(ctx context.Context) (bool, error) {
	stdout, err := (&Git{}).RunWithContext(ctx, "status", "--porcelain")
	if err != nil {
		return false, fmt.Errorf("failed to check if the repository is dirty: %w", err)
	}

	return len(stdout) > 0, nil
}

func IsInRebaseOrAm(ctx context.Context) (bool, error) {
	repoRoot, err := getRepoRoot(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to check if the repository is in a rebase or am: %w", err)
	}

	var rebaseMagicFile = fmt.Sprintf("%s/.git/rebase-apply", repoRoot)
	if _, err := os.Stat(rebaseMagicFile); err == nil {
		return true, nil
	} else if !os.IsNotExist(err) {
		return false, fmt.Errorf("failed to check if the repository is in a rebase or am: %w", err)
	}

	return false, nil
}

func getRepoRoot(ctx context.Context) (string, error) {
	stdout, err := (&Git{}).RunWithContext(ctx, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("failed to resolve the repository root: %w", err)
	}

	return stdout, nil
}
