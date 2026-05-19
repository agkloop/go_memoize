package configsnapshot

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestConfigServiceLoadsBeeceptorCompanySnapshot(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/companies" {
			t.Fatalf("path = %q, want /companies", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"id":1,"name":"Acme Corp","industry":"Manufacturing"},
			{"id":2,"name":"Globex","industry":"Logistics"}
		]`))
	}))
	defer server.Close()

	source, err := NewBeeceptorConfigSource(BeeceptorConfigSourceConfig{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewBeeceptorConfigSource failed: %v", err)
	}
	service, err := StartConfigService(context.Background(), ConfigServiceConfig{Source: source, RefreshInterval: time.Hour})
	if err != nil {
		t.Fatalf("StartConfigService failed: %v", err)
	}
	defer service.Close()

	cfg := service.Current()
	if cfg.Version != "beeceptor-companies" || !cfg.FeatureFlags["industry:Manufacturing"] || !cfg.FeatureFlags["industry:Logistics"] {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	cfg.FeatureFlags["industry:Manufacturing"] = false
	if !service.Current().FeatureFlags["industry:Manufacturing"] {
		t.Fatal("Current returned mutable shared FeatureFlags map")
	}
}
