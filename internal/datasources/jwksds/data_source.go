// Package jwksds provides the clerk_jwks data source: the JSON Web Key Set
// of the instance, for key pinning and JWT validation setups. It exposes
// the key metadata, not the raw key material.
package jwksds

import (
	"context"

	sdkjwks "github.com/clerk/clerk-sdk-go/v2/jwks"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vanillauys/terraform-provider-clerk/internal/clerkapi"
	"github.com/vanillauys/terraform-provider-clerk/internal/tfutil"
)

var (
	_ datasource.DataSource              = (*jwksDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*jwksDataSource)(nil)
)

type jwksDataSource struct {
	client *clerkapi.Client
}

type dataSourceModel struct {
	Keys []keyModel `tfsdk:"keys"`
}

type keyModel struct {
	Kid       types.String `tfsdk:"kid"`
	Algorithm types.String `tfsdk:"algorithm"`
	Use       types.String `tfsdk:"use"`
}

func NewDataSource() datasource.DataSource { return &jwksDataSource{} }

func (d *jwksDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_jwks"
}

func (d *jwksDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The JSON Web Key Set of the instance.",
		Attributes: map[string]schema.Attribute{
			"keys": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Signing keys of the instance.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"kid": schema.StringAttribute{
							Computed:    true,
							Description: "Key id.",
						},
						"algorithm": schema.StringAttribute{
							Computed:    true,
							Description: "Signing algorithm, for example `RS256`.",
						},
						"use": schema.StringAttribute{
							Computed:    true,
							Description: "Key use, normally `sig`.",
						},
					},
				},
			},
		},
	}
}

func (d *jwksDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := tfutil.ClientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if c != nil {
		d.client = c
	}
}

func (d *jwksDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	set, err := d.client.JWKS.Get(ctx, &sdkjwks.GetParams{})
	if err != nil {
		resp.Diagnostics.AddError("Reading JWKS", err.Error())
		return
	}
	state := dataSourceModel{Keys: make([]keyModel, 0, len(set.Keys))}
	for _, k := range set.Keys {
		state.Keys = append(state.Keys, keyModel{
			Kid:       types.StringValue(k.KeyID),
			Algorithm: types.StringValue(k.Algorithm),
			Use:       types.StringValue(k.Use),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
