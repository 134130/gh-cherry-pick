package main

import (
	"context"
	"flag"
	"os"

	"github.com/134130/gh-cherry-pick/git"
	"github.com/134130/gh-cherry-pick/internal/tui"
)

var (
	prNumber = flag.Int("pr", 0, "The PR number onto cherry-pick (required)")
	onto     = flag.String("onto", "", "The branch to cherry-pick onto (required)")
	merge    = flag.String("merge", "auto", "The merge strategy to use (rebase, squash, or auto) (default: auto)")
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
	}

	ctx := context.Background()
	if err := cherryPick.RunWithContext(ctx); err != nil {
		tui.PrintError(err.Error())
		os.Exit(1)
	}
}
