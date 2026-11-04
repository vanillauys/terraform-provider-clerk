package clerkapi

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRetryTransportRetries429(t *testing.T) {
	var calls int
	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		b, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(b))
		if calls < 3 {
			w.Header().Set("Retry-After", "2")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	var waits []time.Duration
	tr := &retryTransport{base: http.DefaultTransport, sleep: func(_ *http.Request, d time.Duration) error {
		waits = append(waits, d)
		return nil
	}}
	req, _ := http.NewRequest(http.MethodPatch, srv.URL, strings.NewReader(`{"a":1}`))
	resp, err := (&http.Client{Transport: tr}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK || calls != 3 {
		t.Fatalf("status %d after %d calls, want 200 after 3", resp.StatusCode, calls)
	}
	for i, b := range bodies {
		if b != `{"a":1}` {
			t.Errorf("attempt %d sent body %q, want the original body", i+1, b)
		}
	}
	if len(waits) != 2 || waits[0] != 2*time.Second {
		t.Errorf("waits %v, want two waits of the Retry-After 2s", waits)
	}
}

func TestRetryTransportStopsAfterMaxRetries(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	tr := &retryTransport{base: http.DefaultTransport, sleep: func(*http.Request, time.Duration) error { return nil }}
	resp, err := (&http.Client{Transport: tr}).Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusTooManyRequests || calls != maxRetries+1 {
		t.Fatalf("status %d after %d calls, want 429 after %d", resp.StatusCode, calls, maxRetries+1)
	}
}

func TestRetryAfter(t *testing.T) {
	cases := []struct {
		header  string
		attempt int
		want    time.Duration
	}{
		{"", 0, time.Second},
		{"", 3, 8 * time.Second},
		{"", 10, maxWait},
		{"5", 0, 5 * time.Second},
		{"600", 0, maxWait},
		{"soon", 1, 2 * time.Second},
	}
	for _, c := range cases {
		if got := retryAfter(c.header, c.attempt); got != c.want {
			t.Errorf("retryAfter(%q, %d) = %v, want %v", c.header, c.attempt, got, c.want)
		}
	}
}
