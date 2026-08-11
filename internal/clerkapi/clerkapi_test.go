package clerkapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/clerk/clerk-sdk-go/v2"
)

func TestIsNotFound(t *testing.T) {
	notFound := &clerk.APIErrorResponse{HTTPStatusCode: http.StatusNotFound}
	if !IsNotFound(notFound) {
		t.Error("IsNotFound(404 APIErrorResponse) = false, want true")
	}
	if !IsNotFound(fmt.Errorf("wrapped: %w", notFound)) {
		t.Error("IsNotFound(wrapped 404) = false, want true")
	}
	if IsNotFound(&clerk.APIErrorResponse{HTTPStatusCode: http.StatusUnprocessableEntity}) {
		t.Error("IsNotFound(422 APIErrorResponse) = true, want false")
	}
	if IsNotFound(errors.New("plain error")) {
		t.Error("IsNotFound(plain error) = true, want false")
	}
	if IsNotFound(nil) {
		t.Error("IsNotFound(nil) = true, want false")
	}
}

// TestNewClientWiring checks that New routes requests to the configured URL
// with the configured key, and that a 404 response surfaces as IsNotFound.
func TestNewClientWiring(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer sk_test_wiring" {
			t.Errorf("Authorization = %q, want Bearer sk_test_wiring", got)
		}
		switch r.URL.Path {
		case "/jwt_templates/jtmp_123":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"object":"jwt_template","id":"jtmp_123","name":"api","claims":{"role":"admin"},"lifetime":3600,"allowed_clock_skew":5}`)
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = fmt.Fprint(w, `{"errors":[{"code":"resource_not_found","message":"not found"}],"status":404}`)
		}
	}))
	defer ts.Close()

	c := New("sk_test_wiring", ts.URL, "test")

	tpl, err := c.JWTTemplates.Get(context.Background(), "jtmp_123")
	if err != nil {
		t.Fatalf("Get(jtmp_123) error: %v", err)
	}
	if tpl.Name != "api" || tpl.Lifetime != 3600 {
		t.Errorf("template = %+v, want name=api lifetime=3600", tpl)
	}

	_, err = c.JWTTemplates.Get(context.Background(), "jtmp_missing")
	if !IsNotFound(err) {
		t.Errorf("IsNotFound(%v) = false, want true", err)
	}
}

// TestRawInstanceCalls checks the raw calls that cover SDK gaps:
// GET /instance and the allowed_origins PATCH.
func TestRawInstanceCalls(t *testing.T) {
	var patchBody string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/instance" {
			t.Errorf("path = %q, want /instance", r.URL.Path)
		}
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"object":"instance","id":"ins_123","environment_type":"development","allowed_origins":["https://a.example.com"]}`)
		case http.MethodPatch:
			b, _ := io.ReadAll(r.Body)
			patchBody = string(b)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("method = %q", r.Method)
		}
	}))
	defer ts.Close()

	c := New("sk_test_wiring", ts.URL, "test")

	instance, err := c.GetInstance(context.Background())
	if err != nil {
		t.Fatalf("GetInstance error: %v", err)
	}
	if instance.ID != "ins_123" || instance.EnvironmentType != "development" || len(instance.AllowedOrigins) != 1 {
		t.Errorf("instance = %+v", instance)
	}

	if err := c.UpdateAllowedOrigins(context.Background(), []string{"https://b.example.com"}); err != nil {
		t.Fatalf("UpdateAllowedOrigins error: %v", err)
	}
	want := `{"allowed_origins":["https://b.example.com"]}`
	if patchBody != want {
		t.Errorf("PATCH body = %s, want %s", patchBody, want)
	}

	// An empty list must reach the API (it clears the origins).
	if err := c.UpdateAllowedOrigins(context.Background(), nil); err != nil {
		t.Fatalf("UpdateAllowedOrigins(nil) error: %v", err)
	}
	want = `{"allowed_origins":[]}`
	if patchBody != want {
		t.Errorf("PATCH body = %s, want %s", patchBody, want)
	}
}
