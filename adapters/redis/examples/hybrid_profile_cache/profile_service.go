package hybridprofilecache

import (
	"context"
	"errors"
	"net/http"
	"time"

	memoize "github.com/agkloop/go_memoize"
	redisstore "github.com/agkloop/go_memoize/adapters/redis"
	"github.com/agkloop/go_memoize/serializers"
	"github.com/agkloop/go_memoize/stores/chain"
	"github.com/agkloop/go_memoize/stores/memory"
	"github.com/redis/go-redis/v9"
)

const (
	defaultHybridCapacity = 25_000
	defaultHybridFreshTTL = 30 * time.Second
	defaultHybridStaleTTL = 5 * time.Minute
	defaultRequestTimeout = 2 * time.Second
)

var (
	ErrMissingRedisClient = errors.New("hybridprofilecache: missing redis client")
	ErrMissingRepository  = errors.New("hybridprofilecache: missing repository")
	ErrMissingStore       = errors.New("hybridprofilecache: missing store")
)

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type Profile struct {
	ID          int64  `json:"id"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Company     string `json:"company"`
}

type ProfileRepository interface {
	LoadProfile(context.Context, int64) (Profile, error)
}

type ProfileServiceConfig struct {
	Repository    ProfileRepository
	RedisClient   redis.UniversalClient
	RedisPrefix   string
	CacheCapacity int
	FreshTTL      time.Duration
	StaleTTL      time.Duration
	Metrics       memoize.Metrics
}

type ProfileServiceStoreConfig struct {
	Repository ProfileRepository
	Store      memoize.Store[uint64, Profile]
	FreshTTL   time.Duration
	StaleTTL   time.Duration
	Metrics    memoize.Metrics
}

type ProfileService struct {
	loadProfile func(context.Context, int64) (Profile, error)
}

func NewProfileService(cfg ProfileServiceConfig) (*ProfileService, error) {
	if cfg.RedisClient == nil {
		return nil, ErrMissingRedisClient
	}
	l1 := memory.New[uint64, Profile](defaultInt(cfg.CacheCapacity, defaultHybridCapacity))
	l2, err := redisstore.New[uint64, Profile](
		redisstore.WithClient[uint64, Profile](cfg.RedisClient),
		redisstore.WithPrefix[uint64, Profile](defaultString(cfg.RedisPrefix, "profiles")),
		redisstore.WithSerializer[uint64, Profile](serializers.JSON[Profile]{}),
	)
	if err != nil {
		return nil, err
	}
	return NewProfileServiceWithStore(ProfileServiceStoreConfig{
		Repository: cfg.Repository,
		Store:      chain.New[uint64, Profile](l1, l2),
		FreshTTL:   cfg.FreshTTL,
		StaleTTL:   cfg.StaleTTL,
		Metrics:    cfg.Metrics,
	})
}

func NewProfileServiceWithStore(cfg ProfileServiceStoreConfig) (*ProfileService, error) {
	if cfg.Repository == nil {
		return nil, ErrMissingRepository
	}
	if cfg.Store == nil {
		return nil, ErrMissingStore
	}
	opts := memoize.Opts().
		WithStore(cfg.Store).
		WithTTL(defaultDuration(cfg.FreshTTL, defaultHybridFreshTTL)).
		WithStaleTTL(defaultDuration(cfg.StaleTTL, defaultHybridStaleTTL)).
		KeepStaleOnError()
	if cfg.Metrics != nil {
		opts = opts.WithMetrics(cfg.Metrics)
	}
	cached, err := memoize.MemoizeCtx1E(cfg.Repository.LoadProfile, opts)
	if err != nil {
		return nil, err
	}
	return &ProfileService{loadProfile: cached}, nil
}

func (s *ProfileService) GetProfile(ctx context.Context, profileID int64) (Profile, error) {
	return s.loadProfile(ctx, profileID)
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func defaultInt(value, fallback int) int {
	if value == 0 {
		return fallback
	}
	return value
}

func defaultDuration(value, fallback time.Duration) time.Duration {
	if value == 0 {
		return fallback
	}
	return value
}
