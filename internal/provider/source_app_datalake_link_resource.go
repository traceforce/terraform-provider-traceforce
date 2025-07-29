// Copyright (c) Traceforce, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
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
	_ resource.Resource                = &sourceAppDatalakeLinkResource{}
	_ resource.ResourceWithConfigure   = &sourceAppDatalakeLinkResource{}
	_ resource.ResourceWithImportState = &sourceAppDatalakeLinkResource{}
)

// NewSourceAppDatalakeLinkResource creates a new source app datalake link resource.
func NewSourceAppDatalakeLinkResource() resource.Resource {
	return &sourceAppDatalakeLinkResource{}
}

// sourceAppDatalakeLinkResource is the resource implementation.
type sourceAppDatalakeLinkResource struct {
	client *traceforce.Client
}

// sourceAppDatalakeLinkResourceModel maps source app datalake link schema data.
type sourceAppDatalakeLinkResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	SourceAppID          types.String `tfsdk:"source_app_id"`
	DatalakeID           types.String `tfsdk:"datalake_id"`
	HostingEnvironmentID types.String `tfsdk:"hosting_environment_id"`
	CreatedAt            types.String `tfsdk:"created_at"`
	UpdatedAt            types.String `tfsdk:"updated_at"`
}

// Metadata returns the resource type name.
func (r *sourceAppDatalakeLinkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_source_app_datalake_link"
}

func (r *sourceAppDatalakeLinkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *sourceAppDatalakeLinkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Links a source app to a datalake within the same hosting environment.",
		Attributes: map[string]schema.Attribute{
			"source_app_id": schema.StringAttribute{
				Description: "ID of the source app to link.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"datalake_id": schema.StringAttribute{
				Description: "ID of the datalake to link.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			// The following attributes are computed and should never be reflected in changes.
			"hosting_environment_id": schema.StringAttribute{
				Description: "ID of the hosting environment (derived from linked resources).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"id": schema.StringAttribute{
				Description: "System generated ID of the link",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "Date and time the link was created",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "Date and time the link was last updated",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *sourceAppDatalakeLinkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan sourceAppDatalakeLinkResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := traceforce.CreateSourceAppDatalakeLinkRequest{
		SourceAppID: plan.SourceAppID.ValueString(),
		DatalakeID:  plan.DatalakeID.ValueString(),
	}

	link, err := r.client.CreateSourceAppDatalakeLink(input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating source app datalake link", err.Error())
		return
	}

	plan = sourceAppDatalakeLinkResourceModel{
		ID:                   types.StringValue(link.ID),
		SourceAppID:          types.StringValue(link.SourceAppID),
		DatalakeID:           types.StringValue(link.DatalakeID),
		HostingEnvironmentID: types.StringValue(link.HostingEnvironmentID),
		CreatedAt:            types.StringValue(link.CreatedAt.Format(time.RFC3339)),
		UpdatedAt:            types.StringValue(link.UpdatedAt.Format(time.RFC3339)),
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *sourceAppDatalakeLinkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state sourceAppDatalakeLinkResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	link, err := r.client.GetSourceAppDatalakeLink(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading source app datalake link", err.Error())
		return
	}

	state = sourceAppDatalakeLinkResourceModel{
		ID:                   types.StringValue(link.ID),
		SourceAppID:          types.StringValue(link.SourceAppID),
		DatalakeID:           types.StringValue(link.DatalakeID),
		HostingEnvironmentID: types.StringValue(link.HostingEnvironmentID),
		CreatedAt:            types.StringValue(link.CreatedAt.Format(time.RFC3339)),
		UpdatedAt:            types.StringValue(link.UpdatedAt.Format(time.RFC3339)),
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *sourceAppDatalakeLinkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Source app datalake links are immutable - any change requires replacement
	// This method should never be called due to RequiresReplace plan modifiers
	resp.Diagnostics.AddError(
		"Update not supported",
		"Source app datalake links cannot be updated. Any changes require replacement of the resource.",
	)
}

func (r *sourceAppDatalakeLinkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state sourceAppDatalakeLinkResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteSourceAppDatalakeLink(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting source app datalake link", err.Error())
		return
	}
}

func (r *sourceAppDatalakeLinkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import source app datalake link by id
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
