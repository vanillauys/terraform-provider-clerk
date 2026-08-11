// Package orgmembership provides the clerk_organization_membership
// resource: one user's membership in one organization. The API keys the
// membership on organization + user, so both force a replacement; only the
// role updates in place.
package orgmembership

import (
	"context"
	"fmt"
	"strings"

	"github.com/clerk/clerk-sdk-go/v2"
	sdkorgmembership "github.com/clerk/clerk-sdk-go/v2/organizationmembership"
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
	_ resource.Resource                = (*membershipResource)(nil)
	_ resource.ResourceWithConfigure   = (*membershipResource)(nil)
	_ resource.ResourceWithImportState = (*membershipResource)(nil)
)

type membershipResource struct {
	client *clerkapi.Client
}

type resourceModel struct {
	ID             types.String `tfsdk:"id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	UserID         types.String `tfsdk:"user_id"`
	Role           types.String `tfsdk:"role"`
	RoleName       types.String `tfsdk:"role_name"`
}

func NewResource() resource.Resource { return &membershipResource{} }

func (r *membershipResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_membership"
}

func (r *membershipResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "One user's membership in one organization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Membership id (`orgmem_...`).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				Required:    true,
				Description: "Organization id. A change forces a replacement.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"user_id": schema.StringAttribute{
				Required:    true,
				Description: "User id. A change forces a replacement.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role": schema.StringAttribute{
				Required:    true,
				Description: "Role key of the member, for example `org:admin` or a custom `clerk_organization_role` key.",
			},
			"role_name": schema.StringAttribute{
				Computed:    true,
				Description: "Display name of the role.",
			},
		},
	}
}

func (r *membershipResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := tfutil.ClientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if c != nil {
		r.client = c
	}
}

func flatten(m *clerk.OrganizationMembership, model *resourceModel) {
	model.ID = types.StringValue(m.ID)
	if m.Organization != nil {
		model.OrganizationID = types.StringValue(m.Organization.ID)
	}
	if m.PublicUserData != nil {
		model.UserID = types.StringValue(m.PublicUserData.UserID)
	}
	model.Role = types.StringValue(m.Role)
	model.RoleName = types.StringValue(m.RoleName)
}

func (r *membershipResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := r.client.OrgMemberships.Create(ctx, &sdkorgmembership.CreateParams{
		OrganizationID: plan.OrganizationID.ValueString(),
		UserID:         clerk.String(plan.UserID.ValueString()),
		Role:           clerk.String(plan.Role.ValueString()),
	})
	if err != nil {
		resp.Diagnostics.AddError("Creating organization membership", err.Error())
		return
	}
	flatten(created, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *membershipResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	list, err := r.client.OrgMemberships.List(ctx, &sdkorgmembership.ListParams{
		OrganizationID: state.OrganizationID.ValueString(),
	})
	if clerkapi.IsNotFound(err) {
		// The whole organization is gone.
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Listing organization memberships", err.Error())
		return
	}
	for _, m := range list.OrganizationMemberships {
		if m.PublicUserData != nil && m.PublicUserData.UserID == state.UserID.ValueString() {
			flatten(m, &state)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}
	resp.Diagnostics.AddWarning("Organization membership not found",
		fmt.Sprintf("user %s is no longer a member of %s; removing the membership from state",
			state.UserID.ValueString(), state.OrganizationID.ValueString()))
	resp.State.RemoveResource(ctx)
}

func (r *membershipResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	updated, err := r.client.OrgMemberships.Update(ctx, &sdkorgmembership.UpdateParams{
		OrganizationID: plan.OrganizationID.ValueString(),
		UserID:         plan.UserID.ValueString(),
		Role:           clerk.String(plan.Role.ValueString()),
	})
	if err != nil {
		resp.Diagnostics.AddError("Updating organization membership", err.Error())
		return
	}
	flatten(updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *membershipResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state resourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.OrgMemberships.Delete(ctx, &sdkorgmembership.DeleteParams{
		OrganizationID: state.OrganizationID.ValueString(),
		UserID:         state.UserID.ValueString(),
	})
	if err != nil && !clerkapi.IsNotFound(err) {
		resp.Diagnostics.AddError("Deleting organization membership", err.Error())
	}
}

// ImportState takes the composite id "org_.../user_...".
func (r *membershipResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid import id",
			fmt.Sprintf("expected \"organization_id/user_id\", got %q", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_id"), parts[1])...)
	// Read matches on organization + user; id refreshes there.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
