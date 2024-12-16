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
	rebase   = flag.Bool("rebase", false, "Rebase the cherry-pick")
)

func main() {
	flag.Parse()
	if *prNumber == 0 || *onto == "" {
		flag.Usage()
		os.Exit(2)
	}

	cherryPick := git.CherryPick{
		PRNumber: *prNumber,
		OnTo:     *onto,
		Rebase:   *rebase,
	}

	ctx := context.Background()
	if err := cherryPick.RunWithContext(ctx); err != nil {
		tui.PrintError(err.Error())
		os.Exit(1)
	}
}
