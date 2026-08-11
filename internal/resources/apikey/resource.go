// Package apikey provides the clerk_api_key resource. An API key belongs
// to a subject (a user, an organization, or a machine). The secret comes
// from the create response and stays retrievable through the secret
// endpoint, so Read keeps it fresh and import is complete. A revoked or
// expired key is unusable: Read drops it from state so the next plan
// recreates it.
package apikey

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/clerk/clerk-sdk-go/v2"
	sdkapikey "github.com/clerk/clerk-sdk-go/v2/apikey"
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
	_ resource.Resource                = (*apiKeyResource)(nil)
	_ resource.ResourceWithConfigure   = (*apiKeyResource)(nil)
	_ resource.ResourceWithImportState = (*apiKeyResource)(nil)
)

type apiKeyResource struct {
	client *clerkapi.Client
}

type resourceModel struct {
	ID                     types.String         `tfsdk:"id"`
	Name                   types.String         `tfsdk:"name"`
	Subject                types.String         `tfsdk:"subject"`
	Description            types.String         `tfsdk:"description"`
	Claims                 jsontypes.Normalized `tfsdk:"claims"`
	Scopes                 types.Set            `tfsdk:"scopes"`
	SecondsUntilExpiration types.Int64          `tfsdk:"seconds_until_expiration"`
	Expiration             types.Int64          `tfsdk:"expiration"`
	Type                   types.String         `tfsdk:"type"`
	Secret                 types.String         `tfsdk:"secret"`
}

func NewResource() resource.Resource { return &apiKeyResource{} }

func (r *apiKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_key"
}

func (r *apiKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An API key for a subject (a user, an organization, or a machine). " +
			"A key that Clerk marks revoked or expired leaves state on the next refresh, " +
			"and the next apply creates a replacement.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "API key id.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Key name. The API cannot change it: a change forces a replacement.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"subject": schema.StringAttribute{
				Required:    true,
				Description: "Id of the owner: a user (`user_...`), an organization (`org_...`), or a machine (`mch_...`).",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Free-form description.",
			},
			"claims": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				CustomType:  jsontypes.NormalizedType{},
				Description: "JSON object with custom claims for the key.",
			},
			"scopes": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Scopes of the key, for example `[\"read\", \"write\"]`.",
			},
			"seconds_until_expiration": schema.Int64Attribute{
				Optional: true,
				Description: "Lifetime of the key in seconds, counted from the moment of the call. " +
					"Clerk does not return this value; the computed `expiration` timestamp does. " +
					"Leave it unset for a key without an expiration.",
			},
			"expiration": schema.Int64Attribute{
				Computed:    true,
				Description: "Expiration as a unix timestamp. Null for a key without an expiration.",
			},
			"type": schema.StringAttribute{
				Computed:    true,
				Description: "Key type, normally `api_key`.",
			},
			"secret": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The key secret (`ak_...`). Store it in a secret manager, not in plain state files that others can read.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *apiKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := tfutil.ClientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if c != nil {
		r.client = c
	}
}

// flatten copies the API key into the model. It does not touch secret and
// seconds_until_expiration: the callers own those.
func flatten(ctx context.Context, k *clerk.APIKey, m *resourceModel, diags *diag.Diagnostics) {
	m.ID = types.StringValue(k.ID)
	m.Name = types.StringValue(k.Name)
	m.Subject = types.StringValue(k.Subject)
	m.Description = types.StringPointerValue(k.Description)
	if len(k.Claims) > 0 && string(k.Claims) != "null" {
		m.Claims = jsontypes.NewNormalizedValue(string(k.Claims))
	} else {
		m.Claims = jsontypes.NewNormalizedNull()
	}
	scopes, d := types.SetValueFrom(ctx, types.StringType, k.Scopes)
	diags.Append(d...)
	m.Scopes = scopes
	m.Expiration = types.Int64PointerValue(k.Expiration)
	m.Type = types.StringValue(k.Type)
}

func (r *apiKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	params := &sdkapikey.CreateParams{
		Name:    clerk.String(plan.Name.ValueString()),
		Subject: clerk.String(plan.Subject.ValueString()),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		params.Description = clerk.String(plan.Description.ValueString())
	}
	if !plan.Claims.IsNull() && !plan.Claims.IsUnknown() {
		params.Claims = json.RawMessage(plan.Claims.ValueString())
	}
	if !plan.Scopes.IsNull() && !plan.Scopes.IsUnknown() {
		resp.Diagnostics.Append(plan.Scopes.ElementsAs(ctx, &params.Scopes, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	if !plan.SecondsUntilExpiration.IsNull() && !plan.SecondsUntilExpiration.IsUnknown() {
		params.SecondsUntilExpiration = clerk.Int64(plan.SecondsUntilExpiration.ValueInt64())
	}
	created, err := r.client.APIKeys.Create(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Creating API key", err.Error())
		return
	}
	flatten(ctx, &created.APIKey, &plan, &resp.Diagnostics)
	plan.Secret = types.StringValue(created.Secret)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *apiKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	k, err := r.client.APIKeys.Get(ctx, state.ID.ValueString())
	if clerkapi.IsNotFound(err) {
		resp.Diagnostics.AddWarning("API key not found",
			fmt.Sprintf("API key %s no longer exists; removing it from state", state.ID.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Reading API key", err.Error())
		return
	}
	if k.Revoked || k.Expired {
		resp.Diagnostics.AddWarning("API key unusable",
			fmt.Sprintf("API key %s is revoked or expired; removing it from state so the next apply replaces it", k.ID))
		resp.State.RemoveResource(ctx)
		return
	}
	flatten(ctx, k, &state, &resp.Diagnostics)
	secret, err := r.client.APIKeys.GetSecret(ctx, k.ID)
	if err != nil {
		resp.Diagnostics.AddError("Reading API key secret", err.Error())
		return
	}
	state.Secret = types.StringValue(secret.Secret)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *apiKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	params := &sdkapikey.UpdateParams{
		Subject: clerk.String(plan.Subject.ValueString()),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		params.Description = clerk.String(plan.Description.ValueString())
	}
	if !plan.Claims.IsNull() && !plan.Claims.IsUnknown() {
		params.Claims = json.RawMessage(plan.Claims.ValueString())
	}
	if !plan.Scopes.IsNull() && !plan.Scopes.IsUnknown() {
		resp.Diagnostics.Append(plan.Scopes.ElementsAs(ctx, &params.Scopes, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	if !plan.SecondsUntilExpiration.IsNull() && !plan.SecondsUntilExpiration.IsUnknown() {
		params.SecondsUntilExpiration = clerk.Int64(plan.SecondsUntilExpiration.ValueInt64())
	}
	updated, err := r.client.APIKeys.Update(ctx, plan.ID.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("Updating API key", err.Error())
		return
	}
	flatten(ctx, updated, &plan, &resp.Diagnostics)
	plan.Secret = state.Secret
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *apiKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state resourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.APIKeys.Delete(ctx, state.ID.ValueString())
	if err != nil && !clerkapi.IsNotFound(err) {
		resp.Diagnostics.AddError("Deleting API key", err.Error())
	}
}

func (r *apiKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
