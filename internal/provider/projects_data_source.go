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
	_ datasource.DataSource              = &projectsDataSource{}
	_ datasource.DataSourceWithConfigure = &projectsDataSource{}
)

func NewProjectsDataSource() datasource.DataSource {
	return &projectsDataSource{}
}

type projectsDataSource struct {
	client *traceforce.Client
}

// projectsDataSourceModel maps the data source schema data.
type projectsDataSourceModel struct {
	Projects []projectsModel `tfsdk:"projects"`
}

// projectsModel maps projects schema data.
type projectsModel struct {
	ID                       types.String `tfsdk:"id"`
	CreatedAt                types.String `tfsdk:"created_at"`
	UpdatedAt                types.String `tfsdk:"updated_at"`
	Name                     types.String `tfsdk:"name"`
	Type                     types.String `tfsdk:"type"`
	CloudProvider            types.String `tfsdk:"cloud_provider"`
	NativeId                 types.String `tfsdk:"native_id"`
	Status                   types.String `tfsdk:"status"`
	ControlPlaneAwsAccountId types.String `tfsdk:"control_plane_aws_account_id"`
	ControlPlaneRoleName     types.String `tfsdk:"control_plane_role_name"`
}

func (d *projectsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_projects"
}

// Configure adds the provider configured client to the data source.
func (d *projectsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *projectsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"projects": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "System generated ID of the project",
							Computed:    true,
						},
						"created_at": schema.StringAttribute{
							Description: "Date and time the project was created",
							Computed:    true,
						},
						"updated_at": schema.StringAttribute{
							Description: "Date and time the project was last updated",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Name of the project. This must be unique.",
							Computed:    true,
						},
						"type": schema.StringAttribute{
							Description: "Type of project. Valid values: Customer Managed, TraceForce Managed.",
							Computed:    true,
						},
						"cloud_provider": schema.StringAttribute{
							Description: "Cloud provider for the project. Valid values: AWS, GCP, Azure.",
							Computed:    true,
						},
						"native_id": schema.StringAttribute{
							Description: "Native ID of the cloud project. For example, an AWS account ID, an Azure subscription ID, a GCP project ID, etc.",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "Status of the project. Valid values: Pending, Disconnected, Connected.",
							Computed:    true,
						},
						"control_plane_aws_account_id": schema.StringAttribute{
							Description: "AWS account ID for the control plane",
							Computed:    true,
						},
						"control_plane_role_name": schema.StringAttribute{
							Description: "AWS IAM role name for the control plane",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *projectsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// Get all hosting environments (projects)
	hostingEnvironments, err := d.client.GetHostingEnvironments()
	if err != nil {
		resp.Diagnostics.AddError("Error reading hosting environments", err.Error())
		return
	}

	var state projectsDataSourceModel
	for _, env := range hostingEnvironments {
		state.Projects = append(state.Projects, projectsModel{
			ID:                       types.StringValue(env.ID),
			CreatedAt:                types.StringValue(env.CreatedAt.Format(time.RFC3339)),
			UpdatedAt:                types.StringValue(env.UpdatedAt.Format(time.RFC3339)),
			Name:                     types.StringValue(env.Name),
			Type:                     types.StringValue(string(env.Type)),
			CloudProvider:            types.StringValue(string(env.CloudProvider)),
			NativeId:                 types.StringValue(env.NativeID),
			Status:                   types.StringValue(string(env.Status)),
			ControlPlaneAwsAccountId: types.StringValue(env.ControlPlaneAwsAccountId),
			ControlPlaneRoleName:     types.StringValue(env.ControlPlaneRoleName),
		})
	}

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
