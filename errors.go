package memoize

import "errors"

var (
	ErrMissingExpirationPolicy = errors.New("memoize: missing expiration policy")
	ErrInvalidTTL              = errors.New("memoize: ttl must be greater than zero")
	ErrInvalidStaleTTL         = errors.New("memoize: stale ttl requires a positive ttl")
	ErrMissingStore            = errors.New("memoize: missing store")
	ErrInvalidStore            = errors.New("memoize: store does not match cache key/value types")
)
