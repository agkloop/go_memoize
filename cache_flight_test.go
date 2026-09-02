package memoize

import (
	"testing"
)

func TestFlightWaitRepanicsWithOriginalValue(t *testing.T) {
	sentinel := &struct{ message string }{message: "boom"}
	f := &flight[int]{
		panicValue: sentinel,
		panicked:   true,
	}
	f.wg.Add(1)
	f.wg.Done()

	defer func() {
		if got := recover(); got != sentinel {
			t.Fatalf("recovered panic = %v, want identical sentinel", got)
		}
	}()
	_, _ = f.wait()
}
