package log

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"

	"github.com/134130/gh-cherry-pick/internal/color"
)

var (
	infoIcon    = color.Blue("•")
	warnIcon    = color.Yellow("!️")
	successIcon = color.Green("✔")
	failIcon    = color.Red("✘")
)

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

func (l *logger) WithField(s string, i interface{}) Logger {
	return newEntry(l).WithField(s, i)
}

func (l *logger) WithError(err error) Logger {
	return newEntry(l).WithError(err)
}

func (l *logger) Info(s string) {
	l.print(infoIcon, s)
}

func (l *logger) Warn(s string) {
	l.print(warnIcon, s)
}

func (l *logger) Success(s string) {
	l.print(successIcon, s)
}

func (l *logger) Fail(s string) {
	l.print(failIcon, s)
}

func (l *logger) Infof(s string, i ...interface{}) {
	l.print(infoIcon, fmt.Sprintf(s, i...))
}

func (l *logger) Warnf(s string, i ...interface{}) {
	l.print(warnIcon, fmt.Sprintf(s, i...))
}

func (l *logger) Successf(s string, i ...interface{}) {
	l.print(successIcon, fmt.Sprintf(s, i...))
}

func (l *logger) Failf(s string, i ...interface{}) {
	l.print(failIcon, fmt.Sprintf(s, i...))
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

func (l *logger) print(icon, msg string) {
	bullet := lipgloss.NewStyle().PaddingLeft(1 + l.Indent).Render(icon)
	content := lipgloss.NewStyle().PaddingLeft(1).Render(msg)

	_, _ = fmt.Fprintln(os.Stdout, lipgloss.JoinHorizontal(lipgloss.Top, bullet, content))
}
