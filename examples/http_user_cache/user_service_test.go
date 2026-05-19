package httpusercache

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestUserServiceCachesBeeceptorUsersEndpoint(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/users" {
			t.Fatalf("path = %q, want /users", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"id":1,"name":"Ada Lovelace","email":"ada@example.test","company":"Analytical Engines"},
			{"id":2,"name":"Grace Hopper","email":"grace@example.test","company":"Compilers Inc"}
		]`))
	}))
	defer server.Close()

	service, err := NewUserService(UserServiceConfig{BaseURL: server.URL, FreshTTL: time.Minute})
	if err != nil {
		t.Fatalf("NewUserService failed: %v", err)
	}
	defer service.Close()

	ctx := context.Background()
	first, err := service.GetUser(ctx, 1)
	if err != nil {
		t.Fatalf("first GetUser failed: %v", err)
	}
	second, err := service.GetUser(ctx, 1)
	if err != nil {
		t.Fatalf("second GetUser failed: %v", err)
	}

	if first.Name != "Ada Lovelace" || second.Name != first.Name {
		t.Fatalf("unexpected cached user: first=%+v second=%+v", first, second)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}
