// Package webhook provides the clerk_webhook singleton. Create enables the
// Svix integration for the instance and destroy disables it. The API has
// no read endpoint for the Svix state, so state is the source of truth and
// the resource cannot see dashboard drift. There is no import: without a
// read there is nothing to import.
package webhook

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/vanillauys/terraform-provider-clerk/internal/clerkapi"
	"github.com/vanillauys/terraform-provider-clerk/internal/tfutil"
)

var (
	_ resource.Resource              = (*webhookResource)(nil)
	_ resource.ResourceWithConfigure = (*webhookResource)(nil)
)

type webhookResource struct {
	client *clerkapi.Client
}

type resourceModel struct {
	ID      types.String `tfsdk:"id"`
	SvixURL types.String `tfsdk:"svix_url"`
}

func NewResource() resource.Resource { return &webhookResource{} }

func (r *webhookResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook"
}

func (r *webhookResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Svix webhook integration of the instance. This is a singleton: create " +
			"enables Svix, destroy disables it. Clerk has no read endpoint for it, so the " +
			"provider cannot see dashboard drift. Manage the webhook endpoints themselves in " +
			"the Svix dashboard.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Instance id.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"svix_url": schema.StringAttribute{
				Computed:  true,
				Sensitive: true,
				Description: "Short-lived Svix dashboard login URL from the enable call. It embeds " +
					"a login key and expires quickly; generate a fresh one in the Clerk dashboard when needed.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *webhookResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := tfutil.ClientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if c != nil {
		r.client = c
	}
}

func (r *webhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := r.client.SvixWebhooks.Create(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Enabling the Svix webhook integration", err.Error())
		return
	}
	instance, err := r.client.GetInstance(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Reading instance", err.Error())
		return
	}
	plan.ID = types.StringValue(instance.ID)
	plan.SvixURL = types.StringValue(created.SvixURL)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read keeps state as-is: the API has no endpoint for the Svix state.
func (r *webhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is unreachable: both attributes are computed.
func (r *webhookResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Updating webhook", "clerk_webhook has no updatable attributes")
}

func (r *webhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state resourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if _, err := r.client.SvixWebhooks.Delete(ctx); err != nil && !clerkapi.IsNotFound(err) {
		resp.Diagnostics.AddError("Disabling the Svix webhook integration", err.Error())
	}
}
