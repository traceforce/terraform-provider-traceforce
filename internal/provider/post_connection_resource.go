// Copyright (c) Traceforce, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	traceforce "github.com/traceforce/traceforce-go-sdk"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &postConnectionResource{}
	_ resource.ResourceWithConfigure   = &postConnectionResource{}
	_ resource.ResourceWithImportState = &postConnectionResource{}
)

// NewPostConnectionResource is a helper function to simplify the provider implementation.
func NewPostConnectionResource() resource.Resource {
	return &postConnectionResource{}
}

// postConnectionResource is the resource implementation.
type postConnectionResource struct {
	client *traceforce.Client
}

// postConnectionResourceModel maps post_connection schema data.
type postConnectionResourceModel struct {
	ProjectId types.String `tfsdk:"project_id"`
	ID        types.String `tfsdk:"id"`
	Status    types.String `tfsdk:"status"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

// Metadata returns the resource type name.
func (r *postConnectionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_post_connection"
}

func (r *postConnectionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *postConnectionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "ID of the project to post-connect.",
				Required:    true,
			},
			// The following attributes are computed and should never be reflected in changes.
			//e need to set them to unknown when the resource is created
			"status": schema.StringAttribute{
				Description: "Status of the connection. For example, connected, disconnected, etc.",
				Computed:    true,
			},
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
func (r *postConnectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan postConnectionResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Execute post-connection process using the project ID
	connection, err := r.client.PostConnection(plan.ProjectId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error executing post-connection", err.Error())
		return
	}

	plan = postConnectionResourceModel{
		ProjectId: plan.ProjectId,
		ID:        types.StringValue(connection.ID),
		Status:    types.StringValue(string(connection.Status)),
		CreatedAt: types.StringValue(connection.CreatedAt.Format(time.RFC3339)),
		UpdatedAt: types.StringValue(connection.UpdatedAt.Format(time.RFC3339)),
	}

	// Set state
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *postConnectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// This is a no-op for this resource. A new post connection resource is an event notification and
	// is always created whenever declared in main.tf.
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *postConnectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// This is a no-op for this resource.
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *postConnectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// This is a no-op for this resource.
}

func (r *postConnectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// This is a no-op for this resource.
}
