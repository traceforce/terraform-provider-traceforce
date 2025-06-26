// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	supabase "github.com/supabase-community/supabase-go"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &connectionResource{}
	_ resource.ResourceWithConfigure = &connectionResource{}
)

// NewConnectionResource is a helper function to simplify the provider implementation.
func NewConnectionResource() resource.Resource {
	return &connectionResource{}
}

// connectionResource is the resource implementation.
type connectionResource struct {
	client *supabase.Client
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

	client, ok := req.ProviderData.(*supabase.Client)

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
			"id": schema.StringAttribute{
				Description: "System generated ID of the connection",
				Computed:    true,
				Optional:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "Date and time the connection was created",
				Computed:    true,
				Optional:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "Date and time the connection was last updated",
				Computed:    true,
				Optional:    true,
			},
			"name": schema.StringAttribute{
				Description: "Name of the connection. This must be unique.",
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
	connection := &connectionsOriginalModel{
		ID:                  uuid.New().String(),
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
		Name:                plan.Name.ValueString(),
		EnvironmentType:     plan.EnvironmentType.ValueString(),
		EnvironmentNativeId: plan.EnvironmentNativeId.ValueString(),
		Status:              plan.Status.ValueString(),
	}

	// Insert connection
	result, _, err := r.client.From("connections").Insert(connection, false, "", "", "").Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error creating connection", err.Error())
		return
	}

	connection, err = verifyUniqueConnectionResult(result)
	if err != nil {
		resp.Diagnostics.AddError("Error verifying result during creation", err.Error())
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

	// Get connection from API
	result, _, err := r.client.From("connections").Select("*", "", false).Eq("id", state.ID.ValueString()).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error selecting connections", err.Error())
		return
	}

	connection, err := verifyUniqueConnectionResult(result)
	if err != nil {
		resp.Diagnostics.AddError("Error verifying connection during read", err.Error())
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
	result, _, err := r.client.From("connections").Select("*", "", false).Eq("name", plan.Name.ValueString()).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error selecting connections", err.Error())
		return
	}

	connection, err := verifyUniqueConnectionResult(result)
	if err != nil {
		resp.Diagnostics.AddError("Error verifying connection during update", err.Error())
		return
	}

	connection.UpdatedAt = time.Now()
	connection.Name = plan.Name.ValueString()
	connection.EnvironmentType = plan.EnvironmentType.ValueString()
	connection.EnvironmentNativeId = plan.EnvironmentNativeId.ValueString()
	connection.Status = plan.Status.ValueString()

	// Update connection
	result, _, err = r.client.From("connections").Update(connection, plan.ID.ValueString(), "").Eq("name", plan.Name.ValueString()).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error updating connection", err.Error())
		return
	}

	connection, err = verifyUniqueConnectionResult(result)
	if err != nil {
		resp.Diagnostics.AddError("Error verifying connection during update", err.Error())
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

	// Delete connection
	_, _, err := r.client.From("connections").Delete("", "").Eq("name", state.Name.ValueString()).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error deleting connection", err.Error())
		return
	}
}

func verifyUniqueConnectionResult(result []byte) (*connectionsOriginalModel, error) {
	var connections []connectionsOriginalModel
	err := json.Unmarshal(result, &connections)
	if err != nil {
		return nil, err
	}

	if len(connections) == 0 {
		return nil, nil
	}

	if len(connections) > 1 {
		return nil, nil
	}

	return &connections[0], nil
}
