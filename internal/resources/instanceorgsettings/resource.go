// Package instanceorgsettings provides the
// clerk_instance_organization_settings singleton. Like the restrictions
// resource, Read uses an empty PATCH: it returns the current object without
// a change, so the resource has real drift detection.
//
// The API accepts role IDs (creator_role_id, domains_default_role_id) but
// returns role NAMES. The id inputs are therefore write-only, and the names
// surface as the computed creator_role and domains_default_role.
package instanceorgsettings

import (
	"context"

	"github.com/clerk/clerk-sdk-go/v2"
	sdkinstancesettings "github.com/clerk/clerk-sdk-go/v2/instancesettings"
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
	_ resource.Resource                = (*orgSettingsResource)(nil)
	_ resource.ResourceWithConfigure   = (*orgSettingsResource)(nil)
	_ resource.ResourceWithImportState = (*orgSettingsResource)(nil)
)

type orgSettingsResource struct {
	client *clerkapi.Client
}

type resourceModel struct {
	ID                     types.String `tfsdk:"id"`
	Enabled                types.Bool   `tfsdk:"enabled"`
	MaxAllowedMemberships  types.Int64  `tfsdk:"max_allowed_memberships"`
	AdminDeleteEnabled     types.Bool   `tfsdk:"admin_delete_enabled"`
	DomainsEnabled         types.Bool   `tfsdk:"domains_enabled"`
	DomainsEnrollmentModes types.List   `tfsdk:"domains_enrollment_modes"`
	CreatorRoleID          types.String `tfsdk:"creator_role_id"`
	DomainsDefaultRoleID   types.String `tfsdk:"domains_default_role_id"`
	CreatorRole            types.String `tfsdk:"creator_role"`
	DomainsDefaultRole     types.String `tfsdk:"domains_default_role"`
	MaxAllowedRoles        types.Int64  `tfsdk:"max_allowed_roles"`
	MaxAllowedPermissions  types.Int64  `tfsdk:"max_allowed_permissions"`
}

func NewResource() resource.Resource { return &orgSettingsResource{} }

func (r *orgSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_organization_settings"
}

func (r *orgSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Organization settings of the instance. This is a singleton: create adopts " +
			"the instance, and destroy only removes the resource from state.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Instance id.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Enable the organizations feature.",
			},
			"max_allowed_memberships": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Maximum members per organization. `0` means unlimited.",
			},
			"admin_delete_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Allow organization admins to delete their organization.",
			},
			"domains_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Enable verified domains for organizations.",
			},
			"domains_enrollment_modes": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Enrollment modes for verified domains: `manual_invitation` and/or `automatic_invitation`, `automatic_suggestion`.",
			},
			"creator_role_id": schema.StringAttribute{
				Optional: true,
				Description: "Role id for the organization creator. Clerk returns the role name, " +
					"not the id, so this input is write-only; the name surfaces in `creator_role`.",
			},
			"domains_default_role_id": schema.StringAttribute{
				Optional: true,
				Description: "Default role id for domain enrollment. Write-only, like `creator_role_id`; " +
					"the name surfaces in `domains_default_role`.",
			},
			"creator_role": schema.StringAttribute{
				Computed:    true,
				Description: "Role name (key) of the organization creator role.",
			},
			"domains_default_role": schema.StringAttribute{
				Computed:    true,
				Description: "Role name (key) of the domain enrollment default role.",
			},
			"max_allowed_roles": schema.Int64Attribute{
				Computed:    true,
				Description: "Maximum custom roles per instance.",
			},
			"max_allowed_permissions": schema.Int64Attribute{
				Computed:    true,
				Description: "Maximum custom permissions per instance.",
			},
		},
	}
}

func (r *orgSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := tfutil.ClientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if c != nil {
		r.client = c
	}
}

func updateParams(ctx context.Context, m resourceModel, diags *diag.Diagnostics) *sdkinstancesettings.UpdateOrganizationSettingsParams {
	p := &sdkinstancesettings.UpdateOrganizationSettingsParams{}
	if !m.Enabled.IsNull() && !m.Enabled.IsUnknown() {
		p.Enabled = clerk.Bool(m.Enabled.ValueBool())
	}
	if !m.MaxAllowedMemberships.IsNull() && !m.MaxAllowedMemberships.IsUnknown() {
		p.MaxAllowedMemberships = clerk.Int64(m.MaxAllowedMemberships.ValueInt64())
	}
	if !m.AdminDeleteEnabled.IsNull() && !m.AdminDeleteEnabled.IsUnknown() {
		p.AdminDeleteEnabled = clerk.Bool(m.AdminDeleteEnabled.ValueBool())
	}
	if !m.DomainsEnabled.IsNull() && !m.DomainsEnabled.IsUnknown() {
		p.DomainsEnabled = clerk.Bool(m.DomainsEnabled.ValueBool())
	}
	if !m.DomainsEnrollmentModes.IsNull() && !m.DomainsEnrollmentModes.IsUnknown() {
		var modes []string
		diags.Append(m.DomainsEnrollmentModes.ElementsAs(ctx, &modes, false)...)
		p.DomainsEnrollmentModes = &modes
	}
	if !m.CreatorRoleID.IsNull() && !m.CreatorRoleID.IsUnknown() {
		p.CreatorRoleID = clerk.String(m.CreatorRoleID.ValueString())
	}
	if !m.DomainsDefaultRoleID.IsNull() && !m.DomainsDefaultRoleID.IsUnknown() {
		p.DomainsDefaultRoleID = clerk.String(m.DomainsDefaultRoleID.ValueString())
	}
	return p
}

func flatten(ctx context.Context, res *clerk.OrganizationSettings, m *resourceModel, diags *diag.Diagnostics) {
	m.Enabled = types.BoolValue(res.Enabled)
	m.MaxAllowedMemberships = types.Int64Value(res.MaxAllowedMemberships)
	m.AdminDeleteEnabled = types.BoolValue(res.AdminDeleteEnabled)
	m.DomainsEnabled = types.BoolValue(res.DomainsEnabled)
	modes, d := types.ListValueFrom(ctx, types.StringType, res.DomainsEnrollmentModes)
	diags.Append(d...)
	m.DomainsEnrollmentModes = modes
	m.CreatorRole = types.StringValue(res.CreatorRole)
	m.DomainsDefaultRole = types.StringValue(res.DomainsDefaultRole)
	m.MaxAllowedRoles = types.Int64Value(res.MaxAllowedRoles)
	m.MaxAllowedPermissions = types.Int64Value(res.MaxAllowedPermissions)
}

func (r *orgSettingsResource) apply(ctx context.Context, plan *resourceModel, diags *diag.Diagnostics) {
	params := updateParams(ctx, *plan, diags)
	if diags.HasError() {
		return
	}
	res, err := r.client.InstanceSettings.UpdateOrganizationSettings(ctx, params)
	if err != nil {
		diags.AddError("Updating instance organization settings", err.Error())
		return
	}
	flatten(ctx, res, plan, diags)
	instance, err := r.client.GetInstance(ctx)
	if err != nil {
		diags.AddError("Reading instance", err.Error())
		return
	}
	plan.ID = types.StringValue(instance.ID)
}

func (r *orgSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.apply(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *orgSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// An empty PATCH returns the current object without a change.
	res, err := r.client.InstanceSettings.UpdateOrganizationSettings(ctx, &sdkinstancesettings.UpdateOrganizationSettingsParams{})
	if err != nil {
		resp.Diagnostics.AddError("Reading instance organization settings", err.Error())
		return
	}
	flatten(ctx, res, &state, &resp.Diagnostics)
	instance, err := r.client.GetInstance(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Reading instance", err.Error())
		return
	}
	state.ID = types.StringValue(instance.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *orgSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.apply(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *orgSettingsResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning("Instance organization settings kept",
		"clerk_instance_organization_settings is a singleton: destroy only removes it from state; the configuration stays on the instance")
}

func (r *orgSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Any id works; Read replaces it with the real instance id.
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
