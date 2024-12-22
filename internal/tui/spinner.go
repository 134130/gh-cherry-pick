package tui

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/briandowns/spinner"

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

func WithSpinner(ctx context.Context, title string, f func(ctx context.Context, logger log.Logger) error) (err error) {
	logger := log.LoggerFromCtx(ctx)
	logger.IncreaseIndent()

	sp := spinner.New(spinner.CharSets[14], 40*time.Millisecond, spinner.WithColor("cyan"))
	sp.Suffix = " " + title
	sp.FinalMSG = fmt.Sprintf("%s %s\n", internalColor.Green("✔"), title)
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
