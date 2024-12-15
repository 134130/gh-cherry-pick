package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
)

var (
	green = color.New(color.FgGreen).SprintFunc()
	red   = color.New(color.FgRed).SprintFunc()
)

func PrintSuccess(message string) {
	_, _ = fmt.Fprintf(color.Output, "%s %s\n", green("✔"), message)
}

func PrintError(message string) {
	_, _ = fmt.Fprintf(color.Output, "%s %s\n", red("x"), message)
}

func WithSpinner(ctx context.Context, message string, f func(ctx context.Context) (string, error)) {
	sp := spinner.New(spinner.CharSets[14], 40*time.Millisecond)
	sp.Suffix = " " + message
	sp.Start()

	if str, err := f(ctx); err != nil {
		sp.Stop()
		PrintError(err.Error())
	} else {
		sp.Stop()
		PrintSuccess(str)
	}
}
