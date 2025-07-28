// Copyright (c) Traceforce, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSourceAppResource(t *testing.T) {
	// Use a resource name that starts with Z to ensure it is sorted last
	// Different Terraform versions may be triggered in parallel so
	// we need to ensure the resource name is unique.
	projectName := "z-project-" + uuid.New().String()
	sourceAppName := "z-sourceapp-" + uuid.New().String()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
resource "traceforce_project" "test" {
  name           = "` + projectName + `"
  type           = "Customer Managed"
  cloud_provider = "GCP"
  native_id      = "my-gcp-project"
}

resource "traceforce_source_app" "test" {
  hosting_environment_id = traceforce_project.test.id
  type                   = "Salesforce"
  name                   = "` + sourceAppName + `"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify attributes
					resource.TestCheckResourceAttr("traceforce_source_app.test", "type", "Salesforce"),
					resource.TestCheckResourceAttr("traceforce_source_app.test", "name", sourceAppName),
					resource.TestCheckResourceAttrSet("traceforce_source_app.test", "hosting_environment_id"),
					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("traceforce_source_app.test", "id"),
					resource.TestCheckResourceAttrSet("traceforce_source_app.test", "status"),
					resource.TestCheckResourceAttrSet("traceforce_source_app.test", "created_at"),
					resource.TestCheckResourceAttrSet("traceforce_source_app.test", "updated_at"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "traceforce_source_app.test",
				ImportState:       true,
				ImportStateVerify: true,
				// The updated_at attribute may not match during import
				ImportStateVerifyIgnore: []string{"updated_at"},
			},
			// Update and Read testing
			{
				Config: providerConfig + `
resource "traceforce_project" "test" {
  name           = "` + projectName + `"
  type           = "Customer Managed"
  cloud_provider = "GCP"
  native_id      = "my-gcp-project"
}

resource "traceforce_source_app" "test" {
  hosting_environment_id = traceforce_project.test.id
  type                   = "Salesforce"
  name                   = "` + sourceAppName + `-updated"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify name was updated
					resource.TestCheckResourceAttr("traceforce_source_app.test", "name", sourceAppName+"-updated"),
					resource.TestCheckResourceAttr("traceforce_source_app.test", "type", "Salesforce"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
