package memoize

import (
	"sync"
	"sync/atomic"
	"time"
)

type Clock interface {
	Now() time.Time
}

type ClockFunc func() time.Time

func (f ClockFunc) Now() time.Time {
	return f()
}

type TickerClock struct {
	now  atomic.Int64
	stop chan struct{}
	once sync.Once
}

func NewTickerClock(interval time.Duration) *TickerClock {
	tc := &TickerClock{stop: make(chan struct{})}
	tc.now.Store(time.Now().UnixMilli())
	go tc.run(interval)
	return tc
}

func (tc *TickerClock) run(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			tc.now.Store(time.Now().UnixMilli())
		case <-tc.stop:
			return
		}
	}
}

func (tc *TickerClock) Stop() {
	tc.once.Do(func() { close(tc.stop) })
}

func (tc *TickerClock) Now() time.Time {
	return time.UnixMilli(tc.now.Load())
}
