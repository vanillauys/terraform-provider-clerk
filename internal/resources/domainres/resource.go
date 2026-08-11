// Package domainres provides the clerk_domain resource. The Backend API
// creates satellite domains only; the primary domain moves through the
// separate change_domain endpoint, which this provider deliberately does
// not wrap. The domains API has no get-by-id, so Read lists and matches.
package domainres

import (
	"context"
	"fmt"

	"github.com/clerk/clerk-sdk-go/v2"
	sdkdomain "github.com/clerk/clerk-sdk-go/v2/domain"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vanillauys/terraform-provider-clerk/internal/clerkapi"
	"github.com/vanillauys/terraform-provider-clerk/internal/tfutil"
)

var (
	_ resource.Resource                = (*domainResource)(nil)
	_ resource.ResourceWithConfigure   = (*domainResource)(nil)
	_ resource.ResourceWithImportState = (*domainResource)(nil)
)

type domainResource struct {
	client *clerkapi.Client
}

type resourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	ProxyURL          types.String `tfsdk:"proxy_url"`
	IsSatellite       types.Bool   `tfsdk:"is_satellite"`
	FrontendAPIURL    types.String `tfsdk:"frontend_api_url"`
	DevelopmentOrigin types.String `tfsdk:"development_origin"`
}

func NewResource() resource.Resource { return &domainResource{} }

func (r *domainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (r *domainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A satellite domain of the instance. The Backend API creates satellite " +
			"domains only; use the Clerk dashboard for a primary-domain change. Use the " +
			"`clerk_domains` data source for the DNS records.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Domain id.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Domain name, for example `satellite.example.com`.",
			},
			"proxy_url": schema.StringAttribute{
				Optional:    true,
				Description: "Proxy URL for the Frontend API on this domain, for example `https://satellite.example.com/__clerk`.",
			},
			"is_satellite": schema.BoolAttribute{
				Computed:    true,
				Description: "Always `true`: the API creates satellite domains only.",
			},
			"frontend_api_url": schema.StringAttribute{
				Computed:    true,
				Description: "Frontend API URL for this domain.",
			},
			"development_origin": schema.StringAttribute{
				Computed:    true,
				Description: "Origin of the development instance.",
			},
		},
	}
}

func (r *domainResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := tfutil.ClientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if c != nil {
		r.client = c
	}
}

func flatten(d *clerk.Domain, m *resourceModel) {
	m.ID = types.StringValue(d.ID)
	m.Name = types.StringValue(d.Name)
	m.ProxyURL = types.StringPointerValue(d.ProxyURL)
	m.IsSatellite = types.BoolValue(d.IsSatellite)
	m.FrontendAPIURL = types.StringValue(d.FrontendAPIURL)
	m.DevelopmentOrigin = types.StringValue(d.DevelopmentOrigin)
}

func (r *domainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	params := &sdkdomain.CreateParams{
		Name:        clerk.String(plan.Name.ValueString()),
		IsSatellite: clerk.Bool(true),
	}
	if !plan.ProxyURL.IsNull() && !plan.ProxyURL.IsUnknown() {
		params.ProxyURL = clerk.String(plan.ProxyURL.ValueString())
	}
	created, err := r.client.Domains.Create(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Creating domain", err.Error())
		return
	}
	flatten(created, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *domainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	list, err := r.client.Domains.List(ctx, &sdkdomain.ListParams{})
	if err != nil {
		resp.Diagnostics.AddError("Listing domains", err.Error())
		return
	}
	for _, d := range list.Domains {
		if d.ID == state.ID.ValueString() {
			flatten(d, &state)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}
	resp.Diagnostics.AddWarning("Domain not found",
		fmt.Sprintf("domain %s no longer exists; removing it from state", state.ID.ValueString()))
	resp.State.RemoveResource(ctx)
}

func (r *domainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	params := &sdkdomain.UpdateParams{
		Name: clerk.String(plan.Name.ValueString()),
	}
	if !plan.ProxyURL.IsNull() && !plan.ProxyURL.IsUnknown() {
		params.ProxyURL = clerk.String(plan.ProxyURL.ValueString())
	}
	updated, err := r.client.Domains.Update(ctx, plan.ID.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Updating domain", err.Error())
		return
	}
	flatten(updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *domainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state resourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.Domains.Delete(ctx, state.ID.ValueString())
	if err != nil && !clerkapi.IsNotFound(err) {
		resp.Diagnostics.AddError("Deleting domain", err.Error())
	}
}

func (r *domainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
