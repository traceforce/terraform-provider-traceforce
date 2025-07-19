// Copyright (c) Traceforce, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccProjectsDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: providerConfig + `data "traceforce_projects" "test" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify the first project to ensure all attributes are set
					resource.TestCheckResourceAttr("data.traceforce_projects.test", "projects.0.name", "test-project-1"),
					resource.TestCheckResourceAttr("data.traceforce_projects.test", "projects.0.type", "Customer Managed"),
					resource.TestCheckResourceAttr("data.traceforce_projects.test", "projects.0.cloud_provider", "GCP"),
					resource.TestCheckResourceAttr("data.traceforce_projects.test", "projects.0.native_id", "test-project-1"),
					resource.TestCheckResourceAttr("data.traceforce_projects.test", "projects.0.status", "Connected"),
					resource.TestCheckResourceAttrSet("data.traceforce_projects.test", "projects.0.control_plane_aws_account_id"),
					resource.TestCheckResourceAttrSet("data.traceforce_projects.test", "projects.0.control_plane_role_name"),
				),
			},
		},
	})
}
