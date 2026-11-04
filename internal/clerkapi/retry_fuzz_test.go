package clerkapi

import (
	"testing"
)

// FuzzRetryAfter feeds retryAfter the Retry-After header that a server can
// send. Every wait must stay between 0 and maxWait, whatever the header
// holds. The unit job runs the seeds; the nightly job explores new inputs.
func FuzzRetryAfter(f *testing.F) {
	for _, seed := range []string{"", "0", "1", "30", "31", "600", "-1", "soon", " 5", "9223372036854775807", "Wed, 21 Oct 2026 07:28:00 GMT"} {
		f.Add(seed, uint8(0))
		f.Add(seed, uint8(maxRetries))
	}
	f.Fuzz(func(t *testing.T, header string, attempt uint8) {
		// RoundTrip calls retryAfter only for attempts 0 to maxRetries-1.
		a := int(attempt) % maxRetries
		got := retryAfter(header, a)
		if got < 0 || got > maxWait {
			t.Fatalf("retryAfter(%q, %d) = %v, want a wait between 0 and %v", header, a, got, maxWait)
		}
		if got == 0 && header == "" {
			t.Fatalf("retryAfter(%q, %d) = 0, want the backoff", header, a)
		}
	})
}
