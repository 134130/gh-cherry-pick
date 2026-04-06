package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	tea "charm.land/bubbletea/v2"
	"github.com/mattn/go-isatty"

	"github.com/134130/gh-cherry-pick/git"
	"github.com/134130/gh-cherry-pick/internal/log"
	"github.com/134130/gh-cherry-pick/internal/tui"
)

var (
	prNumber = flag.Int("pr", 0, "The PR number onto cherry-pick (required)")
	onto     = flag.String("onto", "", "The branch to cherry-pick onto (required)")
	merge    = flag.String("merge", "auto", "The merge strategy to use (rebase, squash, or auto) (default: auto)")
	push     = flag.Bool("push", false, "Push the cherry-picked branch to the remote branch")
	worktree = flag.Bool("worktree", false, "Use a temporary worktree cached in the OS temp directory")
)

func main() {
	flag.Parse()

	mergeStrategy := git.MergeStrategy(*merge)
	if err := mergeStrategy.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		flag.Usage()
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	hasPR := *prNumber != 0
	hasBranch := *onto != ""
	isTTY := isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())

	switch {
	case hasPR && hasBranch && isTTY:
		// Direct TUI mode: show progress with spinners
		runDirectTUI(ctx, mergeStrategy)
	case hasPR && hasBranch:
		// Plain text mode (non-TTY / piped)
		runPlainText(ctx, mergeStrategy)
	case isTTY:
		// Interactive mode: PR selection + branch input
		runInteractive(ctx, mergeStrategy)
	default:
		// Non-TTY without flags: show usage
		flag.Usage()
		os.Exit(2)
	}
}

func makeRunFunc(mergeStrategy git.MergeStrategy) tui.RunFunc {
	return func(ctx context.Context, prNum int, ontoBranch string) error {
		cp := git.CherryPick{
			PRNumber:      prNum,
			OnTo:          ontoBranch,
			MergeStrategy: mergeStrategy,
			Push:          *push,
			Worktree:      *worktree,
		}
		return cp.RunWithContext(ctx)
	}
}

func runInteractive(ctx context.Context, mergeStrategy git.MergeStrategy) {
	model := tui.NewInteractiveModel(ctx, git.ListPullRequests, makeRunFunc(mergeStrategy))
	p := tea.NewProgram(model)
	model.SetProgram(p)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runDirectTUI(ctx context.Context, mergeStrategy git.MergeStrategy) {
	model := tui.NewDirectModel(ctx, makeRunFunc(mergeStrategy), *prNumber, *onto)
	p := tea.NewProgram(model)
	model.SetProgram(p)

	header := fmt.Sprintf("  🍒 cherry-pick PR #%d → %s\n", *prNumber, *onto)
	fmt.Println(header)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runPlainText(ctx context.Context, mergeStrategy git.MergeStrategy) {
	ctx = log.CtxWithLogger(ctx)
	ctx = tui.CtxWithStepRunner(ctx, tui.PlainStepRunner)

	cp := git.CherryPick{
		PRNumber:      *prNumber,
		OnTo:          *onto,
		MergeStrategy: mergeStrategy,
		Push:          *push,
		Worktree:      *worktree,
	}
	if err := cp.RunWithContext(ctx); err != nil {
		log.LoggerFromCtx(ctx).Fail(err.Error())
		os.Exit(1)
	}
}
