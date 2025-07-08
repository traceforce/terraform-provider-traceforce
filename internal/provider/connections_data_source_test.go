// Copyright (c) Traceforce, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccConnectionsDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: providerConfig + `data "traceforce_connections" "test" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify the first connection to ensure all attributes are set
					resource.TestCheckResourceAttr("data.traceforce_connections.test", "connections.0.name", "test-connection-1"),
					resource.TestCheckResourceAttr("data.traceforce_connections.test", "connections.0.environment_type", "GCP"),
					resource.TestCheckResourceAttr("data.traceforce_connections.test", "connections.0.environment_native_id", "test-project-1"),
					resource.TestCheckResourceAttr("data.traceforce_connections.test", "connections.0.status", "connected"),
				),
			},
		},
	})
}
