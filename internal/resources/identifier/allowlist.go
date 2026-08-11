// Package identifier holds the allowlist and blocklist identifier
// resources. The two Clerk APIs are near-identical: create, list, delete,
// with no get-by-id and no update. Read therefore lists all identifiers and
// picks the one with the matching id.
package identifier

import (
	"context"
	"fmt"

	"github.com/clerk/clerk-sdk-go/v2"
	sdkallowlist "github.com/clerk/clerk-sdk-go/v2/allowlistidentifier"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vanillauys/terraform-provider-clerk/internal/clerkapi"
	"github.com/vanillauys/terraform-provider-clerk/internal/tfutil"
)

var (
	_ resource.Resource                = (*allowlistResource)(nil)
	_ resource.ResourceWithConfigure   = (*allowlistResource)(nil)
	_ resource.ResourceWithImportState = (*allowlistResource)(nil)
)

type allowlistResource struct {
	client *clerkapi.Client
}

type allowlistModel struct {
	ID             types.String `tfsdk:"id"`
	Identifier     types.String `tfsdk:"identifier"`
	IdentifierType types.String `tfsdk:"identifier_type"`
	Notify         types.Bool   `tfsdk:"notify"`
}

func NewAllowlistResource() resource.Resource { return &allowlistResource{} }

func (r *allowlistResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_allowlist_identifier"
}

func (r *allowlistResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An entry in the sign-up allowlist. When the allowlist restriction is on, " +
			"only identifiers on this list can sign up.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Allowlist identifier id.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"identifier": schema.StringAttribute{
				Required: true,
				Description: "Email address, phone number (with a `+` prefix), username, or a wildcard " +
					"domain such as `*@example.com`. A change forces a replacement.",
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
			"notify": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
				Description: "Send an invitation email on create. Only valid for an email identifier. " +
					"Clerk never returns this value; a change forces a replacement. Defaults to `false`.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *allowlistResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := tfutil.ClientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if c != nil {
		r.client = c
	}
}

func (r *allowlistResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan allowlistModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := r.client.Allowlist.Create(ctx, &sdkallowlist.CreateParams{
		Identifier: clerk.String(plan.Identifier.ValueString()),
		Notify:     clerk.Bool(plan.Notify.ValueBool()),
	})
	if err != nil {
		resp.Diagnostics.AddError("Creating allowlist identifier", err.Error())
		return
	}
	plan.ID = types.StringValue(created.ID)
	plan.Identifier = types.StringValue(created.Identifier)
	plan.IdentifierType = types.StringValue(created.IdentifierType)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *allowlistResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state allowlistModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	list, err := r.client.Allowlist.List(ctx, &sdkallowlist.ListParams{})
	if err != nil {
		resp.Diagnostics.AddError("Listing allowlist identifiers", err.Error())
		return
	}
	for _, entry := range list.AllowlistIdentifiers {
		if entry.ID == state.ID.ValueString() {
			state.Identifier = types.StringValue(entry.Identifier)
			state.IdentifierType = types.StringValue(entry.IdentifierType)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}
	resp.Diagnostics.AddWarning("Allowlist identifier not found",
		fmt.Sprintf("allowlist identifier %s no longer exists; removing it from state", state.ID.ValueString()))
	resp.State.RemoveResource(ctx)
}

// Update is unreachable: every attribute except id forces a replacement.
func (r *allowlistResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Updating allowlist identifier", "clerk_allowlist_identifier has no updatable attributes")
}

func (r *allowlistResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state allowlistModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.Allowlist.Delete(ctx, state.ID.ValueString())
	if err != nil && !clerkapi.IsNotFound(err) {
		resp.Diagnostics.AddError("Deleting allowlist identifier", err.Error())
	}
}

func (r *allowlistResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	// Clerk does not return notify. Seed the schema default so the first
	// plan after an import is empty.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("notify"), false)...)
}
