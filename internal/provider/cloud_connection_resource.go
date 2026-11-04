// Copyright (c) Traceforce, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	traceforce "github.com/traceforce/traceforce-go-sdk"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &cloudConnectionResource{}
	_ resource.ResourceWithConfigure = &cloudConnectionResource{}
)

// NewCloudConnectionResource registers a customer cloud account (AWS account or
// Microsoft 365 tenant) with Traceforce at the start of terraform apply, and
// disconnects it on destroy.
func NewCloudConnectionResource() resource.Resource {
	return &cloudConnectionResource{}
}

type cloudConnectionResource struct {
	client *traceforce.Client
}

type cloudConnectionResourceModel struct {
	ID            types.String `tfsdk:"id"`
	ProviderName  types.String `tfsdk:"provider_name"`
	Account       types.String `tfsdk:"account"`
	Name          types.String `tfsdk:"name"`
	ModuleVersion types.String `tfsdk:"module_version"`
	ModuleCommit  types.String `tfsdk:"module_commit"`
	Status        types.Int64  `tfsdk:"status"`
}

func (r *cloudConnectionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloud_connection"
}

func (r *cloudConnectionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func (r *cloudConnectionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "A customer cloud account Traceforce collects from. Registered at the start of apply; disconnected on destroy.",
		Attributes: map[string]schema.Attribute{
			// "provider" is reserved in Terraform resource blocks.
			"provider_name": schema.StringAttribute{
				Description:   "Cloud: aws or m365.",
				Required:      true,
				PlanModifiers: replace,
			},
			"account": schema.StringAttribute{
				Description:   "AWS account ID or Microsoft 365 tenant ID.",
				Required:      true,
				PlanModifiers: replace,
			},
			"name": schema.StringAttribute{
				Description: "Name shown on the connection card until the collector's first upload.",
				Optional:    true,
			},
			"module_version": schema.StringAttribute{
				Description: "Collector bundle version, e.g. 1.2.0. Apply fails when it is older than the minimum Traceforce accepts.",
				Required:    true,
			},
			"module_commit": schema.StringAttribute{
				Description: "Git commit of that bundle.",
				Optional:    true,
			},
			// The following attributes are computed.
			"id": schema.StringAttribute{
				Description:   "System generated ID of the connection.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"status": schema.Int64Attribute{
				Description: "1 pending (no upload yet), 2 connected, 3 last upload failed.",
				Computed:    true,
			},
		},
	}
}

// register sends the plan to Traceforce, for create and update alike: the API
// keeps one row per org + cloud + account, so registering again returns the same id.
func (r *cloudConnectionResource) register(plan *cloudConnectionResourceModel) error {
	id, err := r.client.CreateCloudConnection(traceforce.CreateCloudConnectionRequest{
		Provider:      plan.ProviderName.ValueString(),
		Account:       plan.Account.ValueString(),
		Name:          plan.Name.ValueString(),
		ModuleVersion: plan.ModuleVersion.ValueString(),
		ModuleCommit:  plan.ModuleCommit.ValueString(),
	})
	if err != nil {
		return err
	}
	conn, err := r.client.GetCloudConnection(id)
	if err != nil {
		return err
	}
	if conn == nil {
		return fmt.Errorf("cloud connection %s not found right after registering it", id)
	}
	plan.ID = types.StringValue(id)
	plan.Status = types.Int64Value(int64(conn.Status))
	return nil
}

func (r *cloudConnectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan cloudConnectionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// A bundle older than the minimum version fails here (HTTP 426), before
	// anything that depends on this resource is created.
	if err := r.register(&plan); err != nil {
		resp.Diagnostics.AddError("Error registering cloud connection", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *cloudConnectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state cloudConnectionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := r.client.GetCloudConnection(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading cloud connection", err.Error())
		return
	}
	// Disconnected or deleted in Traceforce: the next apply registers it again.
	if conn == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	state.Status = types.Int64Value(int64(conn.Status))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *cloudConnectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan cloudConnectionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.register(&plan); err != nil {
		resp.Diagnostics.AddError("Error updating cloud connection", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete disconnects the connection (terraform destroy). Already gone counts as done.
func (r *cloudConnectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state cloudConnectionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteCloudConnection(state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error disconnecting cloud connection", err.Error())
	}
}
