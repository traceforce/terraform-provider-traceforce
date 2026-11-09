// Copyright (c) Traceforce, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"

	traceforce "github.com/traceforce/traceforce-go-sdk"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	Endpoint     types.String `tfsdk:"endpoint"`
	ApiKey       types.String `tfsdk:"api_key"`
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
	DeviceID     types.String `tfsdk:"device_id"`
	ExtraHeaders types.Map    `tfsdk:"extra_headers"`
}

func (p *traceforceProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "traceforce"
	resp.Version = p.version
}

func (p *traceforceProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		// The Authentication section of the docs is in templates/index.md.tmpl.
		MarkdownDescription: "Manages Traceforce resources.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Description:         "API key to the service. May also be provided via TRACEFORCE_API_KEY environment variable.",
				MarkdownDescription: "API key to the service. Or `TRACEFORCE_API_KEY`.",
				Optional:            true,
			},
			"client_id": schema.StringAttribute{
				Description:         "Client ID of one of your Traceforce API clients (Settings → API Clients), used instead of api_key. May also be provided via TRACEFORCE_CLIENT_ID environment variable.",
				MarkdownDescription: "Client ID of an API client, used instead of `api_key`. Or `TRACEFORCE_CLIENT_ID`.",
				Optional:            true,
			},
			"client_secret": schema.StringAttribute{
				Description:         "Client secret of that API client. May also be provided via TRACEFORCE_CLIENT_SECRET environment variable.",
				MarkdownDescription: "Client secret of that API client. Or `TRACEFORCE_CLIENT_SECRET`.",
				Optional:            true,
				Sensitive:           true,
			},
			"device_id": schema.StringAttribute{
				Description:         "Who the token is for when logging in with client_id, e.g. cloud-collector-aws. May also be provided via TRACEFORCE_DEVICE_ID environment variable.",
				MarkdownDescription: "Who the token is for when logging in with `client_id`, e.g. `cloud-collector-aws`. Or `TRACEFORCE_DEVICE_ID`.",
				Optional:            true,
			},
			"endpoint": schema.StringAttribute{
				Description:         "URI for Traceforce API. May also be provided via TRACEFORCE_ENDPOINT environment variable.",
				MarkdownDescription: "Service endpoint. Or `TRACEFORCE_ENDPOINT`.",
				Optional:            true,
			},
			"extra_headers": schema.MapAttribute{
				Description:         "Additional headers to include in API requests.",
				MarkdownDescription: "Additional headers to include in API requests",
				Optional:            true,
				ElementType:         types.StringType,
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

	login, diags := resolveLogin(config, os.Getenv)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if login.endpoint == "" {
		tflog.Info(ctx,
			"The provider will use the default Traceforce API endpoint. If you want to use a different endpoint, "+
				"Set the endpoint value in the configuration or use the TRACEFORCE_ENDPOINT environment variable.",
		)
	}

	// Handle extra headers
	var extraHeaders map[string]string
	if !config.ExtraHeaders.IsNull() && !config.ExtraHeaders.IsUnknown() {
		extraHeaders = make(map[string]string)
		for k, v := range config.ExtraHeaders.Elements() {
			if strVal, ok := v.(types.String); ok && !strVal.IsNull() {
				extraHeaders[k] = strVal.ValueString()
			}
		}
	}

	options := &traceforce.ClientOptions{
		ExtraHeaders: extraHeaders,
	}

	// API client login when client_id is given, else the api_key as before.
	var client *traceforce.Client
	var err error
	if login.clientID != "" {
		client, err = traceforce.NewClientWithCredentials(
			traceforce.Credentials{ClientID: login.clientID, ClientSecret: login.clientSecret, DeviceID: login.deviceID}, login.endpoint, options)
	} else {
		client, err = traceforce.NewClient(login.apiKey, login.endpoint, options)
	}
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

// providerLogin is how the provider logs in: endpoint, and api_key or an API
// client (client_id, client_secret, device_id).
type providerLogin struct {
	endpoint, apiKey, clientID, clientSecret, deviceID string
}

// resolveLogin takes each setting from the configuration, or else from its
// environment variable (getenv), and checks the result:
//   - a configured value that is still unknown (e.g. made in the same apply) is an error;
//   - api_key and client_id are two ways to log in; the one set in the
//     configuration wins over the environment, and both at the same level is an error;
//   - client_id needs client_secret; device_id is used with client_id only.
func resolveLogin(config traceforceProviderModel, getenv func(string) string) (providerLogin, diag.Diagnostics) {
	var diags diag.Diagnostics
	for _, setting := range []struct {
		value     types.String
		attribute string
		label     string
		envVar    string
	}{
		{config.Endpoint, "endpoint", "API endpoint", "TRACEFORCE_ENDPOINT"},
		{config.ApiKey, "api_key", "API key", "TRACEFORCE_API_KEY"},
		{config.ClientID, "client_id", "client ID", "TRACEFORCE_CLIENT_ID"},
		{config.ClientSecret, "client_secret", "client secret", "TRACEFORCE_CLIENT_SECRET"},
		{config.DeviceID, "device_id", "device ID", "TRACEFORCE_DEVICE_ID"},
	} {
		if setting.value.IsUnknown() {
			diags.AddAttributeError(
				path.Root(setting.attribute),
				"Unknown Traceforce "+setting.label,
				"The provider cannot create the Traceforce API client as there is an unknown configuration value for the Traceforce "+setting.label+". "+
					"Either target apply the source of the value first, set the value statically in the configuration, or use the "+setting.envVar+" environment variable.",
			)
		}
	}
	if diags.HasError() {
		return providerLogin{}, diags
	}

	configured := func(v types.String, envVar string) string {
		if v.ValueString() != "" {
			return v.ValueString()
		}
		return getenv(envVar)
	}
	login := providerLogin{
		endpoint:     configured(config.Endpoint, "TRACEFORCE_ENDPOINT"),
		apiKey:       config.ApiKey.ValueString(),
		clientID:     config.ClientID.ValueString(),
		clientSecret: configured(config.ClientSecret, "TRACEFORCE_CLIENT_SECRET"),
		deviceID:     configured(config.DeviceID, "TRACEFORCE_DEVICE_ID"),
	}
	if login.apiKey == "" && login.clientID == "" {
		login.apiKey, login.clientID = getenv("TRACEFORCE_API_KEY"), getenv("TRACEFORCE_CLIENT_ID")
		// Both in the environment (e.g. a CI runner that also runs the cloud
		// collector): TRACEFORCE_API_KEY, as before client logins existed.
		if login.apiKey != "" && login.clientID != "" {
			login.clientID = ""
			diags.AddWarning("Two Traceforce logins in the environment",
				"Both TRACEFORCE_API_KEY and TRACEFORCE_CLIENT_ID are set; the provider uses TRACEFORCE_API_KEY. "+
					"traceforce_cloud_connection needs the API client login: set client_id in the provider block or unset TRACEFORCE_API_KEY.")
		}
	}

	switch {
	case login.apiKey != "" && login.clientID != "":
		diags.AddAttributeError(
			path.Root("client_id"),
			"Two Traceforce logins",
			"Both api_key and client_id are set in the provider block. Set only one of them.",
		)
	case login.apiKey == "" && login.clientID == "":
		diags.AddAttributeError(
			path.Root("api_key"),
			"Missing Traceforce API key",
			"The provider cannot create the Traceforce API client as there is a missing or empty value for the Traceforce API key. "+
				"Set the api_key value in the configuration or use the TRACEFORCE_API_KEY environment variable, "+
				"or log in with client_id and client_secret (TRACEFORCE_CLIENT_ID, TRACEFORCE_CLIENT_SECRET). "+
				"If either is already set, ensure the value is not empty.",
		)
	case login.clientID != "" && login.clientSecret == "":
		diags.AddAttributeError(
			path.Root("client_secret"),
			"Missing Traceforce client secret",
			"client_id is set, so client_secret (or the TRACEFORCE_CLIENT_SECRET environment variable) is needed too.",
		)
	}
	return login, diags
}

func (p *traceforceProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewProjectResource,
		NewDatalakeResource,
		NewSourceAppResource,
		NewPostConnectionResource,
		NewSourceAppDatalakeLinkResource,
		NewCloudConnectionResource,
	}
}

func (p *traceforceProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{}
}

func (p *traceforceProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewProjectsDataSource,
		NewDatalakesDataSource,
		NewSourceAppsDataSource,
	}
}

func (p *traceforceProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

// clientFrom gives a resource the logged-in API client that the provider's
// Configure made (resp.ResourceData = client, above). Terraform hands it to each
// resource's Configure as req.ProviderData. It is nil on the first call, before
// the provider is configured; anything other than a *traceforce.Client is an error.
// Used by the cloud-connection resources (the older resources keep their own copy).
func clientFrom(data any, diags *diag.Diagnostics) *traceforce.Client {
	if data == nil {
		return nil
	}
	client, ok := data.(*traceforce.Client)
	if !ok {
		diags.AddError("Unexpected Provider Configuration",
			fmt.Sprintf("Expected *traceforce.Client, got: %T. Please report this issue to the provider developers.", data))
	}
	return client
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &traceforceProvider{
			version: version,
		}
	}
}
