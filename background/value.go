// v2/background/value.go
package background

import "sync/atomic"

// Value holds a locally-mirrored copy of V refreshed in the background.
// Get is a single atomic pointer load — sub-nanosecond, never errors.
// Callers must not mutate the returned value; it is shared memory.
type Value[V any] struct {
	v atomic.Pointer[V]
}

// Get returns the current value. Always local — never blocks, never errors.
func (val *Value[V]) Get() V {
	return *val.v.Load()
}

func (val *Value[V]) store(v V) {
	val.v.Store(&v)
}
