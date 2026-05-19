module github.com/agkloop/go_memoize/adapters/redis

go 1.24.0

require (
	github.com/agkloop/go_memoize v0.0.0
	github.com/redis/go-redis/v9 v9.19.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
)

replace github.com/agkloop/go_memoize => ../..
