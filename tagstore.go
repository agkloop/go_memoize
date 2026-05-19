package memoize

import "context"

// TaggedStore is an optional extension of Store[K, V] for tag-based invalidation.
// A Store that supports tags should implement this interface.
type TaggedStore[K comparable, V any] interface {
	Store[K, V]
	// DeleteByTag removes all entries whose Tags slice contains the given tag.
	DeleteByTag(ctx context.Context, tag string) error
}
