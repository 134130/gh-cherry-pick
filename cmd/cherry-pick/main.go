package main

import (
	"context"
	"flag"
	"os"

	"github.com/134130/gh-cherry-pick/git"
)

var (
	prNumber = flag.Int("pr", 0, "The PR number to cherry-pick (required)")
	to       = flag.String("to", "", "The branch to cherry-pick to (required)")
	rebase   = flag.Bool("rebase", false, "Rebase the cherry-pick")
)

func main() {
	flag.Parse()
	if *prNumber == 0 || *to == "" {
		flag.Usage()
		os.Exit(2)
	}

	cherryPick := git.CherryPick{
		PRNumber: *prNumber,
		To:       *to,
		Rebase:   *rebase,
	}

	ctx := context.Background()
	if err := cherryPick.RunWithContext(ctx); err != nil {
		panic(err)
	}
}
