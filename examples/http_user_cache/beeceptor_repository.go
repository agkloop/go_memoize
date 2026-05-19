package httpusercache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const BeeceptorUsersURL = "https://fake-json-api.mock.beeceptor.com/users"

var (
	ErrMissingBaseURL = errors.New("httpusercache: missing base URL")
	ErrUserNotFound   = errors.New("httpusercache: user not found")
)

type BeeceptorUserRepositoryConfig struct {
	BaseURL        string
	HTTPClient     HTTPClient
	RequestTimeout time.Duration
	Logger         Logger
}

type BeeceptorUserRepository struct {
	baseURL        *url.URL
	client         HTTPClient
	requestTimeout time.Duration
	logger         Logger
}

func NewBeeceptorUserRepository(cfg BeeceptorUserRepositoryConfig) (*BeeceptorUserRepository, error) {
	baseURL, err := parseBaseURL(defaultString(cfg.BaseURL, "https://fake-json-api.mock.beeceptor.com"))
	if err != nil {
		return nil, err
	}
	client := cfg.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	return &BeeceptorUserRepository{
		baseURL:        baseURL,
		client:         client,
		requestTimeout: defaultDuration(cfg.RequestTimeout, defaultRequestTimeout),
		logger:         cfg.Logger,
	}, nil
}

func (r *BeeceptorUserRepository) LoadUser(ctx context.Context, userID int64) (User, error) {
	ctx, cancel := context.WithTimeout(ctx, r.requestTimeout)
	defer cancel()

	users, err := r.fetchUsers(ctx)
	if err != nil {
		return User{}, err
	}
	for _, user := range users {
		if user.ID == userID {
			return user, nil
		}
	}
	if r.logger != nil {
		r.logger.Printf("beeceptor user not found id=%d", userID)
	}
	return User{}, ErrUserNotFound
}

func (r *BeeceptorUserRepository) fetchUsers(ctx context.Context) ([]User, error) {
	endpoint := r.baseURL.JoinPath("users")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	res, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("httpusercache: Beeceptor users status %d", res.StatusCode)
	}

	var users []User
	if err := json.NewDecoder(res.Body).Decode(&users); err != nil {
		return nil, err
	}
	return users, nil
}

func parseBaseURL(raw string) (*url.URL, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, ErrMissingBaseURL
	}
	baseURL, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, fmt.Errorf("httpusercache: invalid base URL %q", raw)
	}
	return baseURL, nil
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
