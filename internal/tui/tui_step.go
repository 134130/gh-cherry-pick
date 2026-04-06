package tui

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/134130/gh-cherry-pick/internal/log"
)

// newTUIStepRunner creates a StepRunner that communicates with the bubbletea program via p.Send().
func newTUIStepRunner(p *tea.Program) StepRunner {
	return func(ctx context.Context, title string, f func(ctx context.Context, logger log.Logger) error) error {
		p.Send(stepStartMsg{title: title})

		start := time.Now()
		logger := log.LoggerFromCtx(ctx)
		err := f(ctx, logger)
		elapsed := time.Since(start)

		if err != nil {
			p.Send(stepErrorMsg{err: err, duration: elapsed})
		} else {
			p.Send(stepDoneMsg{duration: elapsed})
		}
		return err
	}
}
