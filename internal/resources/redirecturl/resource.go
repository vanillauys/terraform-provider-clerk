package redirecturl

import (
	"context"
	"fmt"

	"github.com/clerk/clerk-sdk-go/v2"
	sdkredirecturl "github.com/clerk/clerk-sdk-go/v2/redirecturl"
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
	_ resource.Resource                = (*redirectURLResource)(nil)
	_ resource.ResourceWithConfigure   = (*redirectURLResource)(nil)
	_ resource.ResourceWithImportState = (*redirectURLResource)(nil)
)

type redirectURLResource struct {
	client *clerkapi.Client
}

type resourceModel struct {
	ID  types.String `tfsdk:"id"`
	URL types.String `tfsdk:"url"`
}

func NewResource() resource.Resource { return &redirectURLResource{} }

func (r *redirectURLResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_redirect_url"
}

func (r *redirectURLResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An entry in the allowlist of redirect URLs for OAuth and SSO flows. " +
			"Clerk rejects a sign-in redirect to a URL that is not on this list.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Redirect URL id.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"url": schema.StringAttribute{
				Required:    true,
				Description: "The redirect URL, for example `https://app.example.com/sso-callback`. A change forces a replacement.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *redirectURLResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := tfutil.ClientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if c != nil {
		r.client = c
	}
}

func (r *redirectURLResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := r.client.RedirectURLs.Create(ctx, &sdkredirecturl.CreateParams{
		URL: clerk.String(plan.URL.ValueString()),
	})
	if err != nil {
		resp.Diagnostics.AddError("Creating redirect URL", err.Error())
		return
	}
	plan.ID = types.StringValue(created.ID)
	plan.URL = types.StringValue(created.URL)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *redirectURLResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	u, err := r.client.RedirectURLs.Get(ctx, state.ID.ValueString())
	if clerkapi.IsNotFound(err) {
		resp.Diagnostics.AddWarning("Redirect URL not found",
			fmt.Sprintf("redirect URL %s no longer exists; removing it from state", state.ID.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Reading redirect URL", err.Error())
		return
	}
	state.URL = types.StringValue(u.URL)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is unreachable: every attribute except id forces a replacement.
func (r *redirectURLResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Updating redirect URL", "clerk_redirect_url has no updatable attributes")
}

func (r *redirectURLResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state resourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.RedirectURLs.Delete(ctx, state.ID.ValueString())
	if err != nil && !clerkapi.IsNotFound(err) {
		resp.Diagnostics.AddError("Deleting redirect URL", err.Error())
	}
}

func (r *redirectURLResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
