// Package machine provides the clerk_machine resource. A machine is a
// machine-to-machine identity with its own secret key. scoped_machines is
// the set of machines this machine can mint tokens for; Update reconciles
// it through the machine-scopes API.
package machine

import (
	"context"
	"fmt"

	"github.com/clerk/clerk-sdk-go/v2"
	sdkmachine "github.com/clerk/clerk-sdk-go/v2/machine"
	sdkmachinescope "github.com/clerk/clerk-sdk-go/v2/machinescope"
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
	_ resource.Resource                = (*machineResource)(nil)
	_ resource.ResourceWithConfigure   = (*machineResource)(nil)
	_ resource.ResourceWithImportState = (*machineResource)(nil)
)

type machineResource struct {
	client *clerkapi.Client
}

type resourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	DefaultTokenTTL types.Int64  `tfsdk:"default_token_ttl"`
	ScopedMachines  types.Set    `tfsdk:"scoped_machines"`
	SecretKey       types.String `tfsdk:"secret_key"`
}

func NewResource() resource.Resource { return &machineResource{} }

func (r *machineResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_machine"
}

func (r *machineResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A machine-to-machine identity with its own secret key.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Machine id (`mch_...`).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Machine name.",
			},
			"default_token_ttl": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Default lifetime in seconds for tokens that this machine mints.",
			},
			"scoped_machines": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Ids of the machines this machine can mint tokens for.",
			},
			"secret_key": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The machine secret key (`mk_...`). Store it in a secret manager.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *machineResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := tfutil.ClientFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if c != nil {
		r.client = c
	}
}

func flatten(ctx context.Context, m *clerk.MachineWithScopedMachines, model *resourceModel, diags *diag.Diagnostics) {
	model.ID = types.StringValue(m.ID)
	model.Name = types.StringValue(m.Name)
	model.DefaultTokenTTL = types.Int64Value(m.DefaultTokenTTL)
	ids := make([]string, 0, len(m.ScopedMachines))
	for _, sm := range m.ScopedMachines {
		ids = append(ids, sm.ID)
	}
	scoped, d := types.SetValueFrom(ctx, types.StringType, ids)
	diags.Append(d...)
	model.ScopedMachines = scoped
}

func (r *machineResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	params := &sdkmachine.CreateParams{
		Name: plan.Name.ValueString(),
	}
	if !plan.DefaultTokenTTL.IsNull() && !plan.DefaultTokenTTL.IsUnknown() {
		params.DefaultTokenTTL = clerk.Int64(plan.DefaultTokenTTL.ValueInt64())
	}
	if !plan.ScopedMachines.IsNull() && !plan.ScopedMachines.IsUnknown() {
		resp.Diagnostics.Append(plan.ScopedMachines.ElementsAs(ctx, &params.ScopedMachines, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	created, err := r.client.Machines.Create(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError("Creating machine", err.Error())
		return
	}
	flatten(ctx, &created.MachineWithScopedMachines, &plan, &resp.Diagnostics)
	plan.SecretKey = types.StringValue(created.SecretKey)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *machineResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	m, err := r.client.Machines.Get(ctx, state.ID.ValueString())
	if clerkapi.IsNotFound(err) {
		resp.Diagnostics.AddWarning("Machine not found",
			fmt.Sprintf("machine %s no longer exists; removing it from state", state.ID.ValueString()))
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Reading machine", err.Error())
		return
	}
	flatten(ctx, m, &state, &resp.Diagnostics)
	secret, err := r.client.Machines.GetSecretKey(ctx, m.ID)
	if err != nil {
		resp.Diagnostics.AddError("Reading machine secret key", err.Error())
		return
	}
	state.SecretKey = types.StringValue(secret.Secret)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// reconcileScopes applies the scoped_machines delta between state and plan.
func (r *machineResource) reconcileScopes(ctx context.Context, id string, state, plan resourceModel, diags *diag.Diagnostics) {
	var stateIDs, planIDs []string
	if !state.ScopedMachines.IsNull() {
		diags.Append(state.ScopedMachines.ElementsAs(ctx, &stateIDs, false)...)
	}
	if !plan.ScopedMachines.IsNull() && !plan.ScopedMachines.IsUnknown() {
		diags.Append(plan.ScopedMachines.ElementsAs(ctx, &planIDs, false)...)
	}
	if diags.HasError() {
		return
	}
	inState := make(map[string]bool, len(stateIDs))
	for _, s := range stateIDs {
		inState[s] = true
	}
	inPlan := make(map[string]bool, len(planIDs))
	for _, p := range planIDs {
		inPlan[p] = true
	}
	for _, p := range planIDs {
		if !inState[p] {
			if _, err := r.client.MachineScopes.CreateScope(ctx, id, &sdkmachinescope.CreateScopeParams{ToMachineID: p}); err != nil {
				diags.AddError("Creating machine scope", fmt.Sprintf("scope %s -> %s: %s", id, p, err))
				return
			}
		}
	}
	for _, s := range stateIDs {
		if !inPlan[s] {
			if _, err := r.client.MachineScopes.DeleteScope(ctx, id, s); err != nil && !clerkapi.IsNotFound(err) {
				diags.AddError("Deleting machine scope", fmt.Sprintf("scope %s -> %s: %s", id, s, err))
				return
			}
		}
	}
}

func (r *machineResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state resourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	params := &sdkmachine.UpdateParams{
		Name: clerk.String(plan.Name.ValueString()),
	}
	if !plan.DefaultTokenTTL.IsNull() && !plan.DefaultTokenTTL.IsUnknown() {
		params.DefaultTokenTTL = clerk.Int64(plan.DefaultTokenTTL.ValueInt64())
	}
	if _, err := r.client.Machines.Update(ctx, plan.ID.ValueString(), params); err != nil {
		resp.Diagnostics.AddError("Updating machine", err.Error())
		return
	}
	r.reconcileScopes(ctx, plan.ID.ValueString(), state, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	current, err := r.client.Machines.Get(ctx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Reading machine after update", err.Error())
		return
	}
	flatten(ctx, current, &plan, &resp.Diagnostics)
	plan.SecretKey = state.SecretKey
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *machineResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state resourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.Machines.Delete(ctx, state.ID.ValueString())
	if err != nil && !clerkapi.IsNotFound(err) {
		resp.Diagnostics.AddError("Deleting machine", err.Error())
	}
}

func (r *machineResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
