// Package samlconn provides the clerk_saml_connection resource.
package samlconn

import (
	"context"
	"fmt"

	"github.com/clerk/clerk-sdk-go/v2"
	sdksamlconnection "github.com/clerk/clerk-sdk-go/v2/samlconnection"
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
	_ resource.Resource                = (*samlResource)(nil)
	_ resource.ResourceWithConfigure   = (*samlResource)(nil)
	_ resource.ResourceWithImportState = (*samlResource)(nil)
)

type samlResource struct {
	client *clerkapi.Client
}

type resourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Domain             types.String `tfsdk:"domain"`
	Provider           types.String `tfsdk:"provider_key"`
	OrganizationID     types.String `tfsdk:"organization_id"`
	IdpEntityID        types.String `tfsdk:"idp_entity_id"`
	IdpSsoURL          types.String `tfsdk:"idp_sso_url"`
	IdpCertificate     types.String `tfsdk:"idp_certificate"`
	IdpMetadataURL     types.String `tfsdk:"idp_metadata_url"`
	Active             types.Bool   `tfsdk:"active"`
	SyncUserAttributes types.Bool   `tfsdk:"sync_user_attributes"`
	AllowSubdomains    types.Bool   `tfsdk:"allow_subdomains"`
	AllowIdpInitiated  types.Bool   `tfsdk:"allow_idp_initiated"`
	ForceAuthn         types.Bool   `tfsdk:"force_authn"`
	AcsURL             types.String `tfsdk:"acs_url"`
	SPEntityID         types.String `tfsdk:"sp_entity_id"`
	SPMetadataURL      types.String `tfsdk:"sp_metadata_url"`
}

func NewResource() resource.Resource { return &samlResource{} }

func (r *samlResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_saml_connection"
}

func (r *samlResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A SAML connection for enterprise SSO.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "SAML connection id (`samlc_...`).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Connection name.",
			},
			"domain": schema.StringAttribute{
				Required:    true,
				Description: "Email domain of the users of this connection, for example `example.com`.",
			},
			// The attribute is provider_key, not provider: "provider" is a
			// reserved attribute name in Terraform resource schemas.
			"provider_key": schema.StringAttribute{
				Required:    true,
				Description: "Identity provider key: `saml_custom`, `saml_okta`, `saml_google`, or `saml_microsoft`. A change forces a replacement.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"organization_id": schema.StringAttribute{
				Optional:    true,
				Description: "Restrict the connection to one organization.",
			},
			"idp_entity_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Entity id of the identity provider.",
			},
			"idp_sso_url": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Single-sign-on URL of the identity provider.",
			},
			"idp_certificate": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "X.509 certificate of the identity provider.",
			},
			"idp_metadata_url": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Metadata URL of the identity provider. Clerk fills the idp fields from it.",
			},
			"active": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Enable the connection for sign-ins.",
			},
			"sync_user_attributes": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Update user attributes from the identity provider on each sign-in.",
			},
			"allow_subdomains": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Match subdomains of `domain` too.",
			},
			"allow_idp_initiated": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Allow identity-provider-initiated flows.",
			},
			"force_authn": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Force re-authentication at the identity provider on each sign-in.",
			},
			"acs_url": schema.StringAttribute{
				Computed:    true,
				Description: "Assertion consumer service URL — configure this at the identity provider.",
			},
			"sp_entity_id": schema.StringAttribute{
				Computed:    true,
				Description: "Service provider entity id — configure this at the identity provider.",
			},
			"sp_metadata_url": schema.StringAttribute{
				Computed:    true,
				Description: "Service provider metadata URL.",
			},
		},
	}
}

func (r *samlResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := tfutil.ClientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if c != nil {
		r.client = c
	}
}

func flatten(s *clerk.SAMLConnection, m *resourceModel) {
	m.ID = types.StringValue(s.ID)
	m.Name = types.StringValue(s.Name)
	m.Domain = types.StringValue(s.Domain)
	m.Provider = types.StringValue(s.Provider)
	m.OrganizationID = types.StringPointerValue(s.OrganizationID)
	m.IdpEntityID = types.StringPointerValue(s.IdpEntityID)
	m.IdpSsoURL = types.StringPointerValue(s.IdpSsoURL)
	m.IdpCertificate = types.StringPointerValue(s.IdpCertificate)
	m.IdpMetadataURL = types.StringPointerValue(s.IdpMetadataURL)
	m.Active = types.BoolValue(s.Active)
	m.SyncUserAttributes = types.BoolValue(s.SyncUserAttributes)
	m.AllowSubdomains = types.BoolValue(s.AllowSubdomains)
	m.AllowIdpInitiated = types.BoolValue(s.AllowIdpInitiated)
	m.ForceAuthn = types.BoolValue(s.ForceAuthn)
	m.AcsURL = types.StringValue(s.AcsURL)
	m.SPEntityID = types.StringValue(s.SPEntityID)
	m.SPMetadataURL = types.StringValue(s.SPMetadataURL)
}

func stringParam(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	return clerk.String(v.ValueString())
}

func boolParam(v types.Bool) *bool {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	return clerk.Bool(v.ValueBool())
}

func (r *samlResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := r.client.SAMLConnections.Create(ctx, &sdksamlconnection.CreateParams{
		Name:           clerk.String(plan.Name.ValueString()),
		Domain:         clerk.String(plan.Domain.ValueString()),
		Provider:       clerk.String(plan.Provider.ValueString()),
		OrganizationID: stringParam(plan.OrganizationID),
		IdpEntityID:    stringParam(plan.IdpEntityID),
		IdpSsoURL:      stringParam(plan.IdpSsoURL),
		IdpCertificate: stringParam(plan.IdpCertificate),
		IdpMetadataURL: stringParam(plan.IdpMetadataURL),
		ForceAuthn:     boolParam(plan.ForceAuthn),
	})
	if err != nil {
		resp.Diagnostics.AddError("Creating SAML connection", err.Error())
		return
	}
	// active and the sync switches exist only on the update endpoint.
	updateNeeded := boolParam(plan.Active) != nil || boolParam(plan.SyncUserAttributes) != nil ||
		boolParam(plan.AllowSubdomains) != nil || boolParam(plan.AllowIdpInitiated) != nil
	if updateNeeded {
		created, err = r.client.SAMLConnections.Update(ctx, created.ID, &sdksamlconnection.UpdateParams{
			Active:             boolParam(plan.Active),
			SyncUserAttributes: boolParam(plan.SyncUserAttributes),
			AllowSubdomains:    boolParam(plan.AllowSubdomains),
			AllowIdpInitiated:  boolParam(plan.AllowIdpInitiated),
		})
		if err != nil {
			resp.Diagnostics.AddError("Configuring SAML connection after create",
				fmt.Sprintf("connection %s was created, but the follow-up update failed: %s", created.ID, err))
			return
		}
	}
	flatten(created, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *samlResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	s, err := r.client.SAMLConnections.Get(ctx, state.ID.ValueString())
	if clerkapi.IsNotFound(err) {
		resp.Diagnostics.AddWarning("SAML connection not found",
			fmt.Sprintf("SAML connection %s no longer exists; removing it from state", state.ID.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Reading SAML connection", err.Error())
		return
	}
	flatten(s, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *samlResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	updated, err := r.client.SAMLConnections.Update(ctx, plan.ID.ValueString(), &sdksamlconnection.UpdateParams{
		Name:               clerk.String(plan.Name.ValueString()),
		Domain:             clerk.String(plan.Domain.ValueString()),
		OrganizationID:     stringParam(plan.OrganizationID),
		IdpEntityID:        stringParam(plan.IdpEntityID),
		IdpSsoURL:          stringParam(plan.IdpSsoURL),
		IdpCertificate:     stringParam(plan.IdpCertificate),
		IdpMetadataURL:     stringParam(plan.IdpMetadataURL),
		Active:             boolParam(plan.Active),
		SyncUserAttributes: boolParam(plan.SyncUserAttributes),
		AllowSubdomains:    boolParam(plan.AllowSubdomains),
		AllowIdpInitiated:  boolParam(plan.AllowIdpInitiated),
		ForceAuthn:         boolParam(plan.ForceAuthn),
	})
	if err != nil {
		resp.Diagnostics.AddError("Updating SAML connection", err.Error())
		return
	}
	flatten(updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *samlResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state resourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.SAMLConnections.Delete(ctx, state.ID.ValueString())
	if err != nil && !clerkapi.IsNotFound(err) {
		resp.Diagnostics.AddError("Deleting SAML connection", err.Error())
	}
}

func (r *samlResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
