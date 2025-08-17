// Copyright (c) Traceforce, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	traceforce "github.com/traceforce/traceforce-go-sdk"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &postConnectionResource{}
	_ resource.ResourceWithConfigure   = &postConnectionResource{}
	_ resource.ResourceWithImportState = &postConnectionResource{}
)

// NewPostConnectionResource creates a new post connection resource.
func NewPostConnectionResource() resource.Resource {
	return &postConnectionResource{}
}

// postConnectionResource is the resource implementation.
type postConnectionResource struct {
	client *traceforce.Client
}

// baseInfrastructureModel maps base infrastructure schema data.
type baseInfrastructureModel struct {
	DataplaneIdentityIdentifier   types.String `tfsdk:"dataplane_identity_identifier"`
	WorkloadIdentityProviderName  types.String `tfsdk:"workload_identity_provider_name"`
	AuthViewGeneratorFunctionID   types.String `tfsdk:"auth_view_generator_function_id"`
	AuthViewGeneratorFunctionURL  types.String `tfsdk:"auth_view_generator_function_url"`
	TraceforceBucketName          types.String `tfsdk:"traceforce_bucket_name"`
}

// infrastructureModel maps infrastructure schema data.
type infrastructureModel struct {
	Base       *baseInfrastructureModel       `tfsdk:"base"`
	BigQuery   *bigqueryInfrastructureModel   `tfsdk:"bigquery"`
	Salesforce *salesforceInfrastructureModel `tfsdk:"salesforce"`
}

// bigqueryInfrastructureModel maps bigquery infrastructure schema data.
type bigqueryInfrastructureModel struct {
	TraceforceSchema            types.String `tfsdk:"traceforce_schema"`
	TraceforceSecureViewsSchema types.String `tfsdk:"traceforce_secure_views_schema"`
	EventsSubscriptionName      types.String `tfsdk:"events_subscription_name"`
}

// salesforceInfrastructureModel maps salesforce infrastructure schema data.
type salesforceInfrastructureModel struct {
	ClientID     types.String `tfsdk:"salesforce_client_id"`
	Domain       types.String `tfsdk:"salesforce_domain"`
	ClientSecret types.String `tfsdk:"salesforce_client_secret"`
}

// postConnectionResourceModel maps post_connection schema data.
type postConnectionResourceModel struct {
	TraceforceHostingEnvironmentId types.String        `tfsdk:"traceforce_hosting_environment_id"`
	Infrastructure                 infrastructureModel `tfsdk:"infrastructure"`
	TerraformURL                   types.String        `tfsdk:"terraform_url"`
	TerraformModuleVersions        types.String        `tfsdk:"terraform_module_versions"`
	TerraformModuleVersionsHash    types.String        `tfsdk:"terraform_module_versions_hash"`
	DeployedDatalakeIds            types.List          `tfsdk:"deployed_datalake_ids"`
	DeployedSourceAppIds           types.List          `tfsdk:"deployed_source_app_ids"`
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
			"traceforce_hosting_environment_id": schema.StringAttribute{
				Description: "ID of the TraceForce hosting environment to post-connect.",
				Required:    true,
			},
			"infrastructure": schema.SingleNestedAttribute{
				Description: "Infrastructure configuration for deployment",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"base": schema.SingleNestedAttribute{
						Description: "Base infrastructure outputs",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"dataplane_identity_identifier": schema.StringAttribute{
								Description: "Dataplane identity identifier for base infrastructure",
								Required:    true,
							},
							"workload_identity_provider_name": schema.StringAttribute{
								Description: "Workload identity provider name for external authentication",
								Required:    true,
							},
							"auth_view_generator_function_id": schema.StringAttribute{
								Description: "Auth view generator function ID",
								Required:    true,
							},
							"auth_view_generator_function_url": schema.StringAttribute{
								Description: "Auth view generator function URL",
								Required:    true,
							},
							"traceforce_bucket_name": schema.StringAttribute{
								Description: "TraceForce bucket name for artifact storage",
								Required:    true,
							},
						},
					},
					"bigquery": schema.SingleNestedAttribute{
						Description: "BigQuery datalake infrastructure outputs",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"traceforce_schema": schema.StringAttribute{
								Description: "BigQuery dataset ID for TraceForce schema",
								Required:    true,
							},
							"traceforce_secure_views_schema": schema.StringAttribute{
								Description: "BigQuery dataset ID for TraceForce secure views schema",
								Required:    true,
							},
							"events_subscription_name": schema.StringAttribute{
								Description: "PubSub subscription name for BigQuery events",
								Required:    true,
							},
						},
					},
					"salesforce": schema.SingleNestedAttribute{
						Description: "Salesforce source app infrastructure outputs",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"salesforce_client_id": schema.StringAttribute{
								Description: "Salesforce connected app client ID",
								Required:    true,
							},
							"salesforce_domain": schema.StringAttribute{
								Description: "Salesforce domain (e.g., mycompany.my.salesforce.com)",
								Required:    true,
							},
							"salesforce_client_secret": schema.StringAttribute{
								Description: "Secret Manager resource name for Salesforce client secret",
								Required:    true,
							},
						},
					},
				},
			},
			"terraform_url": schema.StringAttribute{
				Description: "URL of the Terraform module repository",
				Required:    true,
			},
			"terraform_module_versions": schema.StringAttribute{
				Description: "JSON string containing Terraform module versions",
				Required:    true,
			},
			"terraform_module_versions_hash": schema.StringAttribute{
				Description: "Hash of the Terraform module versions for integrity verification",
				Required:    true,
			},
			"deployed_datalake_ids": schema.ListAttribute{
				Description: "List of datalake IDs that were deployed by terraform",
				Required:    true,
				ElementType: types.StringType,
			},
			"deployed_source_app_ids": schema.ListAttribute{
				Description: "List of source app IDs that were deployed by terraform",
				Required:    true,
				ElementType: types.StringType,
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

	if err := r.executePostConnection(ctx, plan); err != nil {
		resp.Diagnostics.AddError("Error executing post-connection", err.Error())
		return
	}

	// Set state with the plan data
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// executePostConnection handles the post-connection logic.
func (r *postConnectionResource) executePostConnection(ctx context.Context, plan postConnectionResourceModel) error {
	postConnReq := &traceforce.PostConnectionRequest{
		Infrastructure: &traceforce.Infrastructure{},
	}

	// Add Base configuration if present
	if plan.Infrastructure.Base != nil {
		baseInfra := &traceforce.BaseInfrastructure{
			DataplaneIdentityIdentifier:   plan.Infrastructure.Base.DataplaneIdentityIdentifier.ValueString(),
			WorkloadIdentityProviderName:  plan.Infrastructure.Base.WorkloadIdentityProviderName.ValueString(),
			AuthViewGeneratorFunctionID:   plan.Infrastructure.Base.AuthViewGeneratorFunctionID.ValueString(),
			AuthViewGeneratorFunctionURL:  plan.Infrastructure.Base.AuthViewGeneratorFunctionURL.ValueString(),
			TraceforceBucketName:          plan.Infrastructure.Base.TraceforceBucketName.ValueString(),
		}

		postConnReq.Infrastructure.Base = baseInfra
	}

	// Add BigQuery configuration if present
	if plan.Infrastructure.BigQuery != nil {
		postConnReq.Infrastructure.BigQuery = &traceforce.BigQueryInfrastructure{
			TraceforceSchema:            plan.Infrastructure.BigQuery.TraceforceSchema.ValueString(),
			TraceforceSecureViewsSchema: plan.Infrastructure.BigQuery.TraceforceSecureViewsSchema.ValueString(),
			EventsSubscriptionName:      plan.Infrastructure.BigQuery.EventsSubscriptionName.ValueString(),
		}
	}

	// Add Salesforce configuration if present
	if plan.Infrastructure.Salesforce != nil {
		postConnReq.Infrastructure.Salesforce = &traceforce.SalesforceInfrastructure{
			ClientID:     plan.Infrastructure.Salesforce.ClientID.ValueString(),
			Domain:       plan.Infrastructure.Salesforce.Domain.ValueString(),
			ClientSecret: plan.Infrastructure.Salesforce.ClientSecret.ValueString(),
		}
	}

	// Add terraform metadata
	postConnReq.TerraformURL = plan.TerraformURL.ValueString()
	postConnReq.TerraformModuleVersions = plan.TerraformModuleVersions.ValueString()
	postConnReq.TerraformModuleVersionsHash = plan.TerraformModuleVersionsHash.ValueString()

	// Add deployed resource IDs
	diags := plan.DeployedDatalakeIds.ElementsAs(ctx, &postConnReq.DeployedDatalakeIds, false)
	if diags.HasError() {
		return fmt.Errorf("failed to extract deployed datalake IDs: %v", diags)
	}

	diags = plan.DeployedSourceAppIds.ElementsAs(ctx, &postConnReq.DeployedSourceAppIds, false)
	if diags.HasError() {
		return fmt.Errorf("failed to extract deployed source app IDs: %v", diags)
	}

	// Execute post-connection process using the hosting environment ID and structured request
	err := r.client.PostConnection(plan.TraceforceHostingEnvironmentId.ValueString(), postConnReq)

	return err
}

// Read refreshes the Terraform state with the latest data.
func (r *postConnectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state postConnectionResourceModel

	// Get current state from Terraform
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// For this resource, we simply maintain the current state
	// This allows Terraform to detect changes and trigger re-deployment
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *postConnectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan postConnectionResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.executePostConnection(ctx, plan); err != nil {
		resp.Diagnostics.AddError("Error executing post-connection", err.Error())
		return
	}

	// Set state with the plan data
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *postConnectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// This is a no-op for this resource.
}

func (r *postConnectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// This is a no-op for this resource.
}
