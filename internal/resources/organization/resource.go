// Package organization provides the clerk_organization resource.
package organization

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/clerk/clerk-sdk-go/v2"
	sdkorganization "github.com/clerk/clerk-sdk-go/v2/organization"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
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
	_ resource.Resource                = (*organizationResource)(nil)
	_ resource.ResourceWithConfigure   = (*organizationResource)(nil)
	_ resource.ResourceWithImportState = (*organizationResource)(nil)
)

type organizationResource struct {
	client *clerkapi.Client
}

type resourceModel struct {
	ID                    types.String         `tfsdk:"id"`
	Name                  types.String         `tfsdk:"name"`
	Slug                  types.String         `tfsdk:"slug"`
	MaxAllowedMemberships types.Int64          `tfsdk:"max_allowed_memberships"`
	AdminDeleteEnabled    types.Bool           `tfsdk:"admin_delete_enabled"`
	CreatedBy             types.String         `tfsdk:"created_by"`
	PublicMetadata        jsontypes.Normalized `tfsdk:"public_metadata"`
	PrivateMetadata       jsontypes.Normalized `tfsdk:"private_metadata"`
}

func NewResource() resource.Resource { return &organizationResource{} }

func (r *organizationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

func (r *organizationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An organization of the instance. The organizations feature must be on " +
			"(see `clerk_instance_organization_settings`).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Organization id (`org_...`).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Organization name.",
			},
			"slug": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "URL-safe slug. Clerk derives one from the name when unset.",
			},
			"max_allowed_memberships": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Maximum members. `0` means unlimited.",
			},
			"admin_delete_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Allow organization admins to delete the organization.",
			},
			"created_by": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "User id of the creator. Create-only: a change forces a replacement. " +
					"When set, Clerk adds this user as the first admin member.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"public_metadata": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				CustomType:  jsontypes.NormalizedType{},
				Description: "JSON object, visible to the frontend.",
			},
			"private_metadata": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				CustomType:  jsontypes.NormalizedType{},
				Description: "JSON object, visible to the backend only.",
			},
		},
	}
}

func (r *organizationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := tfutil.ClientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if c != nil {
		r.client = c
	}
}

func metadataValue(raw json.RawMessage) jsontypes.Normalized {
	if len(raw) == 0 || string(raw) == "null" {
		return jsontypes.NewNormalizedValue("{}")
	}
	return jsontypes.NewNormalizedValue(string(raw))
}

func flatten(o *clerk.Organization, m *resourceModel) {
	m.ID = types.StringValue(o.ID)
	m.Name = types.StringValue(o.Name)
	m.Slug = types.StringValue(o.Slug)
	m.MaxAllowedMemberships = types.Int64Value(o.MaxAllowedMemberships)
	m.AdminDeleteEnabled = types.BoolValue(o.AdminDeleteEnabled)
	if o.CreatedBy == "" {
		m.CreatedBy = types.StringNull()
	} else {
		m.CreatedBy = types.StringValue(o.CreatedBy)
	}
	m.PublicMetadata = metadataValue(o.PublicMetadata)
	m.PrivateMetadata = metadataValue(o.PrivateMetadata)
}

func rawMessage(v jsontypes.Normalized) *json.RawMessage {
	raw := json.RawMessage(v.ValueString())
	return &raw
}

func updateParams(m resourceModel) *sdkorganization.UpdateParams {
	p := &sdkorganization.UpdateParams{
		Name: clerk.String(m.Name.ValueString()),
	}
	if !m.Slug.IsNull() && !m.Slug.IsUnknown() {
		p.Slug = clerk.String(m.Slug.ValueString())
	}
	if !m.MaxAllowedMemberships.IsNull() && !m.MaxAllowedMemberships.IsUnknown() {
		p.MaxAllowedMemberships = clerk.Int64(m.MaxAllowedMemberships.ValueInt64())
	}
	if !m.AdminDeleteEnabled.IsNull() && !m.AdminDeleteEnabled.IsUnknown() {
		p.AdminDeleteEnabled = clerk.Bool(m.AdminDeleteEnabled.ValueBool())
	}
	return p
}

// metadataParams builds the replace-metadata request for the configured
// metadata fields, or returns nil when the config manages neither.
// Terraform state is declarative, so a replace (not a merge) is correct.
func metadataParams(m resourceModel) *sdkorganization.ReplaceMetadataParams {
	p := &sdkorganization.ReplaceMetadataParams{}
	if !m.PublicMetadata.IsNull() && !m.PublicMetadata.IsUnknown() {
		p.PublicMetadata = rawMessage(m.PublicMetadata)
	}
	if !m.PrivateMetadata.IsNull() && !m.PrivateMetadata.IsUnknown() {
		p.PrivateMetadata = rawMessage(m.PrivateMetadata)
	}
	if p.PublicMetadata == nil && p.PrivateMetadata == nil {
		return nil
	}
	return p
}

func (r *organizationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	params := &sdkorganization.CreateParams{
		Name: clerk.String(plan.Name.ValueString()),
	}
	if !plan.Slug.IsNull() && !plan.Slug.IsUnknown() {
		params.Slug = clerk.String(plan.Slug.ValueString())
	}
	if !plan.MaxAllowedMemberships.IsNull() && !plan.MaxAllowedMemberships.IsUnknown() {
		params.MaxAllowedMemberships = clerk.Int64(plan.MaxAllowedMemberships.ValueInt64())
	}
	if !plan.CreatedBy.IsNull() && !plan.CreatedBy.IsUnknown() {
		params.CreatedBy = clerk.String(plan.CreatedBy.ValueString())
	}
	if !plan.PublicMetadata.IsNull() && !plan.PublicMetadata.IsUnknown() {
		params.PublicMetadata = rawMessage(plan.PublicMetadata)
	}
	if !plan.PrivateMetadata.IsNull() && !plan.PrivateMetadata.IsUnknown() {
		params.PrivateMetadata = rawMessage(plan.PrivateMetadata)
	}
	created, err := r.client.Organizations.Create(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Creating organization", err.Error())
		return
	}
	// admin_delete_enabled exists only on the update endpoint.
	if !plan.AdminDeleteEnabled.IsNull() && !plan.AdminDeleteEnabled.IsUnknown() {
		created, err = r.client.Organizations.Update(ctx, created.ID, &sdkorganization.UpdateParams{
			AdminDeleteEnabled: clerk.Bool(plan.AdminDeleteEnabled.ValueBool()),
		})
		if err != nil {
			r.recordPartialCreate(ctx, created, &plan, &resp.Diagnostics, resp)
			resp.Diagnostics.AddError("Setting admin_delete_enabled after create", err.Error())
			return
		}
	}
	flatten(created, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// recordPartialCreate keeps the created id in state when the follow-up
// update fails, so the next apply converges instead of leaking the
// organization.
func (r *organizationResource) recordPartialCreate(ctx context.Context, created *clerk.Organization, plan *resourceModel, diags *diag.Diagnostics, resp *resource.CreateResponse) {
	if created == nil || created.ID == "" {
		return
	}
	flatten(created, plan)
	diags.Append(resp.State.Set(ctx, plan)...)
}

func (r *organizationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	o, err := r.client.Organizations.Get(ctx, state.ID.ValueString())
	if clerkapi.IsNotFound(err) {
		resp.Diagnostics.AddWarning("Organization not found",
			fmt.Sprintf("organization %s no longer exists; removing it from state", state.ID.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Reading organization", err.Error())
		return
	}
	flatten(o, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *organizationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	updated, err := r.client.Organizations.Update(ctx, plan.ID.ValueString(), updateParams(plan))
	if err != nil {
		resp.Diagnostics.AddError("Updating organization", err.Error())
		return
	}
	if mp := metadataParams(plan); mp != nil {
		updated, err = r.client.Organizations.ReplaceMetadata(ctx, plan.ID.ValueString(), mp)
		if err != nil {
			resp.Diagnostics.AddError("Replacing organization metadata", err.Error())
			return
		}
	}
	flatten(updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *organizationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state resourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.Organizations.Delete(ctx, state.ID.ValueString())
	if err != nil && !clerkapi.IsNotFound(err) {
		resp.Diagnostics.AddError("Deleting organization", err.Error())
	}
}

func (r *organizationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
