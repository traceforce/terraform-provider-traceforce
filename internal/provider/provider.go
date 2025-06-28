// Copyright (c) Traceforce, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"

	traceforce "github.com/traceforce/traceforce-go-sdk"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure traceforceProvider satisfies various provider interfaces.
var _ provider.Provider = &traceforceProvider{}
var _ provider.ProviderWithFunctions = &traceforceProvider{}
var _ provider.ProviderWithEphemeralResources = &traceforceProvider{}

// traceforceProvider defines the provider implementation.
type traceforceProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// traceforceProviderModel describes the provider data model.
type traceforceProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	ApiKey   types.String `tfsdk:"api_key"`
}

func (p *traceforceProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "traceforce"
	resp.Version = p.version
}

func (p *traceforceProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Description:         "API key to the service. May also be provided via TRACEFORCE_API_KEY environment variable.",
				MarkdownDescription: "API key to the service",
				Required:            true,
				Optional:            false,
			},
			"endpoint": schema.StringAttribute{
				Description:         "URI for Traceforce API. May also be provided via TRACEFORCE_ENDPOINT environment variable.",
				MarkdownDescription: "Service endpoint",
				Optional:            true,
			},
		},
	}
}

func (p *traceforceProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Retrieve provider data from configuration
	var config traceforceProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If practitioner provided a configuration value for any of the
	// attributes, it must be a known value.

	if config.Endpoint.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("endpoint"),
			"Unknown Traceforce API endpoint",
			"The provider cannot create the Traceforce API client as there is an unknown configuration value for the Traceforce API endpoint. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the TRACEFORCE_ENDPOINT environment variable.",
		)
	}

	if config.ApiKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("apiKey"),
			"Unknown Traceforce API key",
			"The provider cannot create the Traceforce API client as there is an unknown configuration value for the Traceforce API key. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the TRACEFORCE_API_KEY environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.

	endpoint := os.Getenv("TRACEFORCE_ENDPOINT")
	apiKey := os.Getenv("TRACEFORCE_API_KEY")

	if !config.Endpoint.IsNull() {
		endpoint = config.Endpoint.ValueString()
	}

	if !config.ApiKey.IsNull() {
		apiKey = config.ApiKey.ValueString()
	}

	// If any of the expected configurations are missing, return
	// errors with provider-specific guidance.

	if endpoint == "" {
		tflog.Info(ctx,
			"The provider will use the default Traceforce API endpoint. If you want to use a different endpoint, "+
				"Set the endpoint value in the configuration or use the TRACEFORCE_ENDPOINT environment variable.",
		)
	}

	if apiKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("apiKey"),
			"Missing Traceforce API key",
			"The provider cannot create the Traceforce API client as there is a missing or empty value for the Traceforce API key. "+
				"Set the apiKey value in the configuration or use the TRACEFORCE_API_KEY environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	client, err := traceforce.NewClient(apiKey, endpoint, nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Traceforce API Client",
			"An unexpected error occurred when creating the Traceforce API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Traceforce Client Error: "+err.Error(),
		)
		return
	}

	// Make the HashiCups client available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *traceforceProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewConnectionResource,
	}
}

func (p *traceforceProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{}
}

func (p *traceforceProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewConnectionsDataSource,
	}
}

func (p *traceforceProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &traceforceProvider{
			version: version,
		}
	}
}
