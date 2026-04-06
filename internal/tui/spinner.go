package tui

import (
	"context"
	"fmt"
	"os"

	internalColor "github.com/134130/gh-cherry-pick/internal/color"
	"github.com/134130/gh-cherry-pick/internal/log"
)

// PlainStepRunner is the plain-text fallback step runner.
// It prints the step title in bold, increases indent for nested logs,
// runs the step function, then decreases indent and prints a blank line.
func PlainStepRunner(ctx context.Context, title string, f func(ctx context.Context, logger log.Logger) error) (err error) {
	logger := log.LoggerFromCtx(ctx)
	logger.Infof(internalColor.Bold(title))

	logger.IncreaseIndent()
	defer logger.DecreaseIndent()

	err = f(ctx, logger)

	_, _ = fmt.Fprintln(os.Stdout)

	return
}
