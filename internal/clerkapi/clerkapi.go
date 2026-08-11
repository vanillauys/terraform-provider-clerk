// Package clerkapi bundles the per-API clients of the Clerk Backend API
// behind one provider-facing type.
package clerkapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/allowlistidentifier"
	"github.com/clerk/clerk-sdk-go/v2/apikey"
	"github.com/clerk/clerk-sdk-go/v2/blocklistidentifier"
	"github.com/clerk/clerk-sdk-go/v2/domain"
	"github.com/clerk/clerk-sdk-go/v2/instancesettings"
	"github.com/clerk/clerk-sdk-go/v2/jwks"
	"github.com/clerk/clerk-sdk-go/v2/jwttemplate"
	"github.com/clerk/clerk-sdk-go/v2/machine"
	"github.com/clerk/clerk-sdk-go/v2/machinescope"
	"github.com/clerk/clerk-sdk-go/v2/oauthapplication"
	"github.com/clerk/clerk-sdk-go/v2/organization"
	"github.com/clerk/clerk-sdk-go/v2/organizationdomain"
	"github.com/clerk/clerk-sdk-go/v2/organizationmembership"
	"github.com/clerk/clerk-sdk-go/v2/organizationpermission"
	"github.com/clerk/clerk-sdk-go/v2/organizationrole"
	"github.com/clerk/clerk-sdk-go/v2/redirecturl"
	"github.com/clerk/clerk-sdk-go/v2/samlconnection"
	"github.com/clerk/clerk-sdk-go/v2/svixwebhook"
)

// Client holds one configured client per Clerk API group. Each provider
// block (each alias included) gets its own Client, so one plan can target
// the dev instance and the prod instance at the same time. The global
// clerk.SetKey is deliberately not used for that reason.
type Client struct {
	JWTTemplates     *jwttemplate.Client
	RedirectURLs     *redirecturl.Client
	Allowlist        *allowlistidentifier.Client
	Blocklist        *blocklistidentifier.Client
	Domains          *domain.Client
	InstanceSettings *instancesettings.Client
	APIKeys          *apikey.Client
	Machines         *machine.Client
	MachineScopes    *machinescope.Client
	SvixWebhooks     *svixwebhook.Client
	JWKS             *jwks.Client
	Organizations    *organization.Client
	OrgRoles         *organizationrole.Client
	OrgPermissions   *organizationpermission.Client
	OrgDomains       *organizationdomain.Client
	OrgMemberships   *organizationmembership.Client
	OAuthApps        *oauthapplication.Client
	SAMLConnections  *samlconnection.Client

	// backend serves the raw calls below for the endpoints that the SDK
	// does not wrap yet.
	backend clerk.Backend
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
		JWTTemplates:     jwttemplate.NewClient(cfg),
		RedirectURLs:     redirecturl.NewClient(cfg),
		Allowlist:        allowlistidentifier.NewClient(cfg),
		Blocklist:        blocklistidentifier.NewClient(cfg),
		Domains:          domain.NewClient(cfg),
		InstanceSettings: instancesettings.NewClient(cfg),
		APIKeys:          apikey.NewClient(cfg),
		Machines:         machine.NewClient(cfg),
		MachineScopes:    machinescope.NewClient(cfg),
		SvixWebhooks:     svixwebhook.NewClient(cfg),
		JWKS:             jwks.NewClient(cfg),
		Organizations:    organization.NewClient(cfg),
		OrgRoles:         organizationrole.NewClient(cfg),
		OrgPermissions:   organizationpermission.NewClient(cfg),
		OrgDomains:       organizationdomain.NewClient(cfg),
		OrgMemberships:   organizationmembership.NewClient(cfg),
		OAuthApps:        oauthapplication.NewClient(cfg),
		SAMLConnections:  samlconnection.NewClient(cfg),
		backend:          clerk.NewBackend(&cfg.BackendConfig),
	}
}

// Instance is the response of GET /instance. The SDK (v2.7.0) does not
// wrap this endpoint; remove this raw call when it does.
type Instance struct {
	clerk.APIResource
	ID              string   `json:"id"`
	EnvironmentType string   `json:"environment_type"`
	AllowedOrigins  []string `json:"allowed_origins"`
}

// GetInstance calls GET /instance.
func (c *Client) GetInstance(ctx context.Context) (*Instance, error) {
	req := clerk.NewAPIRequest(http.MethodGet, "/instance")
	instance := &Instance{}
	err := c.backend.Call(ctx, req, instance)
	return instance, err
}

type updateAllowedOriginsParams struct {
	clerk.APIParams
	// No omitempty: an empty list must reach the API to clear the origins.
	AllowedOrigins []string `json:"allowed_origins"`
}

// UpdateAllowedOrigins calls PATCH /instance with only allowed_origins.
// The SDK's instancesettings.UpdateParams does not carry this field yet.
func (c *Client) UpdateAllowedOrigins(ctx context.Context, origins []string) error {
	if origins == nil {
		// Send [] rather than null: the empty array clears the origins.
		origins = []string{}
	}
	req := clerk.NewAPIRequest(http.MethodPatch, "/instance")
	req.SetParams(&updateAllowedOriginsParams{AllowedOrigins: origins})
	return c.backend.Call(ctx, req, &clerk.APIResource{})
}

// IsNotFound reports whether err is a Clerk API response with HTTP status
// 404. Read uses it to drop a deleted resource from state, and Delete uses
// it to treat an already-deleted resource as success.
func IsNotFound(err error) bool {
	var apiErr *clerk.APIErrorResponse
	return errors.As(err, &apiErr) && apiErr.HTTPStatusCode == http.StatusNotFound
}
