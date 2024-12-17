package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
)

func PrintSuccess(message string) {
	_, _ = fmt.Fprintf(color.Output, "%s %s\n", green("✔"), message)
}

func PrintError(message string) {
	_, _ = fmt.Fprintf(color.Output, "%s %s\n", red("x"), message)
}

func WithSpinner(ctx context.Context, message string, f func(ctx context.Context) (string, error)) (err error) {
	sp := spinner.New(spinner.CharSets[14], 40*time.Millisecond, spinner.WithColor("cyan"))
	sp.Suffix = " " + message
	sp.Start()

	var str string
	defer func() {
		sp.Stop()
		if err == nil {
			PrintSuccess(str)
		}
	}()

	str, err = f(ctx)
	return
}
