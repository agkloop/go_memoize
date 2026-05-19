package memoize

import "time"

type Stored[V any] struct {
	Value      V
	CreatedAt  time.Time
	FreshUntil time.Time
	StaleUntil time.Time
	NoExpire   bool
	Version    string
	Tags       []string
}

type entryState uint8

const (
	entryExpired entryState = iota
	entryFresh
	entryStale
)

func (s Stored[V]) state(now time.Time) entryState {
	if s.NoExpire {
		return entryFresh
	}
	if now.Before(s.FreshUntil) || now.Equal(s.FreshUntil) {
		return entryFresh
	}
	if !s.StaleUntil.IsZero() && (now.Before(s.StaleUntil) || now.Equal(s.StaleUntil)) {
		return entryStale
	}
	return entryExpired
}
