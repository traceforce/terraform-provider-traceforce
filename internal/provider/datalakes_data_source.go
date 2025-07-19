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
	_ datasource.DataSource              = &datalakesDataSource{}
	_ datasource.DataSourceWithConfigure = &datalakesDataSource{}
)

func NewDatalakesDataSource() datasource.DataSource {
	return &datalakesDataSource{}
}

type datalakesDataSource struct {
	client *traceforce.Client
}

// datalakesDataSourceModel maps the data source schema data.
type datalakesDataSourceModel struct {
	ProjectId types.String        `tfsdk:"project_id"`
	Datalakes []datalakesModel    `tfsdk:"datalakes"`
}

// datalakesModel maps datalakes schema data.
type datalakesModel struct {
	ID        types.String `tfsdk:"id"`
	ProjectId types.String `tfsdk:"project_id"`
	Type      types.String `tfsdk:"type"`
	Name      types.String `tfsdk:"name"`
	Status    types.String `tfsdk:"status"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func (d *datalakesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_datalakes"
}

// Configure adds the provider configured client to the data source.
func (d *datalakesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *datalakesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Description: "Filter datalakes by project ID. If not specified, returns all datalakes.",
				Optional:    true,
			},
			"datalakes": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "System generated ID of the datalake",
							Computed:    true,
						},
						"project_id": schema.StringAttribute{
							Description: "ID of the project this datalake belongs to",
							Computed:    true,
						},
						"type": schema.StringAttribute{
							Description: "Type of datalake. For example, BigQuery, Snowflake, etc.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Name of the datalake",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "Status of the datalake. Valid values: Waiting for User Input, Ready.",
							Computed:    true,
						},
						"created_at": schema.StringAttribute{
							Description: "Date and time the datalake was created",
							Computed:    true,
						},
						"updated_at": schema.StringAttribute{
							Description: "Date and time the datalake was last updated",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *datalakesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config datalakesDataSourceModel

	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var datalakes []traceforce.Datalake
	var err error

	if !config.ProjectId.IsNull() {
		// Get datalakes filtered by hosting environment (project) ID
		datalakes, err = d.client.GetDatalakesByHostingEnvironment(config.ProjectId.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading datalakes by hosting environment", err.Error())
			return
		}
	} else {
		// Get all datalakes
		datalakes, err = d.client.GetDatalakes()
		if err != nil {
			resp.Diagnostics.AddError("Error reading datalakes", err.Error())
			return
		}
	}

	var state datalakesDataSourceModel
	state.ProjectId = config.ProjectId

	for _, datalake := range datalakes {
		state.Datalakes = append(state.Datalakes, datalakesModel{
			ID:        types.StringValue(datalake.ID),
			ProjectId: types.StringValue(datalake.HostingEnvironmentID),
			Type:      types.StringValue(string(datalake.Type)),
			Name:      types.StringValue(datalake.Name),
			Status:    types.StringValue(string(datalake.Status)),
			CreatedAt: types.StringValue(datalake.CreatedAt.Format(time.RFC3339)),
			UpdatedAt: types.StringValue(datalake.UpdatedAt.Format(time.RFC3339)),
		})
	}

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}