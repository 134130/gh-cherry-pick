package tui

import (
	"context"
	"time"

	"github.com/briandowns/spinner"

	"github.com/134130/gh-cherry-pick/internal/log"
)

func WithSpinner(ctx context.Context, title string, f func(ctx context.Context, logger log.Logger) error) (err error) {
	logger := log.LoggerFromCtx(ctx)
	logger.IncreaseIndent()

	sp := spinner.New(spinner.CharSets[14], 40*time.Millisecond, spinner.WithColor("cyan"))
	sp.Suffix = " " + title
	sp.Start()
	defer func() {
		sp.Stop()
		logger.DecreaseIndent()

		if err != nil {
			logger.Failf(err.Error())
		} else {
			logger.Successf(title)
		}
	}()

	err = f(ctx, logger)

	return
}
