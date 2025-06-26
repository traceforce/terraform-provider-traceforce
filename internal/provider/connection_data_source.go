package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	supabase "github.com/supabase-community/supabase-go"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &connectionsDataSource{}
	_ datasource.DataSourceWithConfigure = &connectionsDataSource{}
)

func NewConnectionsDataSource() datasource.DataSource {
	return &connectionsDataSource{}
}

type connectionsDataSource struct {
	client *supabase.Client
}

// connectionsDataSourceModel maps the data source schema data.
type connectionsDataSourceModel struct {
	Connections []connectionsModel `tfsdk:"connections"`
}

// connectionsModel maps connections schema data.
type connectionsModel struct {
	ID                  types.String `tfsdk:"id"`
	CreatedAt           types.String `tfsdk:"created_at"`
	UpdatedAt           types.String `tfsdk:"updated_at"`
	Name                types.String `tfsdk:"name"`
	EnvironmentType     types.String `tfsdk:"environment_type"`
	EnvironmentNativeId types.String `tfsdk:"environment_native_id"`
}

type connectionsOriginalModel struct {
	ID                  string    `json:"id"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
	Name                string    `json:"name"`
	EnvironmentType     string    `json:"environment_type"`
	EnvironmentNativeId string    `json:"environment_native_id"`
}

func (d *connectionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connections"
}

// Configure adds the provider configured client to the data source.
func (d *connectionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Add a nil check when handling ProviderData because Terraform
	// sets that data after it calls the ConfigureProvider RPC.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*supabase.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *supabase.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

// Schema defines the schema for the data source.
func (d *connectionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"connections": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed: true,
						},
						"created_at": schema.StringAttribute{
							Computed: true,
						},
						"updated_at": schema.StringAttribute{
							Computed: true,
						},
						"name": schema.StringAttribute{
							Computed: true,
						},
						"environment_type": schema.StringAttribute{
							Computed: true,
						},
						"environment_native_id": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (d *connectionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// Select all connections
	result, _, err := d.client.From("connections").Select("*", "", false).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error selecting connections", err.Error())
		return
	}

	var connections []connectionsOriginalModel
	err = json.Unmarshal(result, &connections)
	if err != nil {
		resp.Diagnostics.AddError("Error parsing connections", err.Error())
		return
	}

	var state connectionsDataSourceModel
	for _, connection := range connections {
		state.Connections = append(state.Connections, connectionsModel{
			ID:                  types.StringValue(connection.ID),
			CreatedAt:           types.StringValue(connection.CreatedAt.Format(time.RFC3339)),
			UpdatedAt:           types.StringValue(connection.UpdatedAt.Format(time.RFC3339)),
			Name:                types.StringValue(connection.Name),
			EnvironmentType:     types.StringValue(connection.EnvironmentType),
			EnvironmentNativeId: types.StringValue(connection.EnvironmentNativeId),
		})
	}

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
