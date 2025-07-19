// Copyright (c) Traceforce, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSourceAppsDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing - all source apps
			{
				Config: providerConfig + `data "traceforce_source_apps" "test" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify the first source app to ensure all attributes are set
					resource.TestCheckResourceAttr("data.traceforce_source_apps.test", "source_apps.0.name", "sales-crm"),
					resource.TestCheckResourceAttr("data.traceforce_source_apps.test", "source_apps.0.type", "Salesforce"),
					resource.TestCheckResourceAttr("data.traceforce_source_apps.test", "source_apps.0.datalake_id", "datalake-1"),
					resource.TestCheckResourceAttr("data.traceforce_source_apps.test", "source_apps.0.status", "Connected"),
					resource.TestCheckResourceAttrSet("data.traceforce_source_apps.test", "source_apps.0.id"),
					resource.TestCheckResourceAttrSet("data.traceforce_source_apps.test", "source_apps.0.created_at"),
					resource.TestCheckResourceAttrSet("data.traceforce_source_apps.test", "source_apps.0.updated_at"),
				),
			},
			// Read testing - filtered by datalake
			{
				Config: providerConfig + `
data "traceforce_source_apps" "datalake_filtered" {
  datalake_id = "datalake-1"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify filtering works
					resource.TestCheckResourceAttr("data.traceforce_source_apps.datalake_filtered", "datalake_id", "datalake-1"),
					// Should have at least one source app for datalake-1
					resource.TestCheckResourceAttr("data.traceforce_source_apps.datalake_filtered", "source_apps.0.datalake_id", "datalake-1"),
				),
			},
		},
	})
}
