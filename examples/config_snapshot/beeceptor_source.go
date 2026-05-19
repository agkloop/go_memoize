package configsnapshot

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

const BeeceptorCompaniesURL = "https://fake-json-api.mock.beeceptor.com/companies"

var ErrMissingBaseURL = errors.New("configsnapshot: missing base URL")

type BeeceptorConfigSourceConfig struct {
	BaseURL        string
	HTTPClient     HTTPClient
	RequestTimeout time.Duration
}

type BeeceptorConfigSource struct {
	baseURL        *url.URL
	client         HTTPClient
	requestTimeout time.Duration
}

func NewBeeceptorConfigSource(cfg BeeceptorConfigSourceConfig) (*BeeceptorConfigSource, error) {
	baseURL, err := parseBaseURL(defaultString(cfg.BaseURL, "https://fake-json-api.mock.beeceptor.com"))
	if err != nil {
		return nil, err
	}
	client := cfg.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	return &BeeceptorConfigSource{baseURL: baseURL, client: client, requestTimeout: defaultDuration(cfg.RequestTimeout, defaultRequestTimeout)}, nil
}

func (s *BeeceptorConfigSource) LoadConfig(ctx context.Context) (AppConfig, error) {
	ctx, cancel := context.WithTimeout(ctx, s.requestTimeout)
	defer cancel()

	endpoint := s.baseURL.JoinPath("companies")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return AppConfig{}, err
	}
	req.Header.Set("Accept", "application/json")
	res, err := s.client.Do(req)
	if err != nil {
		return AppConfig{}, err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return AppConfig{}, fmt.Errorf("configsnapshot: Beeceptor companies status %d", res.StatusCode)
	}

	var companies []struct {
		Name     string `json:"name"`
		Industry string `json:"industry"`
	}
	if err := json.NewDecoder(res.Body).Decode(&companies); err != nil {
		return AppConfig{}, err
	}
	flags := map[string]bool{"companies_loaded": len(companies) > 0}
	for _, company := range companies {
		if company.Industry != "" {
			flags["industry:"+company.Industry] = true
		}
	}
	return AppConfig{Version: "beeceptor-companies", FeatureFlags: flags, UpdatedAt: time.Now().UTC()}, nil
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
		return nil, fmt.Errorf("configsnapshot: invalid base URL %q", raw)
	}
	return baseURL, nil
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
