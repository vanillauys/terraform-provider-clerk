// Package clerkapi bundles the per-API clients of the Clerk Backend API
// behind one provider-facing type.
package clerkapi

import (
	"errors"
	"net/http"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/allowlistidentifier"
	"github.com/clerk/clerk-sdk-go/v2/blocklistidentifier"
	"github.com/clerk/clerk-sdk-go/v2/domain"
	"github.com/clerk/clerk-sdk-go/v2/jwttemplate"
	"github.com/clerk/clerk-sdk-go/v2/redirecturl"
)

// Client holds one configured client per Clerk API group. Each provider
// block (each alias included) gets its own Client, so one plan can target
// the dev instance and the prod instance at the same time. The global
// clerk.SetKey is deliberately not used for that reason.
type Client struct {
	JWTTemplates *jwttemplate.Client
	RedirectURLs *redirecturl.Client
	Allowlist    *allowlistidentifier.Client
	Blocklist    *blocklistidentifier.Client
	Domains      *domain.Client
}

// New builds a Client for one Clerk instance. An empty apiURL keeps the
// default https://api.clerk.com/v1.
func New(secretKey, apiURL, version string) *Client {
	cfg := &clerk.ClientConfig{}
	cfg.Key = clerk.String(secretKey)
	if apiURL != "" {
		cfg.URL = clerk.String(apiURL)
	}
	cfg.CustomRequestHeaders = &clerk.CustomRequestHeaders{
		Application: "terraform-provider-clerk/" + version,
	}
	return &Client{
		JWTTemplates: jwttemplate.NewClient(cfg),
		RedirectURLs: redirecturl.NewClient(cfg),
		Allowlist:    allowlistidentifier.NewClient(cfg),
		Blocklist:    blocklistidentifier.NewClient(cfg),
		Domains:      domain.NewClient(cfg),
	}
}

// IsNotFound reports whether err is a Clerk API response with HTTP status
// 404. Read uses it to drop a deleted resource from state, and Delete uses
// it to treat an already-deleted resource as success.
func IsNotFound(err error) bool {
	var apiErr *clerk.APIErrorResponse
	return errors.As(err, &apiErr) && apiErr.HTTPStatusCode == http.StatusNotFound
}
