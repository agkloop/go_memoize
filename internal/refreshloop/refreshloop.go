package refreshloop

import (
	"context"
	"time"
)

// Hooks receives refresh outcomes from Run.
type Hooks[V any] struct {
	OnValue func(V)
	OnError func(error)
}

// Run calls load on each interval tick until ctx is cancelled.
// Failed loads report OnError and leave the last successful value unchanged.
func Run[V any](ctx context.Context, interval time.Duration, load func(context.Context) (V, error), hooks Hooks[V]) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			v, err := load(ctx)
			if err != nil {
				if hooks.OnError != nil {
					hooks.OnError(err)
				}
				continue
			}
			if hooks.OnValue != nil {
				hooks.OnValue(v)
			}
		}
	}
}
