// Copyright (c) Traceforce, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	traceforce "github.com/traceforce/traceforce-go-sdk"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &cloudConnectionDeploymentResource{}
	_ resource.ResourceWithConfigure = &cloudConnectionDeploymentResource{}
)

// NewCloudConnectionDeploymentResource records the collector Terraform created
// for a cloud connection (after the Lambda / Function App exists).
func NewCloudConnectionDeploymentResource() resource.Resource {
	return &cloudConnectionDeploymentResource{}
}

type cloudConnectionDeploymentResource struct {
	client *traceforce.Client
}

type cloudConnectionDeploymentResourceModel struct {
	ConnectionID types.String `tfsdk:"connection_id"`
	CollectorID  types.String `tfsdk:"collector_id"`
	IdentityID   types.String `tfsdk:"identity_id"`
	Regions      types.List   `tfsdk:"regions"`
	DeployedAt   types.String `tfsdk:"deployed_at"`
}

func (r *cloudConnectionDeploymentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloud_connection_deployment"
}

func (r *cloudConnectionDeploymentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = clientFrom(req.ProviderData, &resp.Diagnostics)
}

func (r *cloudConnectionDeploymentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The collector Terraform created for a cloud connection. Destroy makes no call: disconnecting the connection clears it.",
		Attributes: map[string]schema.Attribute{
			"connection_id": schema.StringAttribute{
				Description:   "ID of the traceforce_cloud_connection.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"collector_id": schema.StringAttribute{
				Description: "AWS: Lambda ARN. Microsoft 365: Function App resource ID.",
				Required:    true,
			},
			"identity_id": schema.StringAttribute{
				Description: "AWS: IAM role ARN. Microsoft 365: managed identity object ID.",
				Required:    true,
			},
			"regions": schema.ListAttribute{
				Description: "AWS Regions the collector reads (AWS only).",
				Optional:    true,
				ElementType: types.StringType,
			},
			// The following attribute is computed.
			"deployed_at": schema.StringAttribute{
				Description: "When Traceforce recorded this deployment.",
				Computed:    true,
			},
		},
	}
}

// record sends the deployment to Traceforce and reads back deployed_at (create and update).
func (r *cloudConnectionDeploymentResource) record(ctx context.Context, plan *cloudConnectionDeploymentResourceModel) diag.Diagnostics {
	var regions []string
	diags := plan.Regions.ElementsAs(ctx, &regions, false)
	if diags.HasError() {
		return diags
	}
	id := plan.ConnectionID.ValueString()
	err := r.client.PutCloudConnectionDeployment(id, traceforce.CloudConnectionDeployment{
		CollectorID: plan.CollectorID.ValueString(),
		IdentityID:  plan.IdentityID.ValueString(),
		Regions:     regions,
	})
	if err != nil {
		diags.AddError("Error recording cloud connection deployment", err.Error())
		return diags
	}
	conn, err := r.client.GetCloudConnection(id)
	if err != nil {
		diags.AddError("Error reading cloud connection", err.Error())
		return diags
	}
	plan.DeployedAt = deployedAt(conn)
	return diags
}

// deployedAt is the connection's deployed_at as RFC 3339, or null when not recorded.
func deployedAt(conn *traceforce.CloudConnection) types.String {
	if conn == nil || conn.DeployedAt == nil {
		return types.StringNull()
	}
	return types.StringValue(conn.DeployedAt.Format(time.RFC3339))
}

func (r *cloudConnectionDeploymentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan cloudConnectionDeploymentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(r.record(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *cloudConnectionDeploymentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state cloudConnectionDeploymentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := r.client.GetCloudConnection(state.ConnectionID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading cloud connection", err.Error())
		return
	}
	// Connection gone, or the deployment cleared (terraform destroy disconnected it):
	// the next apply records the deployment again.
	if conn == nil || conn.DeployedAt == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	state.DeployedAt = deployedAt(conn)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *cloudConnectionDeploymentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan cloudConnectionDeploymentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(r.record(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete makes no call: on destroy the connection's DELETE clears deployed_at.
func (r *cloudConnectionDeploymentResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}
