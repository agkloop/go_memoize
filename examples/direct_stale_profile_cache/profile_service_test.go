package directstaleprofilecache

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestProfileServiceCachesProfileWithDirectStaleMemoizer(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/users" {
			t.Fatalf("path = %q, want /users", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":11,"name":"Grace Hopper","email":"grace@example.com","company":"Navy"}]`))
	}))
	defer server.Close()

	repo, err := NewBeeceptorProfileRepository(BeeceptorProfileRepositoryConfig{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewBeeceptorProfileRepository failed: %v", err)
	}
	service, err := NewProfileService(ProfileServiceConfig{
		Repository: repo,
		FreshTTL:   time.Minute,
		StaleTTL:   5 * time.Minute,
	})
	if err != nil {
		t.Fatalf("NewProfileService failed: %v", err)
	}

	ctx := context.Background()
	first, err := service.GetProfile(ctx, 11)
	if err != nil {
		t.Fatalf("first GetProfile failed: %v", err)
	}
	second, err := service.GetProfile(ctx, 11)
	if err != nil {
		t.Fatalf("second GetProfile failed: %v", err)
	}

	want := Profile{ID: 11, DisplayName: "Grace Hopper", Email: "grace@example.com", Company: "Navy"}
	if first != want || second != first {
		t.Fatalf("unexpected cached profile: first=%+v second=%+v want=%+v", first, second, want)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}
