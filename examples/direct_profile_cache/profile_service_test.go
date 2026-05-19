package directprofilecache

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestProfileServiceMemoizesBeeceptorUserProfile(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/users" {
			t.Fatalf("path = %q, want /users", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"id":7,"name":"Linus Torvalds","email":"linus@example.test","company":"Kernel Labs"}
		]`))
	}))
	defer server.Close()

	repo, err := NewBeeceptorProfileRepository(BeeceptorProfileRepositoryConfig{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewBeeceptorProfileRepository failed: %v", err)
	}
	service, err := NewProfileService(ProfileServiceConfig{Repository: repo, TTL: time.Minute})
	if err != nil {
		t.Fatalf("NewProfileService failed: %v", err)
	}

	ctx := context.Background()
	first, err := service.GetProfile(ctx, 7)
	if err != nil {
		t.Fatalf("first GetProfile failed: %v", err)
	}
	second, err := service.GetProfile(ctx, 7)
	if err != nil {
		t.Fatalf("second GetProfile failed: %v", err)
	}

	if first.DisplayName != "Linus Torvalds" || second != first {
		t.Fatalf("unexpected cached profile: first=%+v second=%+v", first, second)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}
