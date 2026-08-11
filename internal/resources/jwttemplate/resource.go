package jwttemplate

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/clerk/clerk-sdk-go/v2"
	sdkjwttemplate "github.com/clerk/clerk-sdk-go/v2/jwttemplate"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
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
	_ resource.Resource                = (*jwtTemplateResource)(nil)
	_ resource.ResourceWithConfigure   = (*jwtTemplateResource)(nil)
	_ resource.ResourceWithImportState = (*jwtTemplateResource)(nil)
)

type jwtTemplateResource struct {
	client *clerkapi.Client
}

type resourceModel struct {
	ID               types.String         `tfsdk:"id"`
	Name             types.String         `tfsdk:"name"`
	Claims           jsontypes.Normalized `tfsdk:"claims"`
	Lifetime         types.Int64          `tfsdk:"lifetime"`
	AllowedClockSkew types.Int64          `tfsdk:"allowed_clock_skew"`
	CustomSigningKey types.Bool           `tfsdk:"custom_signing_key"`
	SigningAlgorithm types.String         `tfsdk:"signing_algorithm"`
	SigningKey       types.String         `tfsdk:"signing_key"`
}

func NewResource() resource.Resource { return &jwtTemplateResource{} }

func (r *jwtTemplateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_jwt_template"
}

func (r *jwtTemplateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A JWT template. A template defines the shape of the tokens that Clerk mints " +
			"for a session, with custom claims and a custom lifetime.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "JWT template id.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Template name. The name is the key that a frontend SDK uses with `getToken`.",
			},
			"claims": schema.StringAttribute{
				Optional:   true,
				Computed:   true,
				CustomType: jsontypes.NormalizedType{},
				Description: "JSON object with the custom claims. Values support the Clerk shortcodes, " +
					"for example `{{user.public_metadata.role}}`. Defaults to `{}`.",
			},
			"lifetime": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Token lifetime in seconds. The Clerk default is 60.",
			},
			"allowed_clock_skew": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Allowed clock skew in seconds. The Clerk default is 5.",
			},
			"custom_signing_key": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Set to `true` to sign tokens with your own key instead of the instance key.",
			},
			"signing_algorithm": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Signing algorithm, for example `RS256` or `HS256`.",
			},
			"signing_key": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				Description: "Private key for a custom signing key. Clerk never returns this value; " +
					"the provider keeps the configured value in state and cannot detect server-side drift.",
			},
		},
	}
}

func (r *jwtTemplateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := tfutil.ClientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if c != nil {
		r.client = c
	}
}

// flatten copies the server response into the model. It leaves signing_key
// alone: Clerk never returns it, so the configured value stays in state.
func flatten(t *clerk.JWTTemplate, m *resourceModel) {
	m.ID = types.StringValue(t.ID)
	m.Name = types.StringValue(t.Name)
	if len(t.Claims) > 0 {
		m.Claims = jsontypes.NewNormalizedValue(string(t.Claims))
	} else {
		m.Claims = jsontypes.NewNormalizedNull()
	}
	m.Lifetime = types.Int64Value(t.Lifetime)
	m.AllowedClockSkew = types.Int64Value(t.AllowedClockSkew)
	m.CustomSigningKey = types.BoolValue(t.CustomSigningKey)
	m.SigningAlgorithm = types.StringValue(t.SigningAlgorithm)
}

func createParams(m resourceModel) *sdkjwttemplate.CreateParams {
	p := &sdkjwttemplate.CreateParams{
		Name: clerk.String(m.Name.ValueString()),
	}
	if !m.Claims.IsNull() && !m.Claims.IsUnknown() {
		p.Claims = json.RawMessage(m.Claims.ValueString())
	}
	if !m.Lifetime.IsNull() && !m.Lifetime.IsUnknown() {
		p.Lifetime = clerk.Int64(m.Lifetime.ValueInt64())
	}
	if !m.AllowedClockSkew.IsNull() && !m.AllowedClockSkew.IsUnknown() {
		p.AllowedClockSkew = clerk.Int64(m.AllowedClockSkew.ValueInt64())
	}
	if !m.CustomSigningKey.IsNull() && !m.CustomSigningKey.IsUnknown() {
		p.CustomSigningKey = clerk.Bool(m.CustomSigningKey.ValueBool())
	}
	if !m.SigningAlgorithm.IsNull() && !m.SigningAlgorithm.IsUnknown() {
		p.SigningAlgorithm = clerk.String(m.SigningAlgorithm.ValueString())
	}
	if !m.SigningKey.IsNull() && !m.SigningKey.IsUnknown() {
		p.SigningKey = clerk.String(m.SigningKey.ValueString())
	}
	return p
}

func updateParams(m resourceModel) *sdkjwttemplate.UpdateParams {
	c := createParams(m)
	return &sdkjwttemplate.UpdateParams{
		Name:             c.Name,
		Claims:           c.Claims,
		Lifetime:         c.Lifetime,
		AllowedClockSkew: c.AllowedClockSkew,
		CustomSigningKey: c.CustomSigningKey,
		SigningAlgorithm: c.SigningAlgorithm,
		SigningKey:       c.SigningKey,
	}
}

func (r *jwtTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := r.client.JWTTemplates.Create(ctx, createParams(plan))
	if err != nil {
		resp.Diagnostics.AddError("Creating JWT template", err.Error())
		return
	}
	flatten(created, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *jwtTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	t, err := r.client.JWTTemplates.Get(ctx, state.ID.ValueString())
	if clerkapi.IsNotFound(err) {
		resp.Diagnostics.AddWarning("JWT template not found",
			fmt.Sprintf("JWT template %s no longer exists; removing it from state", state.ID.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Reading JWT template", err.Error())
		return
	}
	flatten(t, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *jwtTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	updated, err := r.client.JWTTemplates.Update(ctx, plan.ID.ValueString(), updateParams(plan))
	if err != nil {
		resp.Diagnostics.AddError("Updating JWT template", err.Error())
		return
	}
	flatten(updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *jwtTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state resourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.JWTTemplates.Delete(ctx, state.ID.ValueString())
	if err != nil && !clerkapi.IsNotFound(err) {
		resp.Diagnostics.AddError("Deleting JWT template", err.Error())
	}
}

func (r *jwtTemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
