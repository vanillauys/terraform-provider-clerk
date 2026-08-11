// Package orgdomain provides the clerk_organization_domain resource: a
// verified domain of one organization. The instance organization settings
// must have domains_enabled = true, or the API returns 403.
package orgdomain

import (
	"context"
	"fmt"
	"strings"

	"github.com/clerk/clerk-sdk-go/v2"
	sdkorgdomain "github.com/clerk/clerk-sdk-go/v2/organizationdomain"
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
	_ resource.Resource                = (*orgDomainResource)(nil)
	_ resource.ResourceWithConfigure   = (*orgDomainResource)(nil)
	_ resource.ResourceWithImportState = (*orgDomainResource)(nil)
)

type orgDomainResource struct {
	client *clerkapi.Client
}

type resourceModel struct {
	ID                 types.String `tfsdk:"id"`
	OrganizationID     types.String `tfsdk:"organization_id"`
	Name               types.String `tfsdk:"name"`
	EnrollmentMode     types.String `tfsdk:"enrollment_mode"`
	Verified           types.Bool   `tfsdk:"verified"`
	VerificationStatus types.String `tfsdk:"verification_status"`
}

func NewResource() resource.Resource { return &orgDomainResource{} }

func (r *orgDomainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_domain"
}

func (r *orgDomainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A verified domain of an organization, for enrollment by email domain. " +
			"Needs `domains_enabled = true` on `clerk_instance_organization_settings`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Organization domain id (`orgdmn_...`).",
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
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Domain name, for example `example.com`. A change forces a replacement.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enrollment_mode": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "`manual_invitation`, `automatic_invitation`, or `automatic_suggestion`.",
			},
			"verified": schema.BoolAttribute{
				Optional: true,
				Description: "Mark the domain verified without an email check. Clerk reports the " +
					"result in `verification_status`, not in this field; the input is write-only.",
			},
			"verification_status": schema.StringAttribute{
				Computed:    true,
				Description: "Verification status that Clerk reports, for example `verified` or `unverified`.",
			},
		},
	}
}

func (r *orgDomainResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := tfutil.ClientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if c != nil {
		r.client = c
	}
}

// flatten copies the server object into the model. verified stays as the
// caller set it: the API reports the verification object instead.
func flatten(d *clerk.OrganizationDomain, m *resourceModel) {
	m.ID = types.StringValue(d.ID)
	m.OrganizationID = types.StringValue(d.OrganizationID)
	m.Name = types.StringValue(d.Name)
	m.EnrollmentMode = types.StringValue(d.EnrollmentMode)
	if d.Verification != nil {
		m.VerificationStatus = types.StringValue(d.Verification.Status)
	} else {
		m.VerificationStatus = types.StringValue("unverified")
	}
}

func (r *orgDomainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	params := &sdkorgdomain.CreateParams{
		Name: clerk.String(plan.Name.ValueString()),
	}
	if !plan.EnrollmentMode.IsNull() && !plan.EnrollmentMode.IsUnknown() {
		params.EnrollmentMode = clerk.String(plan.EnrollmentMode.ValueString())
	}
	if !plan.Verified.IsNull() && !plan.Verified.IsUnknown() {
		params.Verified = clerk.Bool(plan.Verified.ValueBool())
	}
	created, err := r.client.OrgDomains.Create(ctx, plan.OrganizationID.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Creating organization domain", err.Error())
		return
	}
	flatten(created, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *orgDomainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	list, err := r.client.OrgDomains.List(ctx, state.OrganizationID.ValueString(), &sdkorgdomain.ListParams{})
	if clerkapi.IsNotFound(err) {
		// The whole organization is gone.
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Listing organization domains", err.Error())
		return
	}
	for _, d := range list.OrganizationDomains {
		if d.ID == state.ID.ValueString() {
			flatten(d, &state)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}
	resp.Diagnostics.AddWarning("Organization domain not found",
		fmt.Sprintf("organization domain %s no longer exists; removing it from state", state.ID.ValueString()))
	resp.State.RemoveResource(ctx)
}

func (r *orgDomainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	params := &sdkorgdomain.UpdateParams{
		OrganizationID: plan.OrganizationID.ValueString(),
		DomainID:       plan.ID.ValueString(),
	}
	if !plan.EnrollmentMode.IsNull() && !plan.EnrollmentMode.IsUnknown() {
		params.EnrollmentMode = clerk.String(plan.EnrollmentMode.ValueString())
	}
	if !plan.Verified.IsNull() && !plan.Verified.IsUnknown() {
		params.Verified = clerk.Bool(plan.Verified.ValueBool())
	}
	updated, err := r.client.OrgDomains.Update(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Updating organization domain", err.Error())
		return
	}
	flatten(updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *orgDomainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state resourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.OrgDomains.Delete(ctx, &sdkorgdomain.DeleteParams{
		OrganizationID: state.OrganizationID.ValueString(),
		DomainID:       state.ID.ValueString(),
	})
	if err != nil && !clerkapi.IsNotFound(err) {
		resp.Diagnostics.AddError("Deleting organization domain", err.Error())
	}
}

// ImportState takes the composite id "org_.../orgdmn_...".
func (r *orgDomainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid import id",
			fmt.Sprintf("expected \"organization_id/domain_id\", got %q", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
