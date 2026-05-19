package redisstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	memoize "github.com/agkloop/go_memoize"
	"github.com/redis/go-redis/v9"
)

var (
	ErrMissingClient     = errors.New("redisstore: missing redis client")
	ErrMissingSerializer = errors.New("redisstore: missing serializer")
)

type Store[K comparable, V any] struct {
	client     redis.UniversalClient
	prefix     string
	serializer memoize.Serializer[V]
	keyEncoder func(K) string
}

type Option[K comparable, V any] func(*Store[K, V])

func WithClient[K comparable, V any](client redis.UniversalClient) Option[K, V] {
	return func(s *Store[K, V]) { s.client = client }
}

func WithPrefix[K comparable, V any](prefix string) Option[K, V] {
	return func(s *Store[K, V]) { s.prefix = prefix }
}

func WithSerializer[K comparable, V any](serializer memoize.Serializer[V]) Option[K, V] {
	return func(s *Store[K, V]) { s.serializer = serializer }
}

func WithKeyEncoder[K comparable, V any](encode func(K) string) Option[K, V] {
	return func(s *Store[K, V]) { s.keyEncoder = encode }
}

func New[K comparable, V any](opts ...Option[K, V]) (*Store[K, V], error) {
	s := &Store[K, V]{keyEncoder: defaultKeyEncoder[K]}
	for _, opt := range opts {
		opt(s)
	}
	if s.client == nil {
		return nil, ErrMissingClient
	}
	if s.serializer == nil {
		return nil, ErrMissingSerializer
	}
	return s, nil
}

func defaultKeyEncoder[K comparable](key K) string {
	switch k := any(key).(type) {
	case string:
		return k
	case uint64:
		return strconv.FormatUint(k, 10)
	case uint:
		return strconv.FormatUint(uint64(k), 10)
	case uint32:
		return strconv.FormatUint(uint64(k), 10)
	case uint16:
		return strconv.FormatUint(uint64(k), 10)
	case uint8:
		return strconv.FormatUint(uint64(k), 10)
	case int:
		return strconv.FormatInt(int64(k), 10)
	case int64:
		return strconv.FormatInt(k, 10)
	case int32:
		return strconv.FormatInt(int64(k), 10)
	case int16:
		return strconv.FormatInt(int64(k), 10)
	case int8:
		return strconv.FormatInt(int64(k), 10)
	default:
		return fmt.Sprint(key)
	}
}

type envelope struct {
	Value      []byte    `json:"value"`
	CreatedAt  time.Time `json:"created_at"`
	FreshUntil time.Time `json:"fresh_until"`
	StaleUntil time.Time `json:"stale_until"`
	NoExpire   bool      `json:"no_expire"`
	Version    string    `json:"version"`
	Tags       []string  `json:"tags"`
}

func (s *Store[K, V]) key(key K) string {
	encode := s.keyEncoder
	if encode == nil {
		encode = defaultKeyEncoder[K]
	}
	return s.prefixed(encode(key))
}

func (s *Store[K, V]) prefixed(encoded string) string {
	if s.prefix == "" {
		return encoded
	}
	return strings.TrimRight(s.prefix, ":") + ":" + encoded
}

func (s *Store[K, V]) Get(ctx context.Context, key K) (memoize.Stored[V], bool, error) {
	var zero memoize.Stored[V]
	data, err := s.client.Get(ctx, s.key(key)).Bytes()
	if errors.Is(err, redis.Nil) {
		return zero, false, nil
	}
	if err != nil {
		return zero, false, err
	}
	var env envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return zero, false, err
	}
	value, err := s.serializer.Unmarshal(env.Value)
	if err != nil {
		return zero, false, err
	}
	return memoize.Stored[V]{Value: value, CreatedAt: env.CreatedAt, FreshUntil: env.FreshUntil, StaleUntil: env.StaleUntil, NoExpire: env.NoExpire, Version: env.Version, Tags: env.Tags}, true, nil
}

func (s *Store[K, V]) Set(ctx context.Context, key K, value memoize.Stored[V]) error {
	encoded, err := s.serializer.Marshal(value.Value)
	if err != nil {
		return err
	}
	data, err := json.Marshal(envelope{Value: encoded, CreatedAt: value.CreatedAt, FreshUntil: value.FreshUntil, StaleUntil: value.StaleUntil, NoExpire: value.NoExpire, Version: value.Version, Tags: value.Tags})
	if err != nil {
		return err
	}
	return s.client.Set(ctx, s.key(key), data, storageTTL(value, time.Now())).Err()
}

func (s *Store[K, V]) Delete(ctx context.Context, key K) error {
	return s.client.Del(ctx, s.key(key)).Err()
}

func (s *Store[K, V]) Clear(ctx context.Context) error {
	pattern := s.prefixed("*")
	iter := s.client.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		if err := s.client.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}
	return iter.Err()
}

func storageTTL[V any](entry memoize.Stored[V], now time.Time) time.Duration {
	if entry.NoExpire {
		return 0
	}
	deadline := entry.FreshUntil
	if entry.StaleUntil.After(deadline) {
		deadline = entry.StaleUntil
	}
	if deadline.IsZero() {
		return 0
	}
	ttl := deadline.Sub(now)
	if ttl <= 0 {
		return time.Millisecond
	}
	return ttl
}
