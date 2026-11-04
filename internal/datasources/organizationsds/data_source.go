// Package organizationsds provides the clerk_organizations data source: all
// organizations of the instance, with an optional search query.
package organizationsds

import (
	"context"

	"github.com/clerk/clerk-sdk-go/v2"
	sdkorganization "github.com/clerk/clerk-sdk-go/v2/organization"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vanillauys/terraform-provider-clerk/internal/clerkapi"
	"github.com/vanillauys/terraform-provider-clerk/internal/tfutil"
)

// pageSize is the maximum page size of GET /organizations.
const pageSize = 500

var (
	_ datasource.DataSource              = (*organizationsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*organizationsDataSource)(nil)
)

type organizationsDataSource struct {
	client *clerkapi.Client
}

type dataSourceModel struct {
	Query         types.String        `tfsdk:"query"`
	Organizations []organizationModel `tfsdk:"organizations"`
}

type organizationModel struct {
	ID                    types.String `tfsdk:"id"`
	Name                  types.String `tfsdk:"name"`
	Slug                  types.String `tfsdk:"slug"`
	MaxAllowedMemberships types.Int64  `tfsdk:"max_allowed_memberships"`
	AdminDeleteEnabled    types.Bool   `tfsdk:"admin_delete_enabled"`
}

func NewDataSource() datasource.DataSource { return &organizationsDataSource{} }

func (d *organizationsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organizations"
}

func (d *organizationsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "All organizations of the instance, newest first.",
		Attributes: map[string]schema.Attribute{
			"query": schema.StringAttribute{
				Optional: true,
				Description: "Search text. Clerk matches it against the organization id, name, and slug. " +
					"When you do not set it, the data source returns every organization.",
			},
			"organizations": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Organizations that match the query.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "Organization id.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Organization name.",
						},
						"slug": schema.StringAttribute{
							Computed:    true,
							Description: "Organization slug.",
						},
						"max_allowed_memberships": schema.Int64Attribute{
							Computed:    true,
							Description: "Maximum members. `0` means unlimited.",
						},
						"admin_delete_enabled": schema.BoolAttribute{
							Computed:    true,
							Description: "`true` when organization admins can delete the organization.",
						},
					},
				},
			},
		},
	}
}

func (d *organizationsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := tfutil.ClientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if c != nil {
		d.client = c
	}
}

func (d *organizationsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state dataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	params := &sdkorganization.ListParams{
		ListParams: clerk.ListParams{Limit: clerk.Int64(pageSize), Offset: clerk.Int64(0)},
		Query:      state.Query.ValueStringPointer(),
	}
	state.Organizations = []organizationModel{}
	for {
		list, err := d.client.Organizations.List(ctx, params)
		if err != nil {
			resp.Diagnostics.AddError("Listing organizations", err.Error())
			return
		}
		for _, o := range list.Organizations {
			state.Organizations = append(state.Organizations, organizationModel{
				ID:                    types.StringValue(o.ID),
				Name:                  types.StringValue(o.Name),
				Slug:                  types.StringValue(o.Slug),
				MaxAllowedMemberships: types.Int64Value(o.MaxAllowedMemberships),
				AdminDeleteEnabled:    types.BoolValue(o.AdminDeleteEnabled),
			})
		}
		if len(list.Organizations) < pageSize || int64(len(state.Organizations)) >= list.TotalCount {
			break
		}
		*params.Offset += pageSize
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
