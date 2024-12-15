package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/134130/gh-cherry-pick/git"
)

var prNumber = flag.Int("pr", 0, "The PR number to detect (required)")

func main() {
	flag.Parse()
	if *prNumber == 0 {
		flag.Usage()
		os.Exit(2)
	}

	ctx := context.Background()
	mergeStrategy, err := git.PRMergedWith(ctx, *prNumber)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s\n", mergeStrategy)
}
