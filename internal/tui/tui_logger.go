package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/134130/gh-cherry-pick/internal/log"
)

var (
	tuiInfoIcon    = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(4)).Render("•")
	tuiWarnIcon    = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(3)).Render("!")
	tuiSuccessIcon = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(2)).Render("✔")
	tuiFailIcon    = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(1)).Render("✘")
)

// --- tuiLogger ---

var _ log.Logger = (*tuiLogger)(nil)

type tuiLogger struct {
	p *tea.Program
}

func newTUILogger(p *tea.Program) *tuiLogger {
	return &tuiLogger{p: p}
}

func (l *tuiLogger) send(icon, text string) {
	l.p.Send(logLineMsg{icon: icon, text: text})
}

func (l *tuiLogger) Info(s string)    { l.send(tuiInfoIcon, s) }
func (l *tuiLogger) Warn(s string)    { l.send(tuiWarnIcon, s) }
func (l *tuiLogger) Success(s string) { l.send(tuiSuccessIcon, s) }
func (l *tuiLogger) Fail(s string)    { l.send(tuiFailIcon, s) }

func (l *tuiLogger) Infof(s string, args ...interface{})    { l.Info(fmt.Sprintf(s, args...)) }
func (l *tuiLogger) Warnf(s string, args ...interface{})    { l.Warn(fmt.Sprintf(s, args...)) }
func (l *tuiLogger) Successf(s string, args ...interface{}) { l.Success(fmt.Sprintf(s, args...)) }
func (l *tuiLogger) Failf(s string, args ...interface{})    { l.Fail(fmt.Sprintf(s, args...)) }

func (l *tuiLogger) WithField(key string, val interface{}) log.Logger {
	return &tuiEntry{logger: l, fields: []tuiField{{key: key, value: val}}}
}

func (l *tuiLogger) WithError(err error) log.Logger {
	if err == nil {
		return l
	}
	return l.WithField("error", err.Error())
}

// No-ops: TUI handles indentation via step structure
func (l *tuiLogger) IncreaseIndent() {}
func (l *tuiLogger) DecreaseIndent() {}
func (l *tuiLogger) ResetIndent()    {}

// --- tuiEntry (for WithField chaining) ---

var _ log.Logger = (*tuiEntry)(nil)

type tuiField struct {
	key   string
	value interface{}
}

type tuiEntry struct {
	logger *tuiLogger
	fields []tuiField
}

func (e *tuiEntry) formatFields() string {
	if len(e.fields) == 0 {
		return ""
	}
	parts := make([]string, len(e.fields))
	fieldStyle := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(5))
	for i, f := range e.fields {
		parts[i] = fmt.Sprintf("%s=%v", fieldStyle.Render(f.key), f.value)
	}
	return "  " + strings.Join(parts, " ")
}

func (e *tuiEntry) send(icon, text string) {
	e.logger.p.Send(logLineMsg{icon: icon, text: text + e.formatFields()})
}

func (e *tuiEntry) Info(s string)    { e.send(tuiInfoIcon, s) }
func (e *tuiEntry) Warn(s string)    { e.send(tuiWarnIcon, s) }
func (e *tuiEntry) Success(s string) { e.send(tuiSuccessIcon, s) }
func (e *tuiEntry) Fail(s string)    { e.send(tuiFailIcon, s) }

func (e *tuiEntry) Infof(s string, args ...interface{})    { e.Info(fmt.Sprintf(s, args...)) }
func (e *tuiEntry) Warnf(s string, args ...interface{})    { e.Warn(fmt.Sprintf(s, args...)) }
func (e *tuiEntry) Successf(s string, args ...interface{}) { e.Success(fmt.Sprintf(s, args...)) }
func (e *tuiEntry) Failf(s string, args ...interface{})    { e.Fail(fmt.Sprintf(s, args...)) }

func (e *tuiEntry) WithField(key string, val interface{}) log.Logger {
	fields := make([]tuiField, len(e.fields))
	copy(fields, e.fields)
	fields = append(fields, tuiField{key: key, value: val})
	return &tuiEntry{logger: e.logger, fields: fields}
}

func (e *tuiEntry) WithError(err error) log.Logger {
	if err == nil {
		return e
	}
	return e.WithField("error", err.Error())
}

func (e *tuiEntry) IncreaseIndent() {}
func (e *tuiEntry) DecreaseIndent() {}
func (e *tuiEntry) ResetIndent()    {}
