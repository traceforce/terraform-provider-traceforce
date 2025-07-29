// Copyright (c) Traceforce, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSourceAppDatalakeLinkResource(t *testing.T) {
	// Use a resource name that starts with Z to ensure it is sorted last
	// Different Terraform versions may be triggered in parallel so
	// we need to ensure the resource name is unique.
	projectName := "z-project-" + uuid.New().String()
	datalakeName := "z-datalake-" + uuid.New().String()
	sourceAppName := "z-source-app-" + uuid.New().String()

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
  project_id            = traceforce_project.test.id
  type                  = "BigQuery"
  name                  = "` + datalakeName + `"
  environment_native_id = "test-project-id"
  region                = "us-central1"
}

resource "traceforce_source_app" "test" {
  project_id = traceforce_project.test.id
  type       = "Salesforce"
  name       = "` + sourceAppName + `"
}

resource "traceforce_source_app_datalake_link" "test" {
  source_app_id = traceforce_source_app.test.id
  datalake_id   = traceforce_datalake.test.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify attributes
					resource.TestCheckResourceAttrSet("traceforce_source_app_datalake_link.test", "source_app_id"),
					resource.TestCheckResourceAttrSet("traceforce_source_app_datalake_link.test", "datalake_id"),
					resource.TestCheckResourceAttrSet("traceforce_source_app_datalake_link.test", "hosting_environment_id"),
					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("traceforce_source_app_datalake_link.test", "id"),
					resource.TestCheckResourceAttrSet("traceforce_source_app_datalake_link.test", "created_at"),
					resource.TestCheckResourceAttrSet("traceforce_source_app_datalake_link.test", "updated_at"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "traceforce_source_app_datalake_link.test",
				ImportState:       true,
				ImportStateVerify: true,
				// The updated_at attribute may not match during import
				ImportStateVerifyIgnore: []string{"updated_at"},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccSourceAppDatalakeLinkResource_RequiresReplace(t *testing.T) {
	projectName1 := "z-project1-" + uuid.New().String()
	projectName2 := "z-project2-" + uuid.New().String()
	datalakeName1 := "z-datalake1-" + uuid.New().String()
	datalakeName2 := "z-datalake2-" + uuid.New().String()
	sourceAppName1 := "z-source-app1-" + uuid.New().String()
	sourceAppName2 := "z-source-app2-" + uuid.New().String()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create initial resources
			{
				Config: providerConfig + `
resource "traceforce_project" "test1" {
  name           = "` + projectName1 + `"
  type           = "Customer Managed"
  cloud_provider = "GCP"
  native_id      = "my-gcp-project-1"
}

resource "traceforce_datalake" "test1" {
  project_id            = traceforce_project.test1.id
  type                  = "BigQuery"
  name                  = "` + datalakeName1 + `"
  environment_native_id = "test-project-id-1"
  region                = "us-central1"
}

resource "traceforce_source_app" "test1" {
  project_id = traceforce_project.test1.id
  type       = "Salesforce"
  name       = "` + sourceAppName1 + `"
}

resource "traceforce_project" "test2" {
  name           = "` + projectName2 + `"
  type           = "Customer Managed"
  cloud_provider = "GCP"
  native_id      = "my-gcp-project-2"
}

resource "traceforce_datalake" "test2" {
  project_id            = traceforce_project.test2.id
  type                  = "BigQuery"
  name                  = "` + datalakeName2 + `"
  environment_native_id = "test-project-id-2"
  region                = "us-central1"
}

resource "traceforce_source_app" "test2" {
  project_id = traceforce_project.test2.id
  type       = "Salesforce"
  name       = "` + sourceAppName2 + `"
}

resource "traceforce_source_app_datalake_link" "test" {
  source_app_id = traceforce_source_app.test1.id
  datalake_id   = traceforce_datalake.test1.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("traceforce_source_app_datalake_link.test", "id"),
				),
			},
			// Verify changing source_app_id requires replacement
			{
				Config: providerConfig + `
resource "traceforce_project" "test1" {
  name           = "` + projectName1 + `"
  type           = "Customer Managed"
  cloud_provider = "GCP"
  native_id      = "my-gcp-project-1"
}

resource "traceforce_datalake" "test1" {
  project_id            = traceforce_project.test1.id
  type                  = "BigQuery"
  name                  = "` + datalakeName1 + `"
  environment_native_id = "test-project-id-1"
  region                = "us-central1"
}

resource "traceforce_source_app" "test1" {
  project_id = traceforce_project.test1.id
  type       = "Salesforce"
  name       = "` + sourceAppName1 + `"
}

resource "traceforce_project" "test2" {
  name           = "` + projectName2 + `"
  type           = "Customer Managed"
  cloud_provider = "GCP"
  native_id      = "my-gcp-project-2"
}

resource "traceforce_datalake" "test2" {
  project_id            = traceforce_project.test2.id
  type                  = "BigQuery"
  name                  = "` + datalakeName2 + `"
  environment_native_id = "test-project-id-2"
  region                = "us-central1"
}

resource "traceforce_source_app" "test2" {
  project_id = traceforce_project.test2.id
  type       = "Salesforce"
  name       = "` + sourceAppName2 + `"
}

resource "traceforce_source_app_datalake_link" "test" {
  source_app_id = traceforce_source_app.test2.id
  datalake_id   = traceforce_datalake.test1.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("traceforce_source_app_datalake_link.test", "id"),
				),
			},
		},
	})
}
