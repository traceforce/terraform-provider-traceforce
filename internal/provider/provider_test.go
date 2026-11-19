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
	traceforce "github.com/traceforce/traceforce-go-sdk"
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
	cfg := func(apiKey, clientID, secret, endpoint types.String) traceforceProviderModel {
		return traceforceProviderModel{ApiKey: apiKey, ClientID: clientID, ClientSecret: secret, Endpoint: endpoint}
	}
	empty := cfg(null, null, null, null)
	for _, tc := range []struct {
		name   string
		config traceforceProviderModel
		env    map[string]string
		want   providerLogin
		errSum string // the error, or "" for none
		warn   bool   // the "Two Traceforce logins in the environment" warning
	}{
		{"api_key in the configuration", cfg(set("k"), null, null, null), nil, providerLogin{apiKey: "k"}, "", false},
		{"api_key in the configuration beats TRACEFORCE_CLIENT_ID", cfg(set("k"), null, null, null),
			map[string]string{"TRACEFORCE_CLIENT_ID": "c", "TRACEFORCE_CLIENT_SECRET": "s"}, providerLogin{apiKey: "k", clientSecret: "s"}, "", false},
		{"client_id in the configuration beats TRACEFORCE_API_KEY", cfg(null, set("c"), set("s"), null),
			map[string]string{"TRACEFORCE_API_KEY": "k"}, providerLogin{clientID: "c", clientSecret: "s"}, "", false},
		{"client login from the environment", empty,
			map[string]string{"TRACEFORCE_CLIENT_ID": "c", "TRACEFORCE_CLIENT_SECRET": "s"},
			providerLogin{clientID: "c", clientSecret: "s"}, "", false},
		{"api_key from the environment, as before", empty, map[string]string{"TRACEFORCE_API_KEY": "k"}, providerLogin{apiKey: "k"}, "", false},
		{"secret and endpoint: configuration beats environment", cfg(null, set("c"), set("s1"), set("https://a")),
			map[string]string{"TRACEFORCE_CLIENT_SECRET": "s2", "TRACEFORCE_ENDPOINT": "https://b"},
			providerLogin{clientID: "c", clientSecret: "s1", endpoint: "https://a"}, "", false},
		{"an empty configured value falls back to the environment", cfg(set(""), null, null, set("")),
			map[string]string{"TRACEFORCE_API_KEY": "k", "TRACEFORCE_ENDPOINT": "https://b"}, providerLogin{apiKey: "k", endpoint: "https://b"}, "", false},
		{"both in the configuration", cfg(set("k"), set("c"), set("s"), null), nil, providerLogin{}, "Two Traceforce logins", false},
		{"both in the environment: TRACEFORCE_API_KEY, as before (with a warning)", empty,
			map[string]string{"TRACEFORCE_API_KEY": "k", "TRACEFORCE_CLIENT_ID": "c", "TRACEFORCE_CLIENT_SECRET": "s"},
			providerLogin{apiKey: "k", clientSecret: "s"}, "", true},
		{"no login", empty, nil, providerLogin{}, "Missing Traceforce API key", false},
		{"client_id without a secret", cfg(null, set("c"), null, null), nil, providerLogin{}, "Missing Traceforce client secret", false},
		{"unknown endpoint", cfg(set("k"), null, null, unknown), nil, providerLogin{}, "Unknown Traceforce API endpoint", false},
		{"unknown api_key", cfg(unknown, null, null, null), nil, providerLogin{}, "Unknown Traceforce API key", false},
		{"unknown client_id", cfg(null, unknown, set("s"), null), nil, providerLogin{}, "Unknown Traceforce client ID", false},
		{"unknown client_secret", cfg(null, set("c"), unknown, null), nil, providerLogin{}, "Unknown Traceforce client secret", false},
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
			if warned := diags.WarningsCount() > 0; warned != tc.warn || diags.WarningsCount() > 1 {
				t.Fatalf("want warning=%v, got %v", tc.warn, diags)
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
		{"a client login from the environment logs in as the connection's cloud collector",
			map[string]string{"TRACEFORCE_CLIENT_ID": "client-1", "TRACEFORCE_CLIENT_SECRET": "secret-1"},
			``, "client cloud-collector-aws"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, v := range []string{"TRACEFORCE_API_KEY", "TRACEFORCE_CLIENT_ID", "TRACEFORCE_CLIENT_SECRET"} {
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

// An M365 connection logs in as the M365 collector, with the same provider block.
func TestAccProviderLoginUsesTheConnectionsCloud(t *testing.T) {
	f, endpoint := newFakeAPI(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: connectionConfig(endpoint, "m365", "00000000-0000-4000-8000-0000000000aa", "1.2.0", "Test"),
			Check: func(*terraform.State) error {
				if got := f.usedLogin(); got != "client cloud-collector-m365" {
					return fmt.Errorf("logged in with %s, want client cloud-collector-m365", got)
				}
				return nil
			},
		}},
	})
}

// cloudClients: one login per cloud, kept; an api_key client serves every cloud.
func TestCloudClients(t *testing.T) {
	f, endpoint := newFakeAPI(t)
	creds := traceforce.Credentials{ClientID: "client-1", ClientSecret: "secret-1"}
	plain, err := traceforce.NewClientWithCredentials(creds, endpoint, nil)
	if err != nil {
		t.Fatal(err)
	}
	clients := &cloudClients{}
	if _, err := clients.forCloud("aws"); err == nil {
		t.Fatal("want an error before the provider is configured")
	}
	clients.configure(plain, creds, endpoint, nil)
	aws, err := clients.forCloud("aws")
	again, _ := clients.forCloud("aws")
	m365, err2 := clients.forCloud("m365")
	if err != nil || err2 != nil || aws != again || aws == m365 || aws == plain {
		t.Fatalf("want one kept client per cloud, got aws=%p again=%p m365=%p plain=%p (%v, %v)", aws, again, m365, plain, err, err2)
	}
	if _, err := clients.forCloud(""); err == nil {
		t.Fatal("want an error for an unknown cloud")
	}
	if len(f.logins) != 3 || f.logins[1]["device_id"] != "cloud-collector-aws" || f.logins[2]["device_id"] != "cloud-collector-m365" {
		t.Fatalf("want 3 logins (plain, aws, m365), got %v", f.logins)
	}

	keyClient, _ := traceforce.NewClient(fakeAPIKey, endpoint, nil)
	clients.configure(keyClient, traceforce.Credentials{}, endpoint, nil)
	if c, err := clients.forCloud("aws"); err != nil || c != keyClient {
		t.Fatalf("an api_key login uses its one client, got %p (%v)", c, err)
	}
}
