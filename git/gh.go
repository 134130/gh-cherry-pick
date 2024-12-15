package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/cli/safeexec"
)

type GH struct {
	Stdin *bytes.Buffer
}

func (gh *GH) RunWithContext(ctx context.Context, arg ...string) (string, error) {
	ghPath, err := safeexec.LookPath("gh")
	if err != nil {
		return "", fmt.Errorf("gh not found in PATH: %w", err)
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, ghPath, arg...)
	if gh.Stdin != nil {
		cmd.Stdin = gh.Stdin
	}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err = cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to run 'gh %s': %w: %s", strings.Join(arg, " "), err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

func (gh *GH) GetNameWithOwner(ctx context.Context) (string, error) {
	stdout, err := gh.RunWithContext(ctx, "repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner")
	if err != nil {
		return "", fmt.Errorf("failed to get repository name with owner: %w", err)
	}

	return stdout, nil
}
