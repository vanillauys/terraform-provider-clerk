// Package orgrole provides the clerk_organization_role resource. The
// role's permissions update as a whole list on the role endpoints, so the
// separate assign/remove-permission calls stay unused.
package orgrole

import (
	"context"
	"fmt"

	"github.com/clerk/clerk-sdk-go/v2"
	sdkorgrole "github.com/clerk/clerk-sdk-go/v2/organizationrole"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	_ resource.Resource                = (*roleResource)(nil)
	_ resource.ResourceWithConfigure   = (*roleResource)(nil)
	_ resource.ResourceWithImportState = (*roleResource)(nil)
)

type roleResource struct {
	client *clerkapi.Client
}

type resourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Key               types.String `tfsdk:"key"`
	Description       types.String `tfsdk:"description"`
	Permissions       types.Set    `tfsdk:"permissions"`
	IsCreatorEligible types.Bool   `tfsdk:"is_creator_eligible"`
}

func NewResource() resource.Resource { return &roleResource{} }

func (r *roleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_role"
}

func (r *roleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A custom organization role with its permission set.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Role id (`role_...`).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Display name.",
			},
			"key": schema.StringAttribute{
				Required:    true,
				Description: "Role key, for example `org:billing_admin`. Membership resources reference this key.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Free-form description.",
			},
			"permissions": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Ids of the permissions of this role (`perm_...`). The whole set updates in place.",
			},
			"is_creator_eligible": schema.BoolAttribute{
				Computed:    true,
				Description: "`true` when the role can be the organization creator role.",
			},
		},
	}
}

func (r *roleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := tfutil.ClientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if c != nil {
		r.client = c
	}
}

func flatten(ctx context.Context, role *clerk.OrganizationRole, m *resourceModel, diags *diag.Diagnostics) {
	m.ID = types.StringValue(role.ID)
	m.Name = types.StringValue(role.Name)
	m.Key = types.StringValue(role.Key)
	m.Description = types.StringPointerValue(role.Description)
	ids := make([]string, 0, len(role.Permissions))
	for _, p := range role.Permissions {
		ids = append(ids, p.ID)
	}
	permissions, d := types.SetValueFrom(ctx, types.StringType, ids)
	diags.Append(d...)
	m.Permissions = permissions
	m.IsCreatorEligible = types.BoolValue(role.IsCreatorEligible)
}

func permissionIDs(ctx context.Context, m resourceModel, diags *diag.Diagnostics) *[]string {
	ids := []string{}
	if !m.Permissions.IsNull() && !m.Permissions.IsUnknown() {
		diags.Append(m.Permissions.ElementsAs(ctx, &ids, false)...)
	}
	return &ids
}

func (r *roleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	params := &sdkorgrole.CreateParams{
		Name:        clerk.String(plan.Name.ValueString()),
		Key:         clerk.String(plan.Key.ValueString()),
		Permissions: permissionIDs(ctx, plan, &resp.Diagnostics),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		params.Description = clerk.String(plan.Description.ValueString())
	}
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := r.client.OrgRoles.Create(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Creating organization role", err.Error())
		return
	}
	flatten(ctx, created, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *roleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	role, err := r.client.OrgRoles.Get(ctx, state.ID.ValueString())
	if clerkapi.IsNotFound(err) {
		resp.Diagnostics.AddWarning("Organization role not found",
			fmt.Sprintf("organization role %s no longer exists; removing it from state", state.ID.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Reading organization role", err.Error())
		return
	}
	flatten(ctx, role, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *roleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	params := &sdkorgrole.UpdateParams{
		Name:        clerk.String(plan.Name.ValueString()),
		Key:         clerk.String(plan.Key.ValueString()),
		Permissions: permissionIDs(ctx, plan, &resp.Diagnostics),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		params.Description = clerk.String(plan.Description.ValueString())
	}
	if resp.Diagnostics.HasError() {
		return
	}
	updated, err := r.client.OrgRoles.Update(ctx, plan.ID.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Updating organization role", err.Error())
		return
	}
	flatten(ctx, updated, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *roleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state resourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.OrgRoles.Delete(ctx, state.ID.ValueString())
	if err != nil && !clerkapi.IsNotFound(err) {
		resp.Diagnostics.AddError("Deleting organization role", err.Error())
	}
}

func (r *roleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
