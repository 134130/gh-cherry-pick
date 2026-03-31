package git

import (
	"bytes"
	"context"
	"strings"
)

// ghAPIQuery runs: gh api --hostname <hostname> [headers] <endpoint> --jq <jqExpr>
// and returns trimmed stdout. Pass nil for headers when none are needed.
func ghAPIQuery(ctx context.Context, hostname, endpoint, jqExpr string, headers map[string]string) (string, error) {
	args := []string{"api", "--hostname", hostname}
	for k, v := range headers {
		args = append(args, "-H", k+": "+v)
	}
	args = append(args, endpoint, "--jq", jqExpr)

	stdout := &bytes.Buffer{}
	if err := NewCommand("gh", args...).Run(ctx, WithStdout(stdout)); err != nil {
		return "", err
	}
	return strings.TrimSpace(stdout.String()), nil
}
