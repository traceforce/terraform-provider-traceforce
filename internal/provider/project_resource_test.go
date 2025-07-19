// Copyright (c) Traceforce, Inc.
package provider

import (
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccProjectResource(t *testing.T) {
	// Use a resource name that starts with Z to ensure it is sorted last``
	// Different Terraform versions may be triggered in parallel so
	// we need to ensure the resource name is unique.
	resourceName := "z-example-" + uuid.New().String()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
resource "traceforce_project" "test" {
  name           = "` + resourceName + `"
  type           = "Customer Managed"
  cloud_provider = "AWS"
  native_id      = "9876543210"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify attributes
					resource.TestCheckResourceAttr("traceforce_project.test", "name", resourceName),
					resource.TestCheckResourceAttr("traceforce_project.test", "type", "Customer Managed"),
					resource.TestCheckResourceAttr("traceforce_project.test", "cloud_provider", "AWS"),
					resource.TestCheckResourceAttr("traceforce_project.test", "native_id", "9876543210"),
					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("traceforce_project.test", "id"),
					resource.TestCheckResourceAttrSet("traceforce_project.test", "status"),
					resource.TestCheckResourceAttrSet("traceforce_project.test", "control_plane_aws_account_id"),
					resource.TestCheckResourceAttrSet("traceforce_project.test", "control_plane_role_name"),
					resource.TestCheckResourceAttrSet("traceforce_project.test", "created_at"),
					resource.TestCheckResourceAttrSet("traceforce_project.test", "updated_at"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "traceforce_project.test",
				ImportState:       true,
				ImportStateId:     resourceName,
				ImportStateVerify: true,
				// The updated_at attribute may not match during import
				ImportStateVerifyIgnore: []string{"updated_at"},
			},
			// Update and Read testing
			{
				Config: providerConfig + `
resource "traceforce_project" "test" {
  name           = "` + resourceName + `"
  type           = "TraceForce Managed"
  cloud_provider = "GCP"
  native_id      = "my-gcp-project"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify updated attributes
					resource.TestCheckResourceAttr("traceforce_project.test", "type", "TraceForce Managed"),
					resource.TestCheckResourceAttr("traceforce_project.test", "cloud_provider", "GCP"),
					resource.TestCheckResourceAttr("traceforce_project.test", "native_id", "my-gcp-project"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
