package httpusercache

import (
	"context"
	"errors"
	"net/http"
	"time"

	memoize "github.com/agkloop/go_memoize"
	"github.com/agkloop/go_memoize/stores/memory"
)

const (
	defaultCacheCapacity  = 50_000
	defaultFreshTTL       = 30 * time.Second
	defaultStaleTTL       = 2 * time.Minute
	defaultRequestTimeout = 2 * time.Second
	defaultRefreshTimeout = 3 * time.Second
)

var ErrMissingRepository = errors.New("httpusercache: missing repository")

type Logger interface {
	Printf(format string, args ...any)
}

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type User struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Company string `json:"company"`
}

type UserRepository interface {
	LoadUser(context.Context, int64) (User, error)
}

type UserServiceConfig struct {
	Repository     UserRepository
	BaseURL        string
	HTTPClient     HTTPClient
	CacheCapacity  int
	FreshTTL       time.Duration
	StaleTTL       time.Duration
	RequestTimeout time.Duration
	RefreshTimeout time.Duration
	Metrics        memoize.Metrics
	Logger         Logger
}

type UserService struct {
	repository UserRepository
	cache      *memoize.Cache[int64, User]
}

func NewUserService(cfg UserServiceConfig) (*UserService, error) {
	repository := cfg.Repository
	if repository == nil {
		created, err := NewBeeceptorUserRepository(BeeceptorUserRepositoryConfig{
			BaseURL:        cfg.BaseURL,
			HTTPClient:     cfg.HTTPClient,
			RequestTimeout: cfg.RequestTimeout,
			Logger:         cfg.Logger,
		})
		if err != nil {
			return nil, err
		}
		repository = created
	}
	if repository == nil {
		return nil, ErrMissingRepository
	}

	opts := memoize.Opts().
		WithStore(memory.New[int64, User](defaultInt(cfg.CacheCapacity, defaultCacheCapacity))).
		WithTTL(defaultDuration(cfg.FreshTTL, defaultFreshTTL)).
		WithStaleTTL(defaultDuration(cfg.StaleTTL, defaultStaleTTL)).
		WithRefreshTimeout(defaultDuration(cfg.RefreshTimeout, defaultRefreshTimeout)).
		KeepStaleOnError()
	if cfg.Metrics != nil {
		opts = opts.WithMetrics(cfg.Metrics)
	}

	cache, err := memoize.New[int64, User](opts)
	if err != nil {
		return nil, err
	}
	return &UserService{repository: repository, cache: cache}, nil
}

func (s *UserService) GetUser(ctx context.Context, userID int64) (User, error) {
	return s.cache.GetOrCompute(ctx, userID, func(ctx context.Context) (User, error) {
		return s.repository.LoadUser(ctx, userID)
	})
}

func (s *UserService) Close() {
	s.cache.Stop()
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
