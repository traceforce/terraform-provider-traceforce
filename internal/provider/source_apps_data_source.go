// Copyright (c) Traceforce, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	traceforce "github.com/traceforce/traceforce-go-sdk"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &sourceAppsDataSource{}
	_ datasource.DataSourceWithConfigure = &sourceAppsDataSource{}
)

func NewSourceAppsDataSource() datasource.DataSource {
	return &sourceAppsDataSource{}
}

type sourceAppsDataSource struct {
	client *traceforce.Client
}

// sourceAppsDataSourceModel maps the data source schema data.
type sourceAppsDataSourceModel struct {
	HostingEnvironmentId types.String      `tfsdk:"hosting_environment_id"`
	SourceApps           []sourceAppsModel `tfsdk:"source_apps"`
}

// sourceAppsModel maps source apps schema data.
type sourceAppsModel struct {
	ID                   types.String `tfsdk:"id"`
	HostingEnvironmentId types.String `tfsdk:"hosting_environment_id"`
	Type                 types.String `tfsdk:"type"`
	Name                 types.String `tfsdk:"name"`
	Status               types.String `tfsdk:"status"`
	CreatedAt            types.String `tfsdk:"created_at"`
	UpdatedAt            types.String `tfsdk:"updated_at"`
}

func (d *sourceAppsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_source_apps"
}

// Configure adds the provider configured client to the data source.
func (d *sourceAppsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Add a nil check when handling ProviderData because Terraform
	// sets that data after it calls the ConfigureProvider RPC.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*traceforce.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *traceforce.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

// Schema defines the schema for the data source.
func (d *sourceAppsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"hosting_environment_id": schema.StringAttribute{
				Description: "Filter source apps by hosting environment ID. If not specified, returns all source apps.",
				Optional:    true,
			},
			"source_apps": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "System generated ID of the source app",
							Computed:    true,
						},
						"hosting_environment_id": schema.StringAttribute{
							Description: "ID of the hosting environment this source app belongs to",
							Computed:    true,
						},
						"type": schema.StringAttribute{
							Description: "Type of source app. For example, Salesforce, HubSpot, etc.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Name of the source app",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: fmt.Sprintf("Status of the source app. Valid values: %s, %s, %s, %s.",
								traceforce.SourceAppStatusPending,
								traceforce.SourceAppStatusDeployed,
								traceforce.SourceAppStatusDisconnected,
								traceforce.SourceAppStatusConnected),
							Computed: true,
						},
						"created_at": schema.StringAttribute{
							Description: "Date and time the source app was created",
							Computed:    true,
						},
						"updated_at": schema.StringAttribute{
							Description: "Date and time the source app was last updated",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *sourceAppsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config sourceAppsDataSourceModel

	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var sourceApps []traceforce.SourceApp
	var err error

	if !config.HostingEnvironmentId.IsNull() {
		// Get source apps filtered by hosting environment ID
		sourceApps, err = d.client.GetSourceAppsByHostingEnvironment(config.HostingEnvironmentId.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading source apps by hosting environment", err.Error())
			return
		}
	} else {
		// Get all source apps
		sourceApps, err = d.client.GetSourceApps()
		if err != nil {
			resp.Diagnostics.AddError("Error reading source apps", err.Error())
			return
		}
	}

	var state sourceAppsDataSourceModel
	state.HostingEnvironmentId = config.HostingEnvironmentId

	for _, sourceApp := range sourceApps {
		state.SourceApps = append(state.SourceApps, sourceAppsModel{
			ID:                   types.StringValue(sourceApp.ID),
			HostingEnvironmentId: types.StringValue(sourceApp.HostingEnvironmentID),
			Type:                 types.StringValue(string(sourceApp.Type)),
			Name:                 types.StringValue(sourceApp.Name),
			Status:               types.StringValue(string(sourceApp.Status)),
			CreatedAt:            types.StringValue(sourceApp.CreatedAt.Format(time.RFC3339)),
			UpdatedAt:            types.StringValue(sourceApp.UpdatedAt.Format(time.RFC3339)),
		})
	}

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
