// Copyright (c) Traceforce, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
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

// Login choice (api_key or client_id + client_secret), against the fake API of
// cloud_connection_resource_test.go. A value in the configuration beats the
// environment; both kinds at the same level is an error.

const loginResource = `
resource "traceforce_cloud_connection" "test" {
  provider_name  = "aws"
  account        = "111111111111"
  module_version = "1.2.0"
}
`

// clearLoginEnv starts a test with no TRACEFORCE_* login variables.
func clearLoginEnv(t *testing.T) {
	for _, v := range []string{"TRACEFORCE_API_KEY", "TRACEFORCE_CLIENT_ID", "TRACEFORCE_CLIENT_SECRET", "TRACEFORCE_DEVICE_ID"} {
		t.Setenv(v, "")
	}
}

func loginStep(endpoint, settings string) resource.TestStep {
	return resource.TestStep{Config: fmt.Sprintf("provider \"traceforce\" {\n  endpoint = %q\n%s\n}\n", endpoint, settings) + loginResource}
}

// usedLogin says how the fake API was called: "client <device id>" or "api_key".
func (f *fakeAPI) usedLogin() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.logins) > 0 && f.auths[0] == "Bearer tok" {
		return fmt.Sprintf("client %v", f.logins[0]["device_id"])
	}
	if len(f.auths) > 0 && f.auths[0] == "Bearer "+fakeAPIKey {
		return "api_key"
	}
	return "none"
}

func expectLogin(f *fakeAPI, want string) func() error {
	return func() error {
		if got := f.usedLogin(); got != want {
			return fmt.Errorf("logged in with %s, want %s", got, want)
		}
		return nil
	}
}

func TestAccProviderLogin(t *testing.T) {
	for _, tc := range []struct {
		name     string
		env      map[string]string
		settings string
		want     string // how the API was called, or the error
	}{
		{"api_key in the configuration beats TRACEFORCE_CLIENT_ID",
			map[string]string{"TRACEFORCE_CLIENT_ID": "client-1", "TRACEFORCE_CLIENT_SECRET": "secret-1"},
			`api_key = "` + fakeAPIKey + `"`, "api_key"},
		{"client_id in the configuration beats TRACEFORCE_API_KEY",
			map[string]string{"TRACEFORCE_API_KEY": fakeAPIKey},
			`client_id = "client-1"` + "\n" + `client_secret = "secret-1"` + "\n" + `device_id = "cloud-collector-aws"`, "client cloud-collector-aws"},
		{"client login from the environment",
			map[string]string{"TRACEFORCE_CLIENT_ID": "client-1", "TRACEFORCE_CLIENT_SECRET": "secret-1", "TRACEFORCE_DEVICE_ID": "cloud-collector-m365"},
			``, "client cloud-collector-m365"},
		{"device_id in the configuration beats TRACEFORCE_DEVICE_ID",
			map[string]string{"TRACEFORCE_DEVICE_ID": "cloud-collector-m365"},
			`client_id = "client-1"` + "\n" + `client_secret = "secret-1"` + "\n" + `device_id = "cloud-collector-aws"`, "client cloud-collector-aws"},
		{"api_key from the environment, as before", map[string]string{"TRACEFORCE_API_KEY": fakeAPIKey}, ``, "api_key"},
		{"both in the configuration", nil, `api_key = "k"` + "\n" + `client_id = "c"` + "\n" + `client_secret = "s"`, "error:Two Traceforce logins"},
		{"both in the environment", map[string]string{"TRACEFORCE_API_KEY": "k", "TRACEFORCE_CLIENT_ID": "c", "TRACEFORCE_CLIENT_SECRET": "s"}, ``, "error:Two Traceforce logins"},
		{"no login", nil, ``, "error:Missing Traceforce API key"},
		{"client_id without a secret", nil, `client_id = "c"`, "error:Missing Traceforce client secret"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clearLoginEnv(t)
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			f, endpoint := newFakeAPI(t)
			step := loginStep(endpoint, tc.settings)
			if msg, isErr := strings.CutPrefix(tc.want, "error:"); isErr {
				step.ExpectError = regexp.MustCompile(msg)
			} else {
				step.Check = func(*terraform.State) error { return expectLogin(f, tc.want)() }
			}
			resource.Test(t, resource.TestCase{ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, Steps: []resource.TestStep{step}})
		})
	}
}

func TestAccProviderUnknownLoginValue(t *testing.T) {
	clearLoginEnv(t)
	_, endpoint := newFakeAPI(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// The secret comes from a resource created in the same apply: unknown at plan time.
				Config: fmt.Sprintf(`
resource "terraform_data" "secret" {
  input = "secret-1"
}

provider "traceforce" {
  endpoint      = %q
  client_id     = "client-1"
  client_secret = terraform_data.secret.output
}
`, endpoint) + loginResource,
				ExpectError: regexp.MustCompile(`Unknown Traceforce client secret`),
			},
		},
	})
}
