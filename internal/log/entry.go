package log

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/134130/gh-cherry-pick/internal/color"
)

var _ Logger = (*entry)(nil)

func newEntry(l *logger) *entry {
	return &entry{
		logger: l,
		indent: l.Indent,
	}
}

type entry struct {
	logger *logger
	indent int
	fields []field
}

type field struct {
	key   string
	value interface{}
}

func (e *entry) WithField(s string, i interface{}) Logger {
	var f []field
	copy(f, e.fields)

	slices.DeleteFunc(f, func(item field) bool {
		return item.key == s
	})

	f = append(f, field{
		key:   s,
		value: i,
	})

	return &entry{
		logger: e.logger,
		fields: f,
	}
}

func (e *entry) WithError(err error) Logger {
	if err == nil {
		return e
	}
	return e.WithField("error", err.Error())
}

func (e *entry) Info(s string) {
	e.print(infoIcon, s)
}

func (e *entry) Warn(s string) {
	e.print(warnIcon, s)
}

func (e *entry) Success(s string) {
	e.print(successIcon, s)
}

func (e *entry) Fail(s string) {
	e.print(failIcon, s)
}

func (e *entry) Infof(s string, i ...interface{}) {
	e.Info(fmt.Sprintf(s, i...))
}

func (e *entry) Warnf(s string, i ...interface{}) {
	e.Warn(fmt.Sprintf(s, i...))
}

func (e *entry) Successf(s string, i ...interface{}) {
	e.Success(fmt.Sprintf(s, i...))
}

func (e *entry) Failf(s string, i ...interface{}) {
	e.Fail(fmt.Sprintf(s, i...))
}

func (e *entry) IncreaseIndent() {
	e.logger.IncreaseIndent()
}

func (e *entry) DecreaseIndent() {
	e.logger.DecreaseIndent()
}

func (e *entry) ResetIndent() {
	e.logger.ResetIndent()
}

func (e *entry) print(icon, msg string) {
	bullet := lipgloss.NewStyle().PaddingLeft(1 + e.logger.Indent).Render(icon)
	content := lipgloss.NewStyle().PaddingLeft(1).Render(msg)

	output := lipgloss.JoinHorizontal(lipgloss.Top, bullet, content)
	if len(e.fields) == 0 {
		_, _ = fmt.Fprintln(os.Stdout, output)
		return
	}

	fields := make([]string, 0, len(e.fields))
	for _, f := range e.fields {
		fields = append(fields, fmt.Sprintf("%s=%v", color.Purple(f.key), f.value))
	}

	_, _ = fmt.Fprintln(os.Stderr, lipgloss.JoinHorizontal(
		lipgloss.Top,
		output,
		lipgloss.NewStyle().PaddingLeft(max(maxIndent-lipgloss.Width(output), 0)).Render(strings.Join(fields, " "))),
	)
}

var maxIndent = 70
