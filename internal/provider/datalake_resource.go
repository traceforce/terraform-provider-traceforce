// Copyright (c) Traceforce, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	traceforce "github.com/traceforce/traceforce-go-sdk"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &datalakeResource{}
	_ resource.ResourceWithConfigure   = &datalakeResource{}
	_ resource.ResourceWithImportState = &datalakeResource{}
)

// NewDatalakeResource is a helper function to simplify the provider implementation.
func NewDatalakeResource() resource.Resource {
	return &datalakeResource{}
}

// datalakeResource is the resource implementation.
type datalakeResource struct {
	client *traceforce.Client
}

// datalakeResourceModel maps datalakes schema data.
type datalakeResourceModel struct {
	ID        types.String `tfsdk:"id"`
	ProjectId types.String `tfsdk:"project_id"`
	Type      types.String `tfsdk:"type"`
	Name      types.String `tfsdk:"name"`
	Status    types.String `tfsdk:"status"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

// Metadata returns the resource type name.
func (r *datalakeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_datalake"
}

func (r *datalakeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Add a nil check when handling ProviderData because Terraform
	// sets that data after it calls the ConfigureProvider RPC.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*traceforce.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Provider Configuration",
			"An error occurred while configuring the provider. Please contact support if this persists.",
		)

		return
	}

	r.client = client
}

// Schema defines the schema for the resource.
func (r *datalakeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "ID of the project this datalake belongs to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Description: fmt.Sprintf("Type of datalake. Currently supported: %s.",
					traceforce.DatalakeTypeBigQuery),
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the datalake. This value must be unique within a project.",
				Required:    true,
			},
			// The following attributes are computed and should never be reflected in changes.
			"status": schema.StringAttribute{
				Description: fmt.Sprintf("Status of the datalake. Valid values: %s, %s, %s.",
					traceforce.DatalakeStatusPending,
					traceforce.DatalakeStatusWaitingForUserInput,
					traceforce.DatalakeStatusReady),
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"id": schema.StringAttribute{
				Description: "System generated ID of the datalake",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "Date and time the datalake was created",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "Date and time the datalake was last updated",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *datalakeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan datalakeResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := traceforce.CreateDatalakeRequest{
		HostingEnvironmentID: plan.ProjectId.ValueString(),
		Type:                 traceforce.DatalakeType(plan.Type.ValueString()),
		Name:                 plan.Name.ValueString(),
	}

	datalake, err := r.client.CreateDatalake(input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating datalake", err.Error())
		return
	}

	plan = datalakeResourceModel{
		ID:        types.StringValue(datalake.ID),
		ProjectId: types.StringValue(datalake.HostingEnvironmentID),
		Type:      types.StringValue(string(datalake.Type)),
		Name:      types.StringValue(datalake.Name),
		Status:    types.StringValue(string(datalake.Status)),
		CreatedAt: types.StringValue(datalake.CreatedAt.Format(time.RFC3339)),
		UpdatedAt: types.StringValue(datalake.UpdatedAt.Format(time.RFC3339)),
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *datalakeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state datalakeResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	datalake, err := r.client.GetDatalake(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading datalake", err.Error())
		return
	}

	state = datalakeResourceModel{
		ID:        types.StringValue(datalake.ID),
		ProjectId: types.StringValue(datalake.HostingEnvironmentID),
		Type:      types.StringValue(string(datalake.Type)),
		Name:      types.StringValue(datalake.Name),
		Status:    types.StringValue(string(datalake.Status)),
		CreatedAt: types.StringValue(datalake.CreatedAt.Format(time.RFC3339)),
		UpdatedAt: types.StringValue(datalake.UpdatedAt.Format(time.RFC3339)),
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *datalakeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan datalakeResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := plan.Name.ValueString()

	input := traceforce.UpdateDatalakeRequest{
		Name: &name,
	}

	datalake, err := r.client.UpdateDatalake(plan.ID.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError("Error updating datalake", err.Error())
		return
	}

	plan = datalakeResourceModel{
		ID:        types.StringValue(datalake.ID),
		ProjectId: types.StringValue(datalake.HostingEnvironmentID),
		Type:      types.StringValue(string(datalake.Type)),
		Name:      types.StringValue(datalake.Name),
		Status:    types.StringValue(string(datalake.Status)),
		CreatedAt: types.StringValue(datalake.CreatedAt.Format(time.RFC3339)),
		UpdatedAt: types.StringValue(datalake.UpdatedAt.Format(time.RFC3339)),
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *datalakeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state datalakeResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteDatalake(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting datalake", err.Error())
		return
	}
}

func (r *datalakeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import datalake by id
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
