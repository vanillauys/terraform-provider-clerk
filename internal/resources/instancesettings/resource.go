// Package instancesettings provides the clerk_instance_settings singleton.
//
// PATCH /instance returns an empty body and GET /instance returns only id,
// environment_type, and allowed_origins. Everything else is write-only:
// state is the source of truth and the resource cannot see dashboard drift
// on those fields. allowed_origins is the exception — the resource reads it
// back, but only when the configuration manages it.
package instancesettings

import (
	"context"
	"time"

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
	_ resource.Resource                = (*settingsResource)(nil)
	_ resource.ResourceWithConfigure   = (*settingsResource)(nil)
	_ resource.ResourceWithImportState = (*settingsResource)(nil)
)

type settingsResource struct {
	client *clerkapi.Client
}

type resourceModel struct {
	ID                          types.String `tfsdk:"id"`
	EnvironmentType             types.String `tfsdk:"environment_type"`
	TestMode                    types.Bool   `tfsdk:"test_mode"`
	HIBP                        types.Bool   `tfsdk:"hibp"`
	EnhancedEmailDeliverability types.Bool   `tfsdk:"enhanced_email_deliverability"`
	URLBasedSessionSyncing      types.Bool   `tfsdk:"url_based_session_syncing"`
	SupportEmail                types.String `tfsdk:"support_email"`
	ClerkJSVersion              types.String `tfsdk:"clerk_js_version"`
	DevelopmentOrigin           types.String `tfsdk:"development_origin"`
	AllowedOrigins              types.Set    `tfsdk:"allowed_origins"`
}

func NewResource() resource.Resource { return &settingsResource{} }

func (r *settingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_settings"
}

func (r *settingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "General settings of the instance. This is a singleton: create adopts the " +
			"instance, and destroy only removes the resource from state. Clerk does not return " +
			"most of these fields, so the provider cannot detect dashboard drift on them; " +
			"`allowed_origins` is the exception.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Instance id.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"environment_type": schema.StringAttribute{
				Computed:    true,
				Description: "`development` or `production`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"test_mode": schema.BoolAttribute{
				Optional:    true,
				Description: "Enable test mode. Clerk enables it on development instances by default.",
			},
			"hibp": schema.BoolAttribute{
				Optional:    true,
				Description: "Check passwords against the Have I Been Pwned breach list.",
			},
			"enhanced_email_deliverability": schema.BoolAttribute{
				Optional:    true,
				Description: "Send OTP verification emails through Clerk's shared domain via Postmark (production instances).",
			},
			"url_based_session_syncing": schema.BoolAttribute{
				Optional:    true,
				Description: "Use URL-based session sync instead of third-party cookies on development instances.",
			},
			"support_email": schema.StringAttribute{
				Optional:    true,
				Description: "Support email address shown on the frontend. Set `\"\"` to clear it server-side.",
			},
			"clerk_js_version": schema.StringAttribute{
				Optional:    true,
				Description: "Pinned Clerk JS version for the hosted account pages. Set `\"\"` to clear the pin.",
			},
			"development_origin": schema.StringAttribute{
				Optional:    true,
				Description: "Origin for custom redirects on development instances.",
			},
			"allowed_origins": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Allowed origins for cross-origin requests, for example native app schemes. " +
					"The provider reads this list back and detects drift. Leave it unset to keep it unmanaged.",
			},
		},
	}
}

func (r *settingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := tfutil.ClientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if c != nil {
		r.client = c
	}
}

func updateParams(m resourceModel) *sdkinstancesettings.UpdateParams {
	p := &sdkinstancesettings.UpdateParams{}
	if !m.TestMode.IsNull() && !m.TestMode.IsUnknown() {
		p.TestMode = clerk.Bool(m.TestMode.ValueBool())
	}
	if !m.HIBP.IsNull() && !m.HIBP.IsUnknown() {
		p.HIBP = clerk.Bool(m.HIBP.ValueBool())
	}
	if !m.EnhancedEmailDeliverability.IsNull() && !m.EnhancedEmailDeliverability.IsUnknown() {
		p.EnhancedEmailDeliverability = clerk.Bool(m.EnhancedEmailDeliverability.ValueBool())
	}
	if !m.URLBasedSessionSyncing.IsNull() && !m.URLBasedSessionSyncing.IsUnknown() {
		p.URLBasedSessionSyncing = clerk.Bool(m.URLBasedSessionSyncing.ValueBool())
	}
	if !m.SupportEmail.IsNull() && !m.SupportEmail.IsUnknown() {
		p.SupportEmail = clerk.String(m.SupportEmail.ValueString())
	}
	if !m.ClerkJSVersion.IsNull() && !m.ClerkJSVersion.IsUnknown() {
		p.ClerkJSVersion = clerk.String(m.ClerkJSVersion.ValueString())
	}
	if !m.DevelopmentOrigin.IsNull() && !m.DevelopmentOrigin.IsUnknown() {
		p.DevelopmentOrigin = clerk.String(m.DevelopmentOrigin.ValueString())
	}
	return p
}

// apply pushes the plan to the API and fills the computed fields. Create and
// Update are the same operation on a singleton.
func (r *settingsResource) apply(ctx context.Context, plan *resourceModel, diags *diag.Diagnostics) {
	if err := r.client.InstanceSettings.Update(ctx, updateParams(*plan)); err != nil {
		diags.AddError("Updating instance settings", err.Error())
		return
	}
	var origins []string
	originsManaged := !plan.AllowedOrigins.IsNull() && !plan.AllowedOrigins.IsUnknown()
	if originsManaged {
		diags.Append(plan.AllowedOrigins.ElementsAs(ctx, &origins, false)...)
		if diags.HasError() {
			return
		}
		if err := r.client.UpdateAllowedOrigins(ctx, origins); err != nil {
			diags.AddError("Updating allowed origins", err.Error())
			return
		}
	}
	// GET /instance can serve a stale allowed_origins right after the
	// PATCH; a refresh in that window reports phantom drift. Wait, with a
	// bound, until the read converges on what we just wrote.
	var instance *clerkapi.Instance
	var err error
	for attempt := 0; attempt < 6; attempt++ {
		instance, err = r.client.GetInstance(ctx)
		if err != nil || !originsManaged || sameStringSet(instance.AllowedOrigins, origins) {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if err != nil {
		diags.AddError("Reading instance", err.Error())
		return
	}
	plan.ID = types.StringValue(instance.ID)
	plan.EnvironmentType = types.StringValue(instance.EnvironmentType)
}

func sameStringSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	set := make(map[string]int, len(a))
	for _, s := range a {
		set[s]++
	}
	for _, s := range b {
		set[s]--
		if set[s] < 0 {
			return false
		}
	}
	return true
}

func (r *settingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
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

func (r *settingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	instance, err := r.client.GetInstance(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Reading instance", err.Error())
		return
	}
	state.ID = types.StringValue(instance.ID)
	state.EnvironmentType = types.StringValue(instance.EnvironmentType)
	// Refresh allowed_origins only when the configuration manages it. An
	// unset attribute stays unmanaged and must not create a diff.
	if !state.AllowedOrigins.IsNull() {
		origins, diags := types.SetValueFrom(ctx, types.StringType, instance.AllowedOrigins)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.AllowedOrigins = origins
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *settingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
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

func (r *settingsResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning("Instance settings kept",
		"clerk_instance_settings is a singleton: destroy only removes it from state; the configuration stays on the instance")
}

func (r *settingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Any id works; Read replaces it with the real instance id. The
	// write-only fields stay null after an import, so the first plan shows
	// the configured values as changes; the first apply converges.
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
