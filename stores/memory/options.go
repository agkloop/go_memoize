package memory

type options struct {
	getRecencySample uint32
	shards           int
	maxBytes         int64 // 0 = unbounded
}

type Option[K comparable, V any] func(*options)

// WithMaxBytes limits the estimated in-memory footprint of the store to n bytes,
// evicting the least-recently-used entry whenever a new insert would exceed the limit.
//
// The byte cost of each entry is estimated as:
//
//	unsafe.Sizeof(key) + unsafe.Sizeof(value)
//
// This is a shallow, compile-time struct estimate. It counts header sizes of
// pointer-bearing types (strings, slices) but NOT the heap data they point to.
// For V=string, the string content bytes are not counted. Plan capacity accordingly.
//
// If a single entry's cost exceeds the limit and the store is empty, it is admitted
// anyway (a store must hold at least one item). In this case usedBytes will exceed maxBytes.
//
// n <= 0 is ignored (unbounded).
func WithMaxBytes[K comparable, V any](n int64) Option[K, V] {
	return func(o *options) {
		if n > 0 {
			o.maxBytes = n
		}
	}
}

// WithGetRecencySample makes Get refresh LRU recency once every n hits.
// n <= 1 preserves exact LRU behavior by refreshing recency on every hit.
func WithGetRecencySample[K comparable, V any](n uint32) Option[K, V] {
	return func(o *options) {
		o.getRecencySample = n
	}
}

// WithShards sets the number of shards used by NewSharded.
// n must be a positive power of two.
func WithShards[K comparable, V any](n int) Option[K, V] {
	return func(o *options) {
		o.shards = n
	}
}
