package main

import (
	"context"
	"flag"
	"os"
	"os/signal"

	"github.com/134130/gh-cherry-pick/git"
	"github.com/134130/gh-cherry-pick/internal/log"
)

var (
	prNumber = flag.Int("pr", 0, "The PR number onto cherry-pick (required)")
	onto     = flag.String("onto", "", "The branch to cherry-pick onto (required)")
	merge    = flag.String("merge", "auto", "The merge strategy to use (rebase, squash, or auto) (default: auto)")
	push     = flag.Bool("push", false, "Push the cherry-picked branch to the remote branch")
)

func main() {
	flag.Parse()
	if *prNumber == 0 || *onto == "" {
		flag.Usage()
		os.Exit(2)
	}

	cherryPick := git.CherryPick{
		PRNumber:      *prNumber,
		OnTo:          *onto,
		MergeStrategy: git.MergeStrategy(*merge),
		Push:          *push,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	ctx = log.CtxWithLogger(ctx)

	if err := cherryPick.RunWithContext(ctx); err != nil {
		log.LoggerFromCtx(ctx).Failf(err.Error())
		os.Exit(1)
	}
}
