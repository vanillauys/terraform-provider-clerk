// Package orgpermission provides the clerk_organization_permission
// resource: an instance-level custom permission for organization roles.
package orgpermission

import (
	"context"
	"fmt"

	"github.com/clerk/clerk-sdk-go/v2"
	sdkorgpermission "github.com/clerk/clerk-sdk-go/v2/organizationpermission"
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
	_ resource.Resource                = (*permissionResource)(nil)
	_ resource.ResourceWithConfigure   = (*permissionResource)(nil)
	_ resource.ResourceWithImportState = (*permissionResource)(nil)
)

type permissionResource struct {
	client *clerkapi.Client
}

type resourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Key         types.String `tfsdk:"key"`
	Description types.String `tfsdk:"description"`
	Type        types.String `tfsdk:"type"`
}

func NewResource() resource.Resource { return &permissionResource{} }

func (r *permissionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_permission"
}

func (r *permissionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A custom organization permission. Reference its id in the `permissions` " +
			"set of a `clerk_organization_role`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Permission id (`perm_...`).",
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
				Description: "Permission key, for example `org:invoices:read`.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Free-form description.",
			},
			"type": schema.StringAttribute{
				Computed:    true,
				Description: "`system` or `user`. Custom permissions are `user`.",
			},
		},
	}
}

func (r *permissionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := tfutil.ClientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if c != nil {
		r.client = c
	}
}

func flatten(p *clerk.OrganizationPermission, m *resourceModel) {
	m.ID = types.StringValue(p.ID)
	m.Name = types.StringValue(p.Name)
	m.Key = types.StringValue(p.Key)
	m.Description = types.StringPointerValue(p.Description)
	m.Type = types.StringValue(p.Type)
}

func params(m resourceModel) (*string, *string, *string) {
	name := clerk.String(m.Name.ValueString())
	key := clerk.String(m.Key.ValueString())
	var description *string
	if !m.Description.IsNull() && !m.Description.IsUnknown() {
		description = clerk.String(m.Description.ValueString())
	}
	return name, key, description
}

func (r *permissionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	name, key, description := params(plan)
	created, err := r.client.OrgPermissions.Create(ctx, &sdkorgpermission.CreateParams{
		Name: name, Key: key, Description: description,
	})
	if err != nil {
		resp.Diagnostics.AddError("Creating organization permission", err.Error())
		return
	}
	flatten(created, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *permissionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	p, err := r.client.OrgPermissions.Get(ctx, state.ID.ValueString())
	if clerkapi.IsNotFound(err) {
		resp.Diagnostics.AddWarning("Organization permission not found",
			fmt.Sprintf("organization permission %s no longer exists; removing it from state", state.ID.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Reading organization permission", err.Error())
		return
	}
	flatten(p, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *permissionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	name, key, description := params(plan)
	updated, err := r.client.OrgPermissions.Update(ctx, plan.ID.ValueString(), &sdkorgpermission.UpdateParams{
		Name: name, Key: key, Description: description,
	})
	if err != nil {
		resp.Diagnostics.AddError("Updating organization permission", err.Error())
		return
	}
	flatten(updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *permissionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state resourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.OrgPermissions.Delete(ctx, state.ID.ValueString())
	if err != nil && !clerkapi.IsNotFound(err) {
		resp.Diagnostics.AddError("Deleting organization permission", err.Error())
	}
}

func (r *permissionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
