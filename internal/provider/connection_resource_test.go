// Copyright (c) Traceforce, Inc.
package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccConnectionResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
resource "traceforce_connection" "test" {
  name                  = "example"
  environment_type      = "AWS"
  environment_native_id = "9876543210"
  status                = "disconnected"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify number of items
					resource.TestCheckResourceAttr("traceforce_connection.test", "name", "example"),
					resource.TestCheckResourceAttr("traceforce_connection.test", "environment_type", "AWS"),
					resource.TestCheckResourceAttr("traceforce_connection.test", "environment_native_id", "9876543210"),
					resource.TestCheckResourceAttr("traceforce_connection.test", "status", "disconnected"),
					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("traceforce_connection.test", "id"),
					resource.TestCheckResourceAttrSet("traceforce_connection.test", "created_at"),
					resource.TestCheckResourceAttrSet("traceforce_connection.test", "updated_at"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "traceforce_connection.test",
				ImportState:       true,
				ImportStateVerify: true,
				// The last_updated attribute does not exist in the HashiCups
				// API, therefore there is no value for it during import.
				ImportStateVerifyIgnore: []string{"updated_at"},
			},
			// Update and Read testing
			{
				Config: providerConfig + `
resource "traceforce_connection" "test" {
  name                  = "example"
  environment_type      = "AWS"
  environment_native_id = "9876543210"
  status                = "connected"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify first order item updated
					resource.TestCheckResourceAttr("traceforce_connection.test", "status", "connected"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
