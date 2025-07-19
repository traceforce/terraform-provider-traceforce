// Copyright (c) Traceforce, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDatalakesDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing - all datalakes
			{
				Config: providerConfig + `data "traceforce_datalakes" "test" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify the first datalake to ensure all attributes are set
					resource.TestCheckResourceAttr("data.traceforce_datalakes.test", "datalakes.0.name", "production-warehouse"),
					resource.TestCheckResourceAttr("data.traceforce_datalakes.test", "datalakes.0.type", "BigQuery"),
					resource.TestCheckResourceAttr("data.traceforce_datalakes.test", "datalakes.0.project_id", "project-1"),
					resource.TestCheckResourceAttr("data.traceforce_datalakes.test", "datalakes.0.status", "Ready"),
					resource.TestCheckResourceAttrSet("data.traceforce_datalakes.test", "datalakes.0.id"),
					resource.TestCheckResourceAttrSet("data.traceforce_datalakes.test", "datalakes.0.created_at"),
					resource.TestCheckResourceAttrSet("data.traceforce_datalakes.test", "datalakes.0.updated_at"),
				),
			},
			// Read testing - filtered by project
			{
				Config: providerConfig + `
data "traceforce_datalakes" "project_filtered" {
  project_id = "project-1"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify filtering works
					resource.TestCheckResourceAttr("data.traceforce_datalakes.project_filtered", "project_id", "project-1"),
					// Should have at least one datalake for project-1
					resource.TestCheckResourceAttr("data.traceforce_datalakes.project_filtered", "datalakes.0.project_id", "project-1"),
				),
			},
		},
	})
}