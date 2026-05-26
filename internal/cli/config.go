package cli

import (
	"fmt"
	"strconv"

	"github.com/134130/gh-cherry-pick/internal/app"
	"github.com/134130/gh-cherry-pick/internal/output"
)

type Config struct {
	PRNumber int
	Method   app.ApplyMethod
	DryRun   bool
	Yes      bool
	Format   output.Format
	NoColor  bool
	Verbose  bool
}

func defaultConfig() Config {
	return Config{
		Method: app.MethodAuto,
		Format: output.FormatHuman,
	}
}

func parsePRNumber(arg string) (int, error) {
	number, err := strconv.Atoi(arg)
	if err != nil {
		return 0, fmt.Errorf("invalid PR number %q: %w", arg, err)
	}
	if number < 1 {
		return 0, fmt.Errorf("PR number must be greater than 0")
	}
	return number, nil
}

type methodValue app.ApplyMethod

func (v *methodValue) String() string {
	if v == nil {
		return ""
	}
	return string(*v)
}

func (v *methodValue) Type() string {
	return "method"
}

func (v *methodValue) Set(value string) error {
	method := app.ApplyMethod(value)
	if err := method.Validate(); err != nil {
		return err
	}
	*v = methodValue(method)
	return nil
}

type formatValue output.Format

func (v *formatValue) String() string {
	if v == nil {
		return ""
	}
	return string(*v)
}

func (v *formatValue) Type() string {
	return "format"
}

func (v *formatValue) Set(value string) error {
	format := output.Format(value)
	if err := format.Validate(); err != nil {
		return err
	}
	*v = formatValue(format)
	return nil
}
