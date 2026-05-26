package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/134130/gh-cherry-pick/internal/output"
	"github.com/spf13/cobra"
)

type commandHandler func(context.Context, Config, io.Reader, io.Writer, io.Writer) error

func Parse(args []string) (Config, error) {
	var parsed Config
	cmd := newCommand(strings.NewReader(""), io.Discard, io.Discard, func(_ context.Context, cfg Config, _ io.Reader, _, _ io.Writer) error {
		parsed = cfg
		return nil
	})
	cmd.SetArgs(args)
	return parsed, cmd.ExecuteContext(context.Background())
}

func newCommand(stdin io.Reader, stdout, stderr io.Writer, handler commandHandler) *cobra.Command {
	cfg := defaultConfig()
	jsonFlag := false

	root := &cobra.Command{
		Use:           "gh cherry-pick <pr>",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("accepts 1 arg, received %d", len(args))
			}
			number, err := parsePRNumber(args[0])
			if err != nil {
				return err
			}
			cfg.PRNumber = number
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			if jsonFlag {
				cfg.Format = output.FormatJSON
			}
			if err := cfg.Method.Validate(); err != nil {
				return err
			}
			if err := cfg.Format.Validate(); err != nil {
				return err
			}
			return handler(cmd.Context(), cfg, stdin, stdout, stderr)
		},
	}
	root.SetOut(stdout)
	root.SetErr(stderr)

	flags := root.Flags()
	flags.Var((*methodValue)(&cfg.Method), "method", "Apply method: auto, patch, commits, squash, merge")
	flags.BoolVar(&cfg.DryRun, "dry-run", false, "Print the plan without applying it")
	flags.BoolVarP(&cfg.Yes, "yes", "y", false, "Run without interactive confirmation")
	flags.Var((*formatValue)(&cfg.Format), "format", "Output format: human, json")
	flags.BoolVar(&jsonFlag, "json", false, "Output JSON")
	flags.BoolVar(&cfg.NoColor, "no-color", false, "Disable ANSI color")
	flags.BoolVarP(&cfg.Verbose, "verbose", "v", false, "Print command/progress details")

	return root
}
