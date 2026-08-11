// Package organizationds provides the clerk_organization data source: an
// organization lookup by id or slug.
package organizationds

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vanillauys/terraform-provider-clerk/internal/clerkapi"
	"github.com/vanillauys/terraform-provider-clerk/internal/tfutil"
)

var (
	_ datasource.DataSource              = (*organizationDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*organizationDataSource)(nil)
)

type organizationDataSource struct {
	client *clerkapi.Client
}

type dataSourceModel struct {
	ID                    types.String `tfsdk:"id"`
	Lookup                types.String `tfsdk:"lookup"`
	Name                  types.String `tfsdk:"name"`
	Slug                  types.String `tfsdk:"slug"`
	MaxAllowedMemberships types.Int64  `tfsdk:"max_allowed_memberships"`
	AdminDeleteEnabled    types.Bool   `tfsdk:"admin_delete_enabled"`
}

func NewDataSource() datasource.DataSource { return &organizationDataSource{} }

func (d *organizationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

func (d *organizationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "One organization, looked up by id or slug.",
		Attributes: map[string]schema.Attribute{
			"lookup": schema.StringAttribute{
				Required:    true,
				Description: "The organization id (`org_...`) or its slug.",
			},
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
	}
}

func (d *organizationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := tfutil.ClientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if c != nil {
		d.client = c
	}
}

func (d *organizationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state dataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	o, err := d.client.Organizations.Get(ctx, state.Lookup.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Reading organization", err.Error())
		return
	}
	state.ID = types.StringValue(o.ID)
	state.Name = types.StringValue(o.Name)
	state.Slug = types.StringValue(o.Slug)
	state.MaxAllowedMemberships = types.Int64Value(o.MaxAllowedMemberships)
	state.AdminDeleteEnabled = types.BoolValue(o.AdminDeleteEnabled)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
