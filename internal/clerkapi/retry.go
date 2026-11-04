package clerkapi

import (
	"net/http"
	"strconv"
	"time"
)

// maxRetries is the number of retries after a 429 response. With the
// backoff below, the last retry starts about 30 seconds after the first call.
const maxRetries = 5

// maxWait caps one wait, also when Retry-After asks for more.
const maxWait = 30 * time.Second

// HTTPClient returns the HTTP client for the Clerk Backend API. It retries a
// rate-limited request (429). Clerk rate-limits each instance, and a large
// apply or the acceptance suite can reach the limit.
//
// The client has no overall timeout: the wait between retries would count
// against it. The context of the Terraform operation stops the call, and
// ResponseHeaderTimeout stops one attempt that hangs.
func HTTPClient() *http.Client {
	base := http.DefaultTransport.(*http.Transport).Clone()
	base.ResponseHeaderTimeout = 30 * time.Second
	return &http.Client{Transport: &retryTransport{base: base, sleep: sleepContext}}
}

type retryTransport struct {
	base  http.RoundTripper
	sleep func(*http.Request, time.Duration) error
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	for attempt := 0; ; attempt++ {
		resp, err := t.base.RoundTrip(req)
		if err != nil || resp.StatusCode != http.StatusTooManyRequests || attempt == maxRetries {
			return resp, err
		}
		// A request with a body can repeat only when the body can be read
		// again. The SDK sets GetBody on each request with a body.
		if req.Body != nil && req.GetBody == nil {
			return resp, nil
		}
		wait := retryAfter(resp.Header.Get("Retry-After"), attempt)
		_ = resp.Body.Close()
		if err := t.sleep(req, wait); err != nil {
			return nil, err
		}
		if req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return nil, err
			}
			req = req.Clone(req.Context())
			req.Body = body
		}
	}
}

// retryAfter returns the wait before the next attempt: the Retry-After
// seconds when Clerk sends them, else 1, 2, 4, 8, and 16 seconds.
func retryAfter(header string, attempt int) time.Duration {
	wait := time.Second << attempt
	if s, err := strconv.Atoi(header); err == nil && s >= 0 {
		wait = time.Duration(s) * time.Second
	}
	return min(wait, maxWait)
}

func sleepContext(req *http.Request, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-req.Context().Done():
		return req.Context().Err()
	case <-timer.C:
		return nil
	}
}
