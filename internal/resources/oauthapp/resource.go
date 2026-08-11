// Package oauthapp provides the clerk_oauth_application resource: the
// instance as an OAuth identity provider for other applications.
package oauthapp

import (
	"context"
	"fmt"

	"github.com/clerk/clerk-sdk-go/v2"
	sdkoauthapp "github.com/clerk/clerk-sdk-go/v2/oauthapplication"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vanillauys/terraform-provider-clerk/internal/clerkapi"
	"github.com/vanillauys/terraform-provider-clerk/internal/tfutil"
)

var (
	_ resource.Resource                = (*oauthAppResource)(nil)
	_ resource.ResourceWithConfigure   = (*oauthAppResource)(nil)
	_ resource.ResourceWithImportState = (*oauthAppResource)(nil)
)

type oauthAppResource struct {
	client *clerkapi.Client
}

type resourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	CallbackURL          types.String `tfsdk:"callback_url"`
	Scopes               types.String `tfsdk:"scopes"`
	Public               types.Bool   `tfsdk:"public"`
	ConsentScreenEnabled types.Bool   `tfsdk:"consent_screen_enabled"`
	ClientID             types.String `tfsdk:"client_id"`
	ClientSecret         types.String `tfsdk:"client_secret"`
	AuthorizeURL         types.String `tfsdk:"authorize_url"`
	TokenFetchURL        types.String `tfsdk:"token_fetch_url"`
	UserInfoURL          types.String `tfsdk:"user_info_url"`
	DiscoveryURL         types.String `tfsdk:"discovery_url"`
}

func NewResource() resource.Resource { return &oauthAppResource{} }

func (r *oauthAppResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oauth_application"
}

func (r *oauthAppResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An OAuth application: the instance acts as an OAuth identity provider " +
			"for another application.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "OAuth application id.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Application name, shown on the consent screen.",
			},
			"callback_url": schema.StringAttribute{
				Required:    true,
				Description: "Redirect URI of the application.",
			},
			"scopes": schema.StringAttribute{
				Optional: true,
				Description: "Space-separated scopes, for example `\"profile email\"`. Clerk " +
					"normalizes the value server-side (order, plus a forced `offline_access`); " +
					"the provider keeps the configured value in state and cannot detect " +
					"server-side drift on this field.",
			},
			"public": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "`true` for a public client (PKCE, no secret). Create-only: a change forces a replacement.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"consent_screen_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Show the consent screen in the authorization flow.",
			},
			"client_id": schema.StringAttribute{
				Computed:    true,
				Description: "OAuth client id.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"client_secret": schema.StringAttribute{
				Computed:  true,
				Sensitive: true,
				Description: "OAuth client secret. Clerk returns it once, on create; the provider " +
					"keeps that value in state. Null for a public client. Replace the resource to rotate it.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"authorize_url": schema.StringAttribute{
				Computed:    true,
				Description: "Authorization endpoint.",
			},
			"token_fetch_url": schema.StringAttribute{
				Computed:    true,
				Description: "Token endpoint.",
			},
			"user_info_url": schema.StringAttribute{
				Computed:    true,
				Description: "User info endpoint.",
			},
			"discovery_url": schema.StringAttribute{
				Computed:    true,
				Description: "OIDC discovery endpoint.",
			},
		},
	}
}

func (r *oauthAppResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := tfutil.ClientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if c != nil {
		r.client = c
	}
}

// flatten copies the application into the model. Two fields stay as the
// caller set them: client_secret (only the create response carries it) and
// scopes (Clerk canonicalizes the string — order plus a forced
// offline_access — so a read-back can never match the config).
func flatten(a *clerk.OAuthApplication, m *resourceModel) {
	m.ID = types.StringValue(a.ID)
	m.Name = types.StringValue(a.Name)
	m.CallbackURL = types.StringValue(a.CallbackURL)
	m.Public = types.BoolValue(a.Public)
	m.ConsentScreenEnabled = types.BoolValue(a.ConsentScreenEnabled)
	m.ClientID = types.StringValue(a.ClientID)
	m.AuthorizeURL = types.StringValue(a.AuthorizeURL)
	m.TokenFetchURL = types.StringValue(a.TokenFetchURL)
	m.UserInfoURL = types.StringValue(a.UserInfoURL)
	m.DiscoveryURL = types.StringValue(a.DiscoveryURL)
}

func (r *oauthAppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	params := &sdkoauthapp.CreateParams{
		Name:        plan.Name.ValueString(),
		CallbackURL: plan.CallbackURL.ValueString(),
	}
	if !plan.Scopes.IsNull() && !plan.Scopes.IsUnknown() {
		params.Scopes = plan.Scopes.ValueString()
	}
	if !plan.Public.IsNull() && !plan.Public.IsUnknown() {
		params.Public = plan.Public.ValueBool()
	}
	if !plan.ConsentScreenEnabled.IsNull() && !plan.ConsentScreenEnabled.IsUnknown() {
		params.ConsentScreenEnabled = clerk.Bool(plan.ConsentScreenEnabled.ValueBool())
	}
	created, err := r.client.OAuthApps.Create(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Creating OAuth application", err.Error())
		return
	}
	flatten(created, &plan)
	plan.ClientSecret = types.StringPointerValue(created.ClientSecret)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *oauthAppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	a, err := r.client.OAuthApps.Get(ctx, state.ID.ValueString())
	if clerkapi.IsNotFound(err) {
		resp.Diagnostics.AddWarning("OAuth application not found",
			fmt.Sprintf("OAuth application %s no longer exists; removing it from state", state.ID.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Reading OAuth application", err.Error())
		return
	}
	flatten(a, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *oauthAppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	params := &sdkoauthapp.UpdateParams{
		Name:        clerk.String(plan.Name.ValueString()),
		CallbackURL: clerk.String(plan.CallbackURL.ValueString()),
	}
	if !plan.Scopes.IsNull() && !plan.Scopes.IsUnknown() {
		params.Scopes = clerk.String(plan.Scopes.ValueString())
	}
	if !plan.ConsentScreenEnabled.IsNull() && !plan.ConsentScreenEnabled.IsUnknown() {
		params.ConsentScreenEnabled = clerk.Bool(plan.ConsentScreenEnabled.ValueBool())
	}
	updated, err := r.client.OAuthApps.Update(ctx, plan.ID.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Updating OAuth application", err.Error())
		return
	}
	flatten(updated, &plan)
	plan.ClientSecret = state.ClientSecret
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *oauthAppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state resourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.OAuthApps.DeleteOAuthApplication(ctx, state.ID.ValueString())
	if err != nil && !clerkapi.IsNotFound(err) {
		resp.Diagnostics.AddError("Deleting OAuth application", err.Error())
	}
}

func (r *oauthAppResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
