// Package domains provides the clerk_domains data source. It lists the
// domains of the instance with the CNAME targets that a production instance
// needs in DNS, so a Terraform stack can feed them straight into its DNS
// provider.
package domains

import (
	"context"

	sdkdomain "github.com/clerk/clerk-sdk-go/v2/domain"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vanillauys/terraform-provider-clerk/internal/clerkapi"
	"github.com/vanillauys/terraform-provider-clerk/internal/tfutil"
)

var (
	_ datasource.DataSource              = (*domainsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*domainsDataSource)(nil)
)

type domainsDataSource struct {
	client *clerkapi.Client
}

type dataSourceModel struct {
	Domains []domainModel `tfsdk:"domains"`
}

type domainModel struct {
	ID                types.String       `tfsdk:"id"`
	Name              types.String       `tfsdk:"name"`
	IsSatellite       types.Bool         `tfsdk:"is_satellite"`
	FrontendAPIURL    types.String       `tfsdk:"frontend_api_url"`
	AccountsPortalURL types.String       `tfsdk:"accounts_portal_url"`
	ProxyURL          types.String       `tfsdk:"proxy_url"`
	DevelopmentOrigin types.String       `tfsdk:"development_origin"`
	CNAMETargets      []cnameTargetModel `tfsdk:"cname_targets"`
}

type cnameTargetModel struct {
	Host  types.String `tfsdk:"host"`
	Value types.String `tfsdk:"value"`
}

func NewDataSource() datasource.DataSource { return &domainsDataSource{} }

func (d *domainsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domains"
}

func (d *domainsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "All domains of the instance, with the CNAME records that a production " +
			"instance needs in DNS. Feed `cname_targets` into your DNS provider to keep the " +
			"Clerk records in code.",
		Attributes: map[string]schema.Attribute{
			"domains": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Domains of the instance.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "Domain id.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Domain name, for example `example.com`.",
						},
						"is_satellite": schema.BoolAttribute{
							Computed:    true,
							Description: "`true` for a satellite domain.",
						},
						"frontend_api_url": schema.StringAttribute{
							Computed:    true,
							Description: "Frontend API URL for this domain.",
						},
						"accounts_portal_url": schema.StringAttribute{
							Computed:    true,
							Description: "Account Portal URL. Null for a satellite domain.",
						},
						"proxy_url": schema.StringAttribute{
							Computed:    true,
							Description: "Proxy URL. Null when the domain does not use a proxy.",
						},
						"development_origin": schema.StringAttribute{
							Computed:    true,
							Description: "Origin of the development instance.",
						},
						"cname_targets": schema.ListNestedAttribute{
							Computed:    true,
							Description: "CNAME records that this domain needs in DNS. Empty for a development instance.",
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"host": schema.StringAttribute{
										Computed:    true,
										Description: "Record host, for example `clerk.example.com`.",
									},
									"value": schema.StringAttribute{
										Computed:    true,
										Description: "Record target, for example `frontend-api.clerk.services`.",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *domainsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := tfutil.ClientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if c != nil {
		d.client = c
	}
}

func (d *domainsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	list, err := d.client.Domains.List(ctx, &sdkdomain.ListParams{})
	if err != nil {
		resp.Diagnostics.AddError("Listing domains", err.Error())
		return
	}
	state := dataSourceModel{Domains: make([]domainModel, 0, len(list.Domains))}
	for _, dom := range list.Domains {
		m := domainModel{
			ID:                types.StringValue(dom.ID),
			Name:              types.StringValue(dom.Name),
			IsSatellite:       types.BoolValue(dom.IsSatellite),
			FrontendAPIURL:    types.StringValue(dom.FrontendAPIURL),
			AccountsPortalURL: types.StringPointerValue(dom.AccountPortalURL),
			ProxyURL:          types.StringPointerValue(dom.ProxyURL),
			DevelopmentOrigin: types.StringValue(dom.DevelopmentOrigin),
			CNAMETargets:      make([]cnameTargetModel, 0, len(dom.CNAMETargets)),
		}
		for _, t := range dom.CNAMETargets {
			m.CNAMETargets = append(m.CNAMETargets, cnameTargetModel{
				Host:  types.StringValue(t.Host),
				Value: types.StringValue(t.Value),
			})
		}
		state.Domains = append(state.Domains, m)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
