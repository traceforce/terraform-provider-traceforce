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
	_ resource.Resource                = &connectionResource{}
	_ resource.ResourceWithConfigure   = &connectionResource{}
	_ resource.ResourceWithImportState = &connectionResource{}
)

// NewConnectionResource is a helper function to simplify the provider implementation.
func NewConnectionResource() resource.Resource {
	return &connectionResource{}
}

// connectionResource is the resource implementation.
type connectionResource struct {
	client *traceforce.Client
}

// connectionResourceModel maps connections schema data.
type connectionResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	CreatedAt           types.String `tfsdk:"created_at"`
	UpdatedAt           types.String `tfsdk:"updated_at"`
	Name                types.String `tfsdk:"name"`
	EnvironmentType     types.String `tfsdk:"environment_type"`
	EnvironmentNativeId types.String `tfsdk:"environment_native_id"`
	Status              types.String `tfsdk:"status"`
}

// Metadata returns the resource type name.
func (r *connectionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connection"
}

func (r *connectionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *connectionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "Name of the connection. This value must be unique.",
				Required:    true,
			},
			"environment_type": schema.StringAttribute{
				Description: "Type of environment the connection is connected to. For example, AWS, Azure, GCP, etc.",
				Required:    true,
			},
			"environment_native_id": schema.StringAttribute{
				Description: "Native ID of the environment the connection is connected to. For example, an AWS account ID, an Azure subscription ID, a GCP project ID, etc.",
				Required:    true,
			},
			"status": schema.StringAttribute{
				Description: "Status of the connection. For example, connected, disconnected, etc.",
				Required:    true,
			},
			// The following attributes are computed and should never be reflected in changes.
			//e need to set them to unknown when the resource is created
			"id": schema.StringAttribute{
				Description: "System generated ID of the connection",
				Computed:    true,
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "Date and time the connection was created",
				Computed:    true,
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "Date and time the connection was last updated",
				Computed:    true,
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *connectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan connectionResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	input := traceforce.ConnectionsModel{
		Name:                plan.Name.ValueString(),
		EnvironmentType:     plan.EnvironmentType.ValueString(),
		EnvironmentNativeId: plan.EnvironmentNativeId.ValueString(),
		Status:              plan.Status.ValueString(),
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	// Insert connection
	connection, err := r.client.CreateConnection(input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating connection", err.Error())
		return
	}

	plan = connectionResourceModel{
		ID:                  types.StringValue(connection.ID),
		CreatedAt:           types.StringValue(connection.CreatedAt.Format(time.RFC3339)),
		UpdatedAt:           types.StringValue(connection.UpdatedAt.Format(time.RFC3339)),
		Name:                types.StringValue(connection.Name),
		EnvironmentType:     types.StringValue(connection.EnvironmentType),
		EnvironmentNativeId: types.StringValue(connection.EnvironmentNativeId),
		Status:              types.StringValue(connection.Status),
	}

	// Set state
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *connectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state connectionResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	connection, err := r.client.GetConnectionByName(state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading a single connection by id", err.Error())
		return
	}

	state = connectionResourceModel{
		ID:                  types.StringValue(connection.ID),
		CreatedAt:           types.StringValue(connection.CreatedAt.Format(time.RFC3339)),
		UpdatedAt:           types.StringValue(connection.UpdatedAt.Format(time.RFC3339)),
		Name:                types.StringValue(connection.Name),
		EnvironmentType:     types.StringValue(connection.EnvironmentType),
		EnvironmentNativeId: types.StringValue(connection.EnvironmentNativeId),
		Status:              types.StringValue(connection.Status),
	}

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *connectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan connectionResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get connection from API
	input, err := r.client.GetConnectionByName(plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading a single connection by id", err.Error())
		return
	}

	input.ID = plan.ID.ValueString()
	input.Name = plan.Name.ValueString()
	input.EnvironmentType = plan.EnvironmentType.ValueString()
	input.EnvironmentNativeId = plan.EnvironmentNativeId.ValueString()
	input.Status = plan.Status.ValueString()
	input.UpdatedAt = time.Now()

	// Update connection
	connection, err := r.client.UpdateConnection(input.ID, *input)
	if err != nil {
		resp.Diagnostics.AddError("Error updating connection", err.Error())
		return
	}

	plan = connectionResourceModel{
		ID:                  types.StringValue(connection.ID),
		CreatedAt:           types.StringValue(connection.CreatedAt.Format(time.RFC3339)),
		UpdatedAt:           types.StringValue(connection.UpdatedAt.Format(time.RFC3339)),
		Name:                types.StringValue(connection.Name),
		EnvironmentType:     types.StringValue(connection.EnvironmentType),
		EnvironmentNativeId: types.StringValue(connection.EnvironmentNativeId),
		Status:              types.StringValue(connection.Status),
	}

	// Set state
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *connectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state connectionResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	connection, err := r.client.GetConnectionByName(state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading a single connection by id", err.Error())
		return
	}

	// Delete connection
	err = r.client.DeleteConnection(connection.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting connection", err.Error())
		return
	}
}

func (r *connectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import connection by name and save to name attribute
	resource.ImportStatePassthroughWithIdentity(ctx, path.Root("name"), path.Root("id"), req, resp)
}
