package directprofilecache

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
	ErrMissingBaseURL  = errors.New("directprofilecache: missing base URL")
	ErrProfileNotFound = errors.New("directprofilecache: profile not found")
)

type BeeceptorProfileRepositoryConfig struct {
	BaseURL        string
	HTTPClient     HTTPClient
	RequestTimeout time.Duration
}

type BeeceptorProfileRepository struct {
	baseURL        *url.URL
	client         HTTPClient
	requestTimeout time.Duration
}

func NewBeeceptorProfileRepository(cfg BeeceptorProfileRepositoryConfig) (*BeeceptorProfileRepository, error) {
	baseURL, err := parseBaseURL(defaultString(cfg.BaseURL, "https://fake-json-api.mock.beeceptor.com"))
	if err != nil {
		return nil, err
	}
	client := cfg.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	return &BeeceptorProfileRepository{baseURL: baseURL, client: client, requestTimeout: defaultDuration(cfg.RequestTimeout, defaultRequestTimeout)}, nil
}

func (r *BeeceptorProfileRepository) LoadProfile(ctx context.Context, profileID int64) (Profile, error) {
	ctx, cancel := context.WithTimeout(ctx, r.requestTimeout)
	defer cancel()

	endpoint := r.baseURL.JoinPath("users")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return Profile{}, err
	}
	req.Header.Set("Accept", "application/json")
	res, err := r.client.Do(req)
	if err != nil {
		return Profile{}, err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return Profile{}, fmt.Errorf("directprofilecache: Beeceptor users status %d", res.StatusCode)
	}

	var users []struct {
		ID      int64  `json:"id"`
		Name    string `json:"name"`
		Email   string `json:"email"`
		Company string `json:"company"`
	}
	if err := json.NewDecoder(res.Body).Decode(&users); err != nil {
		return Profile{}, err
	}
	for _, user := range users {
		if user.ID == profileID {
			return Profile{ID: user.ID, DisplayName: user.Name, Email: user.Email, Company: user.Company}, nil
		}
	}
	return Profile{}, ErrProfileNotFound
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
		return nil, fmt.Errorf("directprofilecache: invalid base URL %q", raw)
	}
	return baseURL, nil
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
