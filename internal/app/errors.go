package app

import (
	"strings"
)

type ApplyFailure struct {
	Summary         string
	Guidance        string
	ConflictedFiles []string
	Detail          string
	Err             error
}

func (e *ApplyFailure) Error() string {
	if e == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(e.Summary)
	if len(e.ConflictedFiles) > 0 {
		b.WriteString("\n\nConflicted files:")
		for _, file := range e.ConflictedFiles {
			b.WriteString("\n  ")
			b.WriteString(file)
		}
	}
	if e.Guidance != "" {
		b.WriteString("\n\n")
		b.WriteString(e.Guidance)
	}
	if len(e.ConflictedFiles) == 0 && e.Detail != "" {
		b.WriteString("\n\n")
		b.WriteString(e.Detail)
	}
	return b.String()
}

func (e *ApplyFailure) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}
