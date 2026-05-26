package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/134130/gh-cherry-pick/internal/app"
	"github.com/134130/gh-cherry-pick/internal/app/gitkitstore"
	"github.com/134130/gh-cherry-pick/internal/output"
	"github.com/134130/gitkit/gitcmd"
	"github.com/mattn/go-isatty"
)

var (
	buildPlanFunc        = buildPlan
	executePlanFunc      = executePlan
	isTerminalReaderFunc = isTerminalReader
)

func Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	return RunWithIO(ctx, args, os.Stdin, stdout, stderr)
}

func RunWithIO(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	cmd := newCommand(stdin, stdout, stderr, runConfig)
	cmd.SetArgs(args)
	return cmd.ExecuteContext(ctx)
}

func runConfig(ctx context.Context, cfg Config, stdin io.Reader, stdout, stderr io.Writer) error {
	if !cfg.DryRun && cfg.Format == output.FormatJSON && !cfg.Yes {
		return fmt.Errorf("applying with JSON output requires --yes")
	}
	if !cfg.DryRun && !cfg.Yes && !isTerminalReaderFunc(stdin) {
		return fmt.Errorf("applying requires --yes when stdin is not a terminal")
	}

	plan, err := buildPlanFunc(ctx, cfg)
	if err != nil {
		return err
	}

	if cfg.DryRun {
		return output.RenderPlan(stdout, plan, output.PlanOptions{Format: cfg.Format, NoColor: cfg.NoColor})
	}

	if cfg.Format == output.FormatHuman {
		if err := output.RenderPlan(stdout, plan, output.PlanOptions{Format: cfg.Format, NoColor: cfg.NoColor}); err != nil {
			return err
		}
	}
	if plan.Blocked() {
		return errors.New(plan.BlockedReason)
	}
	if !cfg.Yes {
		ok, err := confirm(stdin, stderr, "Apply this PR?")
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}

	result, err := executePlanFunc(ctx, cfg, plan)
	if err != nil {
		return err
	}
	return output.RenderRunResult(stdout, result, cfg.Format)
}

func buildPlan(ctx context.Context, cfg Config) (app.Plan, error) {
	runner := gitcmd.NewRunner()
	store := gitkitstore.New(runner)
	return app.NewPlanner(store).BuildPlan(ctx, app.PlanOptions{
		PRNumber: cfg.PRNumber,
		Method:   cfg.Method,
	})
}

func executePlan(ctx context.Context, _ Config, plan app.Plan) (app.RunResult, error) {
	runner := gitcmd.NewRunner()
	executor := app.NewGitKitExecutor(runner)
	return executor.Execute(ctx, plan)
}

func isTerminalReader(reader io.Reader) bool {
	file, ok := reader.(*os.File)
	if !ok {
		return false
	}
	return isatty.IsTerminal(file.Fd()) || isatty.IsCygwinTerminal(file.Fd())
}
