package identifier

import (
	"context"
	"fmt"

	"github.com/clerk/clerk-sdk-go/v2"
	sdkblocklist "github.com/clerk/clerk-sdk-go/v2/blocklistidentifier"
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
	_ resource.Resource                = (*blocklistResource)(nil)
	_ resource.ResourceWithConfigure   = (*blocklistResource)(nil)
	_ resource.ResourceWithImportState = (*blocklistResource)(nil)
)

type blocklistResource struct {
	client *clerkapi.Client
}

type blocklistModel struct {
	ID             types.String `tfsdk:"id"`
	Identifier     types.String `tfsdk:"identifier"`
	IdentifierType types.String `tfsdk:"identifier_type"`
}

func NewBlocklistResource() resource.Resource { return &blocklistResource{} }

func (r *blocklistResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_blocklist_identifier"
}

func (r *blocklistResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An entry in the sign-up blocklist. Identifiers on this list cannot sign up.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Blocklist identifier id.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"identifier": schema.StringAttribute{
				Required: true,
				Description: "Email address, phone number (with a `+` prefix), or a wildcard domain " +
					"such as `*@spam.example`. A change forces a replacement.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"identifier_type": schema.StringAttribute{
				Computed:    true,
				Description: "Type that Clerk detected: `email_address`, `phone_number`, or `web3_wallet`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *blocklistResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := tfutil.ClientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if c != nil {
		r.client = c
	}
}

func (r *blocklistResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan blocklistModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := r.client.Blocklist.Create(ctx, &sdkblocklist.CreateParams{
		Identifier: clerk.String(plan.Identifier.ValueString()),
	})
	if err != nil {
		resp.Diagnostics.AddError("Creating blocklist identifier", err.Error())
		return
	}
	plan.ID = types.StringValue(created.ID)
	plan.Identifier = types.StringValue(created.Identifier)
	plan.IdentifierType = types.StringValue(created.IdentifierType)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *blocklistResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state blocklistModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	list, err := r.client.Blocklist.List(ctx, &sdkblocklist.ListParams{})
	if err != nil {
		resp.Diagnostics.AddError("Listing blocklist identifiers", err.Error())
		return
	}
	for _, entry := range list.BlocklistIdentifiers {
		if entry.ID == state.ID.ValueString() {
			state.Identifier = types.StringValue(entry.Identifier)
			state.IdentifierType = types.StringValue(entry.IdentifierType)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}
	resp.Diagnostics.AddWarning("Blocklist identifier not found",
		fmt.Sprintf("blocklist identifier %s no longer exists; removing it from state", state.ID.ValueString()))
	resp.State.RemoveResource(ctx)
}

// Update is unreachable: every attribute except id forces a replacement.
func (r *blocklistResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Updating blocklist identifier", "clerk_blocklist_identifier has no updatable attributes")
}

func (r *blocklistResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state blocklistModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.Blocklist.Delete(ctx, state.ID.ValueString())
	if err != nil && !clerkapi.IsNotFound(err) {
		resp.Diagnostics.AddError("Deleting blocklist identifier", err.Error())
	}
}

func (r *blocklistResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
