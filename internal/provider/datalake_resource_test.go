// Copyright (c) Traceforce, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDatalakeResource(t *testing.T) {
	// Use a resource name that starts with Z to ensure it is sorted last
	// Different Terraform versions may be triggered in parallel so
	// we need to ensure the resource name is unique.
	projectName := "z-project-" + uuid.New().String()
	datalakeName := "z-datalake-" + uuid.New().String()

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

resource "traceforce_datalake" "test" {
  project_id = traceforce_project.test.id
  type       = "BigQuery"
  name       = "` + datalakeName + `"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify attributes
					resource.TestCheckResourceAttr("traceforce_datalake.test", "type", "BigQuery"),
					resource.TestCheckResourceAttr("traceforce_datalake.test", "name", datalakeName),
					resource.TestCheckResourceAttrSet("traceforce_datalake.test", "project_id"),
					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("traceforce_datalake.test", "id"),
					resource.TestCheckResourceAttrSet("traceforce_datalake.test", "status"),
					resource.TestCheckResourceAttrSet("traceforce_datalake.test", "created_at"),
					resource.TestCheckResourceAttrSet("traceforce_datalake.test", "updated_at"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "traceforce_datalake.test",
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

resource "traceforce_datalake" "test" {
  project_id = traceforce_project.test.id
  type       = "BigQuery"
  name       = "` + datalakeName + `-updated"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify name was updated
					resource.TestCheckResourceAttr("traceforce_datalake.test", "name", datalakeName+"-updated"),
					resource.TestCheckResourceAttr("traceforce_datalake.test", "type", "BigQuery"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
