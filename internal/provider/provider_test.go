// Copyright (c) Traceforce, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const (
	// providerConfig is a shared configuration to combine with the actual
	// test configuration so the Traceforce client is properly configured.
	// The provider will use the TRACEFORCE_API_KEY environment variable for authentication.
	// To run acceptance tests, set the environment variable:
	//   export TRACEFORCE_API_KEY="your-api-key-here"
	providerConfig = `
provider "traceforce" {
  # api_key is configured via TRACEFORCE_API_KEY environment variable
}
`
)

var (
	// testAccProtoV6ProviderFactories are used to instantiate a provider during
	// acceptance testing. The factory function will be invoked for every Terraform
	// CLI command executed to create a provider server to which the CLI can
	// reattach.
	testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"traceforce": providerserver.NewProtocol6WithError(New("test")()),
	}
)

// testAccPreCheck validates that the necessary environment variables are set for acceptance tests.
// This function should be called in the PreCheck field of resource.TestCase.
func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("TRACEFORCE_API_KEY"); v == "" {
		t.Skip("TRACEFORCE_API_KEY environment variable must be set for acceptance tests. " +
			"Set it to run tests against the live API: export TRACEFORCE_API_KEY=\"your-api-key\"")
	}
}

// Login choice (api_key or client_id + client_secret): resolveLogin is a pure
// function, tested with a table; two end-to-end cases check that the chosen
// login reaches the API (fake_cloud_api_test.go).

func TestResolveLogin(t *testing.T) {
	set := types.StringValue
	null := types.StringNull()
	unknown := types.StringUnknown()
	cfg := func(apiKey, clientID, secret, device, endpoint types.String) traceforceProviderModel {
		return traceforceProviderModel{ApiKey: apiKey, ClientID: clientID, ClientSecret: secret, DeviceID: device, Endpoint: endpoint}
	}
	empty := cfg(null, null, null, null, null)
	for _, tc := range []struct {
		name   string
		config traceforceProviderModel
		env    map[string]string
		want   providerLogin
		errSum string // the error, or "" for none
	}{
		{"api_key in the configuration", cfg(set("k"), null, null, null, null), nil, providerLogin{apiKey: "k"}, ""},
		{"api_key in the configuration beats TRACEFORCE_CLIENT_ID", cfg(set("k"), null, null, null, null),
			map[string]string{"TRACEFORCE_CLIENT_ID": "c", "TRACEFORCE_CLIENT_SECRET": "s"}, providerLogin{apiKey: "k", clientSecret: "s"}, ""},
		{"client_id in the configuration beats TRACEFORCE_API_KEY", cfg(null, set("c"), set("s"), set("cloud-collector-aws"), null),
			map[string]string{"TRACEFORCE_API_KEY": "k"}, providerLogin{clientID: "c", clientSecret: "s", deviceID: "cloud-collector-aws"}, ""},
		{"client login from the environment", empty,
			map[string]string{"TRACEFORCE_CLIENT_ID": "c", "TRACEFORCE_CLIENT_SECRET": "s", "TRACEFORCE_DEVICE_ID": "cloud-collector-m365"},
			providerLogin{clientID: "c", clientSecret: "s", deviceID: "cloud-collector-m365"}, ""},
		{"api_key from the environment, as before", empty, map[string]string{"TRACEFORCE_API_KEY": "k"}, providerLogin{apiKey: "k"}, ""},
		{"secret, device and endpoint: configuration beats environment", cfg(null, set("c"), set("s1"), set("d1"), set("https://a")),
			map[string]string{"TRACEFORCE_CLIENT_SECRET": "s2", "TRACEFORCE_DEVICE_ID": "d2", "TRACEFORCE_ENDPOINT": "https://b"},
			providerLogin{clientID: "c", clientSecret: "s1", deviceID: "d1", endpoint: "https://a"}, ""},
		{"an empty configured value falls back to the environment", cfg(set(""), null, null, null, set("")),
			map[string]string{"TRACEFORCE_API_KEY": "k", "TRACEFORCE_ENDPOINT": "https://b"}, providerLogin{apiKey: "k", endpoint: "https://b"}, ""},
		{"both in the configuration", cfg(set("k"), set("c"), set("s"), null, null), nil, providerLogin{}, "Two Traceforce logins"},
		{"both in the environment: TRACEFORCE_API_KEY, as before (with a warning)", empty,
			map[string]string{"TRACEFORCE_API_KEY": "k", "TRACEFORCE_CLIENT_ID": "c", "TRACEFORCE_CLIENT_SECRET": "s"},
			providerLogin{apiKey: "k", clientSecret: "s"}, ""},
		{"no login", empty, nil, providerLogin{}, "Missing Traceforce API key"},
		{"client_id without a secret", cfg(null, set("c"), null, null, null), nil, providerLogin{}, "Missing Traceforce client secret"},
		{"unknown endpoint", cfg(set("k"), null, null, null, unknown), nil, providerLogin{}, "Unknown Traceforce API endpoint"},
		{"unknown api_key", cfg(unknown, null, null, null, null), nil, providerLogin{}, "Unknown Traceforce API key"},
		{"unknown client_id", cfg(null, unknown, set("s"), null, null), nil, providerLogin{}, "Unknown Traceforce client ID"},
		{"unknown client_secret", cfg(null, set("c"), unknown, null, null), nil, providerLogin{}, "Unknown Traceforce client secret"},
		{"unknown device_id", cfg(null, set("c"), set("s"), unknown, null), nil, providerLogin{}, "Unknown Traceforce device ID"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, diags := resolveLogin(tc.config, func(k string) string { return tc.env[k] })
			if tc.errSum != "" {
				if !diags.HasError() || diags.Errors()[0].Summary() != tc.errSum {
					t.Fatalf("want error %q, got %v", tc.errSum, diags)
				}
				return
			}
			if diags.HasError() {
				t.Fatalf("unexpected error: %v", diags)
			}
			if bothInEnv := tc.env["TRACEFORCE_API_KEY"] != "" && tc.env["TRACEFORCE_CLIENT_ID"] != "" && tc.config.ApiKey.ValueString() == "" && tc.config.ClientID.ValueString() == ""; bothInEnv != (diags.WarningsCount() == 1) {
				t.Fatalf("warning only when both logins are in the environment, got %v", diags)
			}
			if got != tc.want {
				t.Fatalf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

const loginResource = `
resource "traceforce_cloud_connection" "test" {
  provider_name  = "aws"
  account        = "111111111111"
  module_version = "1.2.0"
}
`

func TestAccProviderLogin(t *testing.T) {
	for _, tc := range []struct {
		name     string
		env      map[string]string
		settings string
		want     string // how the fake API was called
	}{
		{"a configured api_key is sent even with TRACEFORCE_CLIENT_ID exported",
			map[string]string{"TRACEFORCE_CLIENT_ID": "client-1", "TRACEFORCE_CLIENT_SECRET": "secret-1"},
			`api_key = "` + fakeAPIKey + `"`, "api_key"},
		{"both logins in the environment use TRACEFORCE_API_KEY, as before",
			map[string]string{"TRACEFORCE_API_KEY": fakeAPIKey, "TRACEFORCE_CLIENT_ID": "client-1", "TRACEFORCE_CLIENT_SECRET": "secret-1"},
			``, "api_key"},
		{"a client login from the environment logs in with its device id",
			map[string]string{"TRACEFORCE_CLIENT_ID": "client-1", "TRACEFORCE_CLIENT_SECRET": "secret-1", "TRACEFORCE_DEVICE_ID": "cloud-collector-m365"},
			``, "client cloud-collector-m365"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, v := range []string{"TRACEFORCE_API_KEY", "TRACEFORCE_CLIENT_ID", "TRACEFORCE_CLIENT_SECRET", "TRACEFORCE_DEVICE_ID"} {
				t.Setenv(v, tc.env[v])
			}
			f, endpoint := newFakeAPI(t)
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{{
					Config: fmt.Sprintf("provider \"traceforce\" {\n  endpoint = %q\n%s\n}\n", endpoint, tc.settings) + loginResource,
					Check: func(*terraform.State) error {
						if got := f.usedLogin(); got != tc.want {
							return fmt.Errorf("logged in with %s, want %s", got, tc.want)
						}
						return nil
					},
				}},
			})
		})
	}
}
