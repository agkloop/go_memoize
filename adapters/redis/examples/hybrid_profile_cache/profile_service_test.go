package hybridprofilecache

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/agkloop/go_memoize/stores/chain"
	"github.com/agkloop/go_memoize/stores/memory"
)

func TestProfileServiceCachesThroughHybridStore(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/users" {
			t.Fatalf("path = %q, want /users", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"id":9,"name":"Margaret Hamilton","email":"margaret@example.test","company":"Apollo Guidance"}
		]`))
	}))
	defer server.Close()

	repo, err := NewBeeceptorProfileRepository(BeeceptorProfileRepositoryConfig{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewBeeceptorProfileRepository failed: %v", err)
	}
	hybrid := chain.New[uint64, Profile](memory.New[uint64, Profile](16), memory.New[uint64, Profile](16))
	service, err := NewProfileServiceWithStore(ProfileServiceStoreConfig{Repository: repo, Store: hybrid, FreshTTL: time.Minute})
	if err != nil {
		t.Fatalf("NewProfileServiceWithStore failed: %v", err)
	}

	ctx := context.Background()
	first, err := service.GetProfile(ctx, 9)
	if err != nil {
		t.Fatalf("first GetProfile failed: %v", err)
	}
	second, err := service.GetProfile(ctx, 9)
	if err != nil {
		t.Fatalf("second GetProfile failed: %v", err)
	}

	if first.DisplayName != "Margaret Hamilton" || second != first {
		t.Fatalf("unexpected cached profile: first=%+v second=%+v", first, second)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}
