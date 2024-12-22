package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"

	"github.com/134130/gh-cherry-pick/internal/log"
)

func PrintSuccess(message string) {
	_, _ = fmt.Fprintf(color.Output, "%s %s\n", green("✔"), message)
}

func PrintError(message string) {
	_, _ = fmt.Fprintf(color.Output, "%s %s\n", red("x"), message)
}

func WithSpinner(ctx context.Context, title string, f func(ctx context.Context, logger log.Logger) error) error {
	logger := log.LoggerFromCtx(ctx)
	logger.IncreaseIndent()
	defer logger.DecreaseIndent()

	sp := spinner.New(spinner.CharSets[14], 40*time.Millisecond, spinner.WithColor("cyan"))
	sp.Suffix = " " + title
	sp.Start()
	defer sp.Stop()

	return f(ctx, logger)
}
