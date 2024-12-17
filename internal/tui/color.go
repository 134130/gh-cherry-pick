package tui

import (
	"github.com/fatih/color"
)

var (
	cyan   = color.New(color.FgCyan).SprintFunc()
	green  = color.New(color.FgGreen).SprintFunc()
	red    = color.New(color.FgRed).SprintFunc()
	yellow = color.New(color.FgYellow).SprintFunc()
	purple = color.New(color.FgMagenta).SprintFunc()
	grey   = color.New(color.FgHiWhite).SprintFunc()
)

func Cyan(a ...interface{}) string {
	return cyan(a...)
}

func Green(a ...interface{}) string {
	return green(a...)
}

func Red(a ...interface{}) string {
	return red(a...)
}

func Yellow(a ...interface{}) string {
	return yellow(a...)
}

func Purple(a ...interface{}) string {
	return purple(a...)
}

func Grey(a ...interface{}) string {
	return grey(a...)
}
