package directstaleprofilecache

import (
	"context"
	"errors"
	"net/http"
	"time"

	memoize "github.com/agkloop/go_memoize"
)

const (
	defaultFreshTTL       = time.Minute
	defaultStaleTTL       = 5 * time.Minute
	defaultRequestTimeout = 2 * time.Second
)

var ErrMissingRepository = errors.New("directstaleprofilecache: missing repository")

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type Profile struct {
	ID          int64
	DisplayName string
	Email       string
	Company     string
}

type ProfileRepository interface {
	LoadProfile(context.Context, int64) (Profile, error)
}

type ProfileServiceConfig struct {
	Repository ProfileRepository
	FreshTTL   time.Duration
	StaleTTL   time.Duration
	Metrics    memoize.Metrics
}

type ProfileService struct {
	loadProfile func(context.Context, int64) (Profile, error)
}

func NewProfileService(cfg ProfileServiceConfig) (*ProfileService, error) {
	if cfg.Repository == nil {
		return nil, ErrMissingRepository
	}

	opts := memoize.Opts().
		WithTTL(defaultDuration(cfg.FreshTTL, defaultFreshTTL)).
		WithStaleTTL(defaultDuration(cfg.StaleTTL, defaultStaleTTL)).
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

func defaultDuration(value, fallback time.Duration) time.Duration {
	if value == 0 {
		return fallback
	}
	return value
}
