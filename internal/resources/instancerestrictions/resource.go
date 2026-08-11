// Package instancerestrictions provides the clerk_instance_restrictions
// singleton. The API has no GET for restrictions, but an empty PATCH (all
// params omitted) returns the current object without a change — Read uses
// that, so this resource has real drift detection.
package instancerestrictions

import (
	"context"

	"github.com/clerk/clerk-sdk-go/v2"
	sdkinstancesettings "github.com/clerk/clerk-sdk-go/v2/instancesettings"
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
	_ resource.Resource                = (*restrictionsResource)(nil)
	_ resource.ResourceWithConfigure   = (*restrictionsResource)(nil)
	_ resource.ResourceWithImportState = (*restrictionsResource)(nil)
)

type restrictionsResource struct {
	client *clerkapi.Client
}

type resourceModel struct {
	ID                          types.String `tfsdk:"id"`
	Allowlist                   types.Bool   `tfsdk:"allowlist"`
	Blocklist                   types.Bool   `tfsdk:"blocklist"`
	BlockEmailSubaddresses      types.Bool   `tfsdk:"block_email_subaddresses"`
	BlockDisposableEmailDomains types.Bool   `tfsdk:"block_disposable_email_domains"`
	IgnoreDotsForGmailAddresses types.Bool   `tfsdk:"ignore_dots_for_gmail_addresses"`
}

func NewResource() resource.Resource { return &restrictionsResource{} }

func (r *restrictionsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_restrictions"
}

func boolAttr(description string) schema.Attribute {
	return schema.BoolAttribute{
		Optional:    true,
		Computed:    true,
		Description: description,
	}
}

func (r *restrictionsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Sign-up restrictions of the instance. This is a singleton: create adopts the " +
			"instance, and destroy only removes the resource from state.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Instance id.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"allowlist":                       boolAttr("Restrict sign-ups to the allowlist (see `clerk_allowlist_identifier`)."),
			"blocklist":                       boolAttr("Block sign-ups from the blocklist (see `clerk_blocklist_identifier`)."),
			"block_email_subaddresses":        boolAttr("Block email addresses with `+`, `=` or `#` subaddresses."),
			"block_disposable_email_domains":  boolAttr("Block sign-ups from disposable email domains."),
			"ignore_dots_for_gmail_addresses": boolAttr("Treat dots in Gmail addresses as insignificant."),
		},
	}
}

func (r *restrictionsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := tfutil.ClientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if c != nil {
		r.client = c
	}
}

func updateParams(m resourceModel) *sdkinstancesettings.UpdateRestrictionsParams {
	p := &sdkinstancesettings.UpdateRestrictionsParams{}
	if !m.Allowlist.IsNull() && !m.Allowlist.IsUnknown() {
		p.Allowlist = clerk.Bool(m.Allowlist.ValueBool())
	}
	if !m.Blocklist.IsNull() && !m.Blocklist.IsUnknown() {
		p.Blocklist = clerk.Bool(m.Blocklist.ValueBool())
	}
	if !m.BlockEmailSubaddresses.IsNull() && !m.BlockEmailSubaddresses.IsUnknown() {
		p.BlockEmailSubaddresses = clerk.Bool(m.BlockEmailSubaddresses.ValueBool())
	}
	if !m.BlockDisposableEmailDomains.IsNull() && !m.BlockDisposableEmailDomains.IsUnknown() {
		p.BlockDisposableEmailDomains = clerk.Bool(m.BlockDisposableEmailDomains.ValueBool())
	}
	if !m.IgnoreDotsForGmailAddresses.IsNull() && !m.IgnoreDotsForGmailAddresses.IsUnknown() {
		p.IgnoreDotsForGmailAddresses = clerk.Bool(m.IgnoreDotsForGmailAddresses.ValueBool())
	}
	return p
}

func flatten(res *clerk.InstanceRestrictions, m *resourceModel) {
	m.Allowlist = types.BoolValue(res.Allowlist)
	m.Blocklist = types.BoolValue(res.Blocklist)
	m.BlockEmailSubaddresses = types.BoolValue(res.BlockEmailSubaddresses)
	m.BlockDisposableEmailDomains = types.BoolValue(res.BlockDisposableEmailDomains)
	m.IgnoreDotsForGmailAddresses = types.BoolValue(res.IgnoreDotsForGmailAddresses)
}

func (r *restrictionsResource) apply(ctx context.Context, plan *resourceModel, resp *resource.CreateResponse) {
	res, err := r.client.InstanceSettings.UpdateRestrictions(ctx, updateParams(*plan))
	if err != nil {
		resp.Diagnostics.AddError("Updating instance restrictions", err.Error())
		return
	}
	flatten(res, plan)
	instance, err := r.client.GetInstance(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Reading instance", err.Error())
		return
	}
	plan.ID = types.StringValue(instance.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *restrictionsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.apply(ctx, &plan, resp)
}

func (r *restrictionsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// An empty PATCH returns the current object without a change.
	res, err := r.client.InstanceSettings.UpdateRestrictions(ctx, &sdkinstancesettings.UpdateRestrictionsParams{})
	if err != nil {
		resp.Diagnostics.AddError("Reading instance restrictions", err.Error())
		return
	}
	flatten(res, &state)
	instance, err := r.client.GetInstance(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Reading instance", err.Error())
		return
	}
	state.ID = types.StringValue(instance.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *restrictionsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	res, err := r.client.InstanceSettings.UpdateRestrictions(ctx, updateParams(plan))
	if err != nil {
		resp.Diagnostics.AddError("Updating instance restrictions", err.Error())
		return
	}
	flatten(res, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *restrictionsResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning("Instance restrictions kept",
		"clerk_instance_restrictions is a singleton: destroy only removes it from state; the configuration stays on the instance")
}

func (r *restrictionsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Any id works; Read replaces it with the real instance id.
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
