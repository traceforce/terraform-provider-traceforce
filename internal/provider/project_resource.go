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
	_ resource.Resource                = &projectResource{}
	_ resource.ResourceWithConfigure   = &projectResource{}
	_ resource.ResourceWithImportState = &projectResource{}
)

// NewProjectResource is a helper function to simplify the provider implementation.
func NewProjectResource() resource.Resource {
	return &projectResource{}
}

// projectResource is the resource implementation.
type projectResource struct {
	client *traceforce.Client
}

// projectResourceModel maps projects schema data.
type projectResourceModel struct {
	ID                               types.String `tfsdk:"id"`
	Name                             types.String `tfsdk:"name"`
	Type                             types.String `tfsdk:"type"`
	CloudProvider                    types.String `tfsdk:"cloud_provider"`
	NativeId                         types.String `tfsdk:"native_id"`
	Status                           types.String `tfsdk:"status"`
	ControlPlaneAwsAccountId         types.String `tfsdk:"control_plane_aws_account_id"`
	ControlPlaneRoleName             types.String `tfsdk:"control_plane_role_name"`
	CreatedAt                        types.String `tfsdk:"created_at"`
	UpdatedAt                        types.String `tfsdk:"updated_at"`
}

// Metadata returns the resource type name.
func (r *projectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (r *projectResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *projectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "Name of the project. This value must be unique.",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description: fmt.Sprintf("Type of project. Valid values: %s, %s.", 
					traceforce.HostingEnvironmentTypeCustomerManaged,
					traceforce.HostingEnvironmentTypeTraceForceManaged),
				Required:    true,
			},
			"cloud_provider": schema.StringAttribute{
				Description: fmt.Sprintf("Cloud provider for the project. Valid values: %s, %s, %s.", 
					traceforce.CloudProviderAWS,
					traceforce.CloudProviderGCP,
					traceforce.CloudProviderAzure),
				Required:    true,
			},
			"native_id": schema.StringAttribute{
				Description: "Native ID of the cloud project. For example, an AWS account ID, an Azure subscription ID, a GCP project ID, etc.",
				Required:    true,
			},
			// The following attributes are computed and should never be reflected in changes.
			"status": schema.StringAttribute{
				Description: fmt.Sprintf("Status of the project. Valid values: %s, %s, %s.", 
					traceforce.HostingEnvironmentStatusPending,
					traceforce.HostingEnvironmentStatusDisconnected,
					traceforce.HostingEnvironmentStatusConnected),
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"id": schema.StringAttribute{
				Description: "System generated ID of the project",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"control_plane_aws_account_id": schema.StringAttribute{
				Description: "AWS account ID for the control plane",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"control_plane_role_name": schema.StringAttribute{
				Description: "AWS IAM role name for the control plane",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "Date and time the project was created",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "Date and time the project was last updated",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *projectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan projectResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	input := traceforce.HostingEnvironment{
		Name:          plan.Name.ValueString(),
		Type:          traceforce.HostingEnvironmentType(plan.Type.ValueString()),
		CloudProvider: traceforce.CloudProvider(plan.CloudProvider.ValueString()),
		NativeID:      plan.NativeId.ValueString(),
		Status:        traceforce.HostingEnvironmentStatusPending,
	}

	// Create hosting environment (project)
	project, err := r.client.CreateHostingEnvironment(input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating hosting environment", err.Error())
		return
	}


	plan = projectResourceModel{
		ID:                               types.StringValue(project.ID),
		Name:                             types.StringValue(project.Name),
		Type:                             types.StringValue(string(project.Type)),
		CloudProvider:                    types.StringValue(string(project.CloudProvider)),
		NativeId:                         types.StringValue(project.NativeID),
		Status:                           types.StringValue(string(project.Status)),
		ControlPlaneAwsAccountId:         types.StringValue(project.ControlPlaneAwsAccountId),
		ControlPlaneRoleName:             types.StringValue(project.ControlPlaneRoleName),
		CreatedAt:                        types.StringValue(project.CreatedAt.Format(time.RFC3339)),
		UpdatedAt:                        types.StringValue(project.UpdatedAt.Format(time.RFC3339)),
	}

	// Set state
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *projectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state projectResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get hosting environment by ID
	project, err := r.client.GetHostingEnvironment(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading hosting environment", err.Error())
		return
	}


	state = projectResourceModel{
		ID:                               types.StringValue(project.ID),
		Name:                             types.StringValue(project.Name),
		Type:                             types.StringValue(string(project.Type)),
		CloudProvider:                    types.StringValue(string(project.CloudProvider)),
		NativeId:                         types.StringValue(project.NativeID),
		Status:                           types.StringValue(string(project.Status)),
		ControlPlaneAwsAccountId:         types.StringValue(project.ControlPlaneAwsAccountId),
		ControlPlaneRoleName:             types.StringValue(project.ControlPlaneRoleName),
		CreatedAt:                        types.StringValue(project.CreatedAt.Format(time.RFC3339)),
		UpdatedAt:                        types.StringValue(project.UpdatedAt.Format(time.RFC3339)),
	}

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *projectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan projectResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate update request from complete user-specified state
	input := traceforce.HostingEnvironment{
		ID:            plan.ID.ValueString(),
		Name:          plan.Name.ValueString(),
		Type:          traceforce.HostingEnvironmentType(plan.Type.ValueString()),
		CloudProvider: traceforce.CloudProvider(plan.CloudProvider.ValueString()),
		NativeID:      plan.NativeId.ValueString(),
	}

	// Update hosting environment
	project, err := r.client.UpdateHostingEnvironment(input.ID, input)
	if err != nil {
		resp.Diagnostics.AddError("Error updating hosting environment", err.Error())
		return
	}


	plan = projectResourceModel{
		ID:                               types.StringValue(project.ID),
		Name:                             types.StringValue(project.Name),
		Type:                             types.StringValue(string(project.Type)),
		CloudProvider:                    types.StringValue(string(project.CloudProvider)),
		NativeId:                         types.StringValue(project.NativeID),
		Status:                           types.StringValue(string(project.Status)),
		ControlPlaneAwsAccountId:         types.StringValue(project.ControlPlaneAwsAccountId),
		ControlPlaneRoleName:             types.StringValue(project.ControlPlaneRoleName),
		CreatedAt:                        types.StringValue(project.CreatedAt.Format(time.RFC3339)),
		UpdatedAt:                        types.StringValue(project.UpdatedAt.Format(time.RFC3339)),
	}

	// Set state
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *projectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state projectResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete hosting environment
	err := r.client.DeleteHostingEnvironment(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting hosting environment", err.Error())
		return
	}
}

func (r *projectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import project by name and save to name attribute
	resource.ImportStatePassthroughWithIdentity(ctx, path.Root("name"), path.Root("id"), req, resp)
}
