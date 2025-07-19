// Copyright (c) Traceforce, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
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
