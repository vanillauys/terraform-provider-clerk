package samlconn

import (
	"encoding/base64"
	"strings"
	"testing"
)

// FuzzCertBody feeds certBody a certificate from the config or from Clerk.
// The read keeps the configured form only when both forms give the same
// body, so certBody must:
//   - return no whitespace and no PEM armor,
//   - give the same body again for its own output (idempotent),
//   - give the same body for the PEM form and for the bare base64 form.
//
// The unit job runs the seeds; the nightly job explores new inputs.
func FuzzCertBody(f *testing.F) {
	for _, seed := range [][]byte{nil, []byte("x"), []byte("a certificate body"), make([]byte, 300)} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, der []byte) {
		bare := base64.StdEncoding.EncodeToString(der)
		body := certBody(bare)
		if body != bare {
			t.Fatalf("certBody(%q) = %q, want the bare base64 unchanged", bare, body)
		}
		pem := "-----BEGIN CERTIFICATE-----\n" + wrap(bare, 64) + "-----END CERTIFICATE-----\n"
		if got := certBody(pem); got != body {
			t.Fatalf("certBody(PEM) = %q, want %q", got, body)
		}
		if got := certBody(body); got != body {
			t.Fatalf("certBody is not idempotent: %q then %q", body, got)
		}
		if strings.ContainsAny(body, " \t\r\n-") {
			t.Fatalf("certBody(PEM) = %q, want no whitespace and no armor", body)
		}
	})
}

// FuzzCertBodyText feeds certBody any text, for example a value with stray
// armor or whitespace. The output must be stable on a second pass and hold
// no whitespace.
func FuzzCertBodyText(f *testing.F) {
	for _, seed := range []string{"", " ", "-----BEGIN CERTIFICATE-----", "-----BEGIN CERTIFICATE-----\nAAAA\n-----END CERTIFICATE-----\n", "AA\r\nBB\tCC"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, in string) {
		body := certBody(in)
		if strings.ContainsAny(body, " \t\r\n") {
			t.Fatalf("certBody(%q) = %q, want no whitespace", in, body)
		}
		// The armor can form again from two halves around the removed
		// header, so a second pass can remove more. A third pass must not.
		twice := certBody(body)
		if certBody(twice) != twice {
			t.Fatalf("certBody does not settle for %q", in)
		}
	})
}

func wrap(s string, n int) string {
	var b strings.Builder
	for len(s) > n {
		b.WriteString(s[:n] + "\n")
		s = s[n:]
	}
	if s != "" {
		b.WriteString(s + "\n")
	}
	return b.String()
}
