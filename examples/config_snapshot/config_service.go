package configsnapshot

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/agkloop/go_memoize/background"
)

const (
	defaultRefreshInterval = time.Minute
	defaultRequestTimeout  = 2 * time.Second
)

var ErrMissingSource = errors.New("configsnapshot: missing source")

type Logger interface {
	Printf(format string, args ...any)
}

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type AppConfig struct {
	Version      string
	FeatureFlags map[string]bool
	UpdatedAt    time.Time
}

type ConfigSource interface {
	LoadConfig(context.Context) (AppConfig, error)
}

type ConfigServiceConfig struct {
	Source          ConfigSource
	RefreshInterval time.Duration
	Logger          Logger
}

type ConfigService struct {
	cancel context.CancelFunc
	value  *background.Value[AppConfig]
}

func StartConfigService(ctx context.Context, cfg ConfigServiceConfig) (*ConfigService, error) {
	if cfg.Source == nil {
		return nil, ErrMissingSource
	}
	runCtx, cancel := context.WithCancel(ctx)
	value, err := background.Keep(runCtx, func(ctx context.Context) (AppConfig, error) {
		loaded, err := cfg.Source.LoadConfig(ctx)
		if err != nil {
			return AppConfig{}, err
		}
		return cloneConfig(loaded), nil
	}, defaultDuration(cfg.RefreshInterval, defaultRefreshInterval),
		background.OnError[AppConfig](func(err error) {
			if cfg.Logger != nil {
				cfg.Logger.Printf("config refresh failed: %v", err)
			}
		}),
		background.OnRefresh[AppConfig](func(AppConfig) {
			if cfg.Logger != nil {
				cfg.Logger.Printf("config refreshed")
			}
		}),
	)
	if err != nil {
		cancel()
		return nil, err
	}
	return &ConfigService{cancel: cancel, value: value}, nil
}

func (s *ConfigService) Current() AppConfig {
	return cloneConfig(s.value.Get())
}

func (s *ConfigService) Close() {
	s.cancel()
}

func cloneConfig(cfg AppConfig) AppConfig {
	flags := make(map[string]bool, len(cfg.FeatureFlags))
	for key, value := range cfg.FeatureFlags {
		flags[key] = value
	}
	cfg.FeatureFlags = flags
	return cfg
}

func defaultDuration(value, fallback time.Duration) time.Duration {
	if value == 0 {
		return fallback
	}
	return value
}
