package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"

	"github.com/134130/gh-cherry-pick/internal/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := cli.Run(ctx, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		if errors.Is(err, context.Canceled) {
			os.Exit(130)
		}
		if cli.IsReportedError(err) {
			os.Exit(1)
		}
		_, _ = fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
