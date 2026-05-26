package tui

import (
	"context"

	"github.com/134130/gh-cherry-pick/internal/log"
)

// StepRunner executes a named step. Implementations handle display (TUI spinner or plain text).
type StepRunner func(ctx context.Context, title string, f func(ctx context.Context, logger log.Logger) error) error

type stepRunnerKey = struct{}

func CtxWithStepRunner(ctx context.Context, sr StepRunner) context.Context {
	return context.WithValue(ctx, stepRunnerKey{}, sr)
}

func StepRunnerFromCtx(ctx context.Context) StepRunner {
	sr, ok := ctx.Value(stepRunnerKey{}).(StepRunner)
	if !ok {
		return PlainStepRunner
	}
	return sr
}
