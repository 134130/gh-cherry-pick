package tui

import (
	"context"
	"fmt"
	"os"

	internalColor "github.com/134130/gh-cherry-pick/internal/color"
	"github.com/134130/gh-cherry-pick/internal/log"
)

func WithStep(ctx context.Context, title string, f func(ctx context.Context, logger log.Logger) error) (err error) {
	logger := log.LoggerFromCtx(ctx)
	logger.Infof(internalColor.Bold(title))

	logger.IncreaseIndent()
	defer logger.DecreaseIndent()

	err = f(ctx, logger)

	_, _ = fmt.Fprintln(os.Stdout)

	return
}

