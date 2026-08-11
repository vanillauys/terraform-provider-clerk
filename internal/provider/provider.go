package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vanillauys/terraform-provider-clerk/internal/clerkapi"
	dsdomains "github.com/vanillauys/terraform-provider-clerk/internal/datasources/domains"
	dsjwks "github.com/vanillauys/terraform-provider-clerk/internal/datasources/jwksds"
	dsorganization "github.com/vanillauys/terraform-provider-clerk/internal/datasources/organizationds"
	"github.com/vanillauys/terraform-provider-clerk/internal/resources/apikey"
	"github.com/vanillauys/terraform-provider-clerk/internal/resources/domainres"
	"github.com/vanillauys/terraform-provider-clerk/internal/resources/identifier"
	"github.com/vanillauys/terraform-provider-clerk/internal/resources/machine"
	"github.com/vanillauys/terraform-provider-clerk/internal/resources/organization"
	"github.com/vanillauys/terraform-provider-clerk/internal/resources/orgdomain"
	"github.com/vanillauys/terraform-provider-clerk/internal/resources/orgmembership"
	"github.com/vanillauys/terraform-provider-clerk/internal/resources/orgpermission"
	"github.com/vanillauys/terraform-provider-clerk/internal/resources/orgrole"
	"github.com/vanillauys/terraform-provider-clerk/internal/resources/webhook"
	"github.com/vanillauys/terraform-provider-clerk/internal/resources/instanceorgsettings"
	"github.com/vanillauys/terraform-provider-clerk/internal/resources/instancerestrictions"
	"github.com/vanillauys/terraform-provider-clerk/internal/resources/instancesettings"
	"github.com/vanillauys/terraform-provider-clerk/internal/resources/jwttemplate"
	"github.com/vanillauys/terraform-provider-clerk/internal/resources/redirecturl"
)

var _ provider.Provider = (*ClerkProvider)(nil)

type resolvedConfig struct {
	secretKey string
	apiURL    string
}

// resolveConfig applies the CLERK_SECRET_KEY / CLERK_API_URL fallbacks and
// reports which required settings are still missing.
func resolveConfig(m ClerkProviderModel, getenv func(string) string) (resolvedConfig, []string) {
	rc := resolvedConfig{
		secretKey: m.SecretKey.ValueString(),
		apiURL:    m.APIURL.ValueString(),
	}
	if rc.secretKey == "" {
		rc.secretKey = getenv("CLERK_SECRET_KEY")
	}
	if rc.apiURL == "" {
		rc.apiURL = getenv("CLERK_API_URL")
	}
	var missing []string
	if rc.secretKey == "" {
		missing = append(missing, `"secret_key" (or the CLERK_SECRET_KEY environment variable)`)
	}
	return rc, missing
}

type ClerkProvider struct {
	version string
}

type ClerkProviderModel struct {
	SecretKey types.String `tfsdk:"secret_key"`
	APIURL    types.String `tfsdk:"api_url"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ClerkProvider{version: version}
	}
}

func (p *ClerkProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "clerk"
	resp.Version = p.version
}

func (p *ClerkProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage the configuration of a Clerk instance through the Clerk Backend API. " +
			"One provider block targets one instance; use a provider alias for a second instance " +
			"(for example dev and prod).",
		Attributes: map[string]schema.Attribute{
			"secret_key": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				Description: "Clerk secret key for the instance (`sk_test_...` or `sk_live_...`). " +
					"Falls back to the `CLERK_SECRET_KEY` environment variable.",
			},
			"api_url": schema.StringAttribute{
				Optional: true,
				Description: "Base URL of the Clerk Backend API. Falls back to the `CLERK_API_URL` " +
					"environment variable, then to `https://api.clerk.com/v1`.",
			},
		},
	}
}

func (p *ClerkProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config ClerkProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rc, missing := resolveConfig(config, os.Getenv)
	for _, m := range missing {
		resp.Diagnostics.AddError(
			"Missing provider configuration",
			fmt.Sprintf("The %s must be set.", m),
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}
	c := clerkapi.New(rc.secretKey, rc.apiURL, p.version)
	resp.ResourceData = c
	resp.DataSourceData = c
}

func (p *ClerkProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		jwttemplate.NewResource,
		redirecturl.NewResource,
		identifier.NewAllowlistResource,
		identifier.NewBlocklistResource,
		instancesettings.NewResource,
		instancerestrictions.NewResource,
		instanceorgsettings.NewResource,
		apikey.NewResource,
		machine.NewResource,
		domainres.NewResource,
		webhook.NewResource,
		organization.NewResource,
		orgpermission.NewResource,
		orgrole.NewResource,
		orgdomain.NewResource,
		orgmembership.NewResource,
	}
}

func (p *ClerkProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		dsdomains.NewDataSource,
		dsjwks.NewDataSource,
		dsorganization.NewDataSource,
	}
}
