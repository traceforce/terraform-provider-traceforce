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
	_ resource.Resource                = &sourceAppResource{}
	_ resource.ResourceWithConfigure   = &sourceAppResource{}
	_ resource.ResourceWithImportState = &sourceAppResource{}
)

// NewSourceAppResource is a helper function to simplify the provider implementation.
func NewSourceAppResource() resource.Resource {
	return &sourceAppResource{}
}

// sourceAppResource is the resource implementation.
type sourceAppResource struct {
	client *traceforce.Client
}

// sourceAppResourceModel maps source_apps schema data.
type sourceAppResourceModel struct {
	ID               types.String `tfsdk:"id"`
	DatalakeId       types.String `tfsdk:"datalake_id"`
	Type             types.String `tfsdk:"type"`
	Name             types.String `tfsdk:"name"`
	Status           types.String `tfsdk:"status"`
	CreatedAt        types.String `tfsdk:"created_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
}

// Metadata returns the resource type name.
func (r *sourceAppResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_source_app"
}

func (r *sourceAppResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Add a nil check when handling ProviderData because Terraform
	// sets that data after it calls the ConfigureProvider RPC.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*traceforce.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *hashicups.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

// Schema defines the schema for the resource.
func (r *sourceAppResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"datalake_id": schema.StringAttribute{
				Description: "ID of the datalake this source app belongs to.",
				Required:    true,
				ForceNew:    true,
			},
			"type": schema.StringAttribute{
				Description: fmt.Sprintf("Type of source app. Currently supported: %s.", 
					traceforce.SourceAppTypeSalesforce),
				Required:    true,
				ForceNew:    true,
			},
			"name": schema.StringAttribute{
				Description: "Name of the source app. This value must be unique within a datalake.",
				Required:    true,
			},
			// The following attributes are computed and should never be reflected in changes.
			"status": schema.StringAttribute{
				Description: fmt.Sprintf("Status of the source app. Valid values: %s, %s, %s.", 
					traceforce.SourceAppStatusPending,
					traceforce.SourceAppStatusDisconnected,
					traceforce.SourceAppStatusConnected),
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"id": schema.StringAttribute{
				Description: "System generated ID of the source app",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "Date and time the source app was created",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "Date and time the source app was last updated",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *sourceAppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan sourceAppResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := traceforce.CreateSourceAppRequest{
		DatalakeID: plan.DatalakeId.ValueString(),
		Type:       traceforce.SourceAppType(plan.Type.ValueString()),
		Name:       plan.Name.ValueString(),
	}

	sourceApp, err := r.client.CreateSourceApp(input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating source app", err.Error())
		return
	}

	plan = sourceAppResourceModel{
		ID:         types.StringValue(sourceApp.ID),
		DatalakeId: types.StringValue(sourceApp.DatalakeID),
		Type:       types.StringValue(string(sourceApp.Type)),
		Name:       types.StringValue(sourceApp.Name),
		Status:     types.StringValue(string(sourceApp.Status)),
		CreatedAt:  types.StringValue(sourceApp.CreatedAt.Format(time.RFC3339)),
		UpdatedAt:  types.StringValue(sourceApp.UpdatedAt.Format(time.RFC3339)),
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *sourceAppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state sourceAppResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	sourceApp, err := r.client.GetSourceApp(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading source app", err.Error())
		return
	}

	state = sourceAppResourceModel{
		ID:         types.StringValue(sourceApp.ID),
		DatalakeId: types.StringValue(sourceApp.DatalakeID),
		Type:       types.StringValue(string(sourceApp.Type)),
		Name:       types.StringValue(sourceApp.Name),
		Status:     types.StringValue(string(sourceApp.Status)),
		CreatedAt:  types.StringValue(sourceApp.CreatedAt.Format(time.RFC3339)),
		UpdatedAt:  types.StringValue(sourceApp.UpdatedAt.Format(time.RFC3339)),
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *sourceAppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan sourceAppResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := plan.Name.ValueString()

	input := traceforce.UpdateSourceAppRequest{
		Name: &name,
	}

	sourceApp, err := r.client.UpdateSourceApp(plan.ID.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError("Error updating source app", err.Error())
		return
	}

	plan = sourceAppResourceModel{
		ID:         types.StringValue(sourceApp.ID),
		DatalakeId: types.StringValue(sourceApp.DatalakeID),
		Type:       types.StringValue(string(sourceApp.Type)),
		Name:       types.StringValue(sourceApp.Name),
		Status:     types.StringValue(string(sourceApp.Status)),
		CreatedAt:  types.StringValue(sourceApp.CreatedAt.Format(time.RFC3339)),
		UpdatedAt:  types.StringValue(sourceApp.UpdatedAt.Format(time.RFC3339)),
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *sourceAppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state sourceAppResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteSourceApp(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting source app", err.Error())
		return
	}
}

func (r *sourceAppResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import source app by id
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}