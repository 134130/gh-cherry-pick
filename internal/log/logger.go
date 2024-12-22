package log

import (
	"context"
	"fmt"

	"github.com/fatih/color"

	"github.com/134130/gh-cherry-pick/internal/tui"
)

type Logger interface {
	Infof(string, ...interface{})
	Warnf(string, ...interface{})
	Successf(string, ...interface{})
	Failf(string, ...interface{})

	IncreaseIndent()
	DecreaseIndent()
	ResetIndent()
}

func NewLogger() Logger {
	return &logger{}
}

var loggerKey = struct{}{}

func CtxWithLogger(ctx context.Context) context.Context {
	return context.WithValue(ctx, loggerKey, NewLogger())
}

func LoggerFromCtx(ctx context.Context) Logger {
	l, ok := ctx.Value(loggerKey).(Logger)
	if !ok {
		return NewLogger()
	}
	return l
}

var _ Logger = (*logger)(nil)

type logger struct {
	Indent int
}

func (l *logger) Infof(s string, i ...interface{}) {
	_, _ = fmt.Fprintf(color.Output, "%*s %s %s\n", l.Indent, "", tui.Blue("•"), fmt.Sprintf(s, i...))
}

func (l *logger) Warnf(s string, i ...interface{}) {
	_, _ = fmt.Fprintf(color.Output, "%*s %s %s\n", l.Indent, "", tui.Yellow("!️"), fmt.Sprintf(s, i...))
}

func (l *logger) Successf(s string, i ...interface{}) {
	_, _ = fmt.Fprintf(color.Output, "%*s %s %s\n", l.Indent, "", tui.Green("✔"), fmt.Sprintf(s, i...))
}

func (l *logger) Failf(s string, i ...interface{}) {
	_, _ = fmt.Fprintf(color.Output, "%*s %s %s\n", l.Indent, "", tui.Red("✘"), fmt.Sprintf(s, i...))
}

func (l *logger) IncreaseIndent() {
	l.Indent += 2
}

func (l *logger) DecreaseIndent() {
	l.Indent -= 2
}

func (l *logger) ResetIndent() {
	l.Indent = 0
}
