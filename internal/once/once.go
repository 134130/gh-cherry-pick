package once

import (
	"context"
	"sync"
)

type OnceValue[T any] struct {
	mu    sync.Mutex
	done  bool
	value T
	err   error
}

func (o *OnceValue[T]) Do(ctx context.Context, f func(context.Context) (T, error)) (T, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if !o.done {
		o.value, o.err = f(ctx)
		o.done = true
	}

	return o.value, o.err
}
