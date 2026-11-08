// Copyright (c) Traceforce, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	_ resource.Resource                   = &cloudConnectionDeploymentResource{}
	_ resource.ResourceWithConfigure      = &cloudConnectionDeploymentResource{}
	_ resource.ResourceWithValidateConfig = &cloudConnectionDeploymentResource{}
)

// NewCloudConnectionDeploymentResource records in Traceforce the collector
// Terraform created for a cloud connection, and clears it on destroy.
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
	Regions      types.Set    `tfsdk:"regions"`
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
		Description: "Records in Traceforce the collector (Lambda / Function App) and the identity deployed for a " +
			"traceforce_cloud_connection. If the deployment is cleared in Traceforce, the next apply records it again. " +
			"Destroy clears it.",
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
			"regions": schema.SetAttribute{
				Description: "AWS Regions the collector reads (AWS only, at least one; order does not matter). " +
					"Sent only when set: leaving it out keeps the regions Traceforce has, which the collector's uploads may also update.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"deployed_at": schema.StringAttribute{
				Description: "When Traceforce recorded this deployment.",
				Computed:    true,
			},
		},
	}
}

// ValidateConfig refuses an empty regions set: it would not clear the stored regions.
func (r *cloudConnectionDeploymentResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var regions types.Set
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("regions"), &regions)...)
	if !regions.IsNull() && !regions.IsUnknown() && len(regions.Elements()) == 0 {
		resp.Diagnostics.AddAttributeError(path.Root("regions"), "Empty regions",
			"Set at least one region, or leave regions out (the stored regions are kept).")
	}
}

// backendHint explains an answer that comes from a backend without these routes.
func backendHint(err error) string {
	var apiErr *traceforce.APIError
	if errors.As(err, &apiErr) && (apiErr.StatusCode == http.StatusMethodNotAllowed ||
		(apiErr.StatusCode == http.StatusNotFound && !traceforce.IsCloudConnectionNotFound(err))) {
		return "\n\nThe Traceforce API at this endpoint has no cloud-connection deployment route; it may be older than this provider. Update the Traceforce backend."
	}
	return ""
}

// record sends the deployment (create and update) and keeps deployed_at from the answer.
func (r *cloudConnectionDeploymentResource) record(ctx context.Context, plan *cloudConnectionDeploymentResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	// Known at apply (not computed); nil when left out: none sent, the stored ones are kept.
	var regions []string
	diags.Append(plan.Regions.ElementsAs(ctx, &regions, false)...)
	if diags.HasError() {
		return diags
	}
	conn, err := r.client.PutCloudConnectionDeployment(plan.ConnectionID.ValueString(), traceforce.CloudConnectionDeployment{
		CollectorID: plan.CollectorID.ValueString(),
		IdentityID:  plan.IdentityID.ValueString(),
		Regions:     regions,
	})
	if err != nil {
		diags.AddError("Error recording cloud connection deployment", err.Error()+backendHint(err))
		return diags
	}
	if conn.DeployedAt == nil {
		diags.AddError("Error recording cloud connection deployment",
			fmt.Sprintf("Traceforce answered connection %s without deployedAt, so the deployment was not recorded.", conn.ID))
		return diags
	}
	plan.DeployedAt = types.StringValue(conn.DeployedAt.Format(time.RFC3339))
	return diags
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

// Read refreshes the Terraform state with the latest data. Regions are not
// refreshed: the collector's uploads may replace them.
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
	// Connection gone, or its deployment cleared: the next apply records it again.
	if conn == nil || conn.DeployedAt == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	state.CollectorID = types.StringPointerValue(conn.CollectorID)
	state.IdentityID = types.StringPointerValue(conn.IdentityID)
	state.DeployedAt = types.StringValue(conn.DeployedAt.Format(time.RFC3339))
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

// Delete clears the deployment in Traceforce (terraform destroy, or the
// resource removed from the configuration). A connection that is gone has
// nothing to clear. The backend must have the deployment DELETE route: a
// plain 404 / 405 stops here with the "update the backend" hint.
func (r *cloudConnectionDeploymentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state cloudConnectionDeploymentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if _, err := r.client.DeleteCloudConnectionDeployment(state.ConnectionID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error clearing cloud connection deployment", err.Error()+backendHint(err))
	}
}
