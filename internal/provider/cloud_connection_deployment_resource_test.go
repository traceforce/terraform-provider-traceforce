// Copyright (c) Traceforce, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// These tests run against the fake Traceforce API of cloud_connection_resource_test.go.

const deployment = "traceforce_cloud_connection_deployment.test"

// deploymentConfig is the provider block, the connection and its deployment
// (extra: more deployment attributes, e.g. regions).
func deploymentConfig(endpoint, collectorID, extra string) string {
	return collectorConfig(endpoint, "1.2.0") + fmt.Sprintf(`
resource "traceforce_cloud_connection_deployment" "test" {
  connection_id = traceforce_cloud_connection.test.id
  collector_id  = %q
  identity_id   = "arn:aws:iam::111111111111:role/R"
%s
}
`, collectorID, extra)
}

const (
	lambdaA     = "arn:aws:lambda:us-east-1:111111111111:function:A"
	lambdaB     = "arn:aws:lambda:us-east-1:111111111111:function:B"
	usEast1Only = `  regions = ["us-east-1"]`
)

// expect checks what the fake API saw.
func (f *fakeAPI) expect(check func() error) resource.TestCheckFunc {
	return func(*terraform.State) error {
		f.mu.Lock()
		defer f.mu.Unlock()
		return check()
	}
}

func TestAccCloudConnectionDeploymentLifecycle(t *testing.T) {
	f, endpoint := newFakeAPI(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: deploymentConfig(endpoint, lambdaA, usEast1Only),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(deployment, "deployed_at", "2026-10-30T10:00:00Z"),
					resource.TestCheckResourceAttr(deployment, "regions.0", "us-east-1"),
					f.expect(func() error {
						if len(f.puts) != 1 || f.puts[0]["collectorId"] != lambdaA {
							return fmt.Errorf("PUT bodies %v", f.puts)
						}
						return nil
					}),
				),
			},
			{
				// The Lambda is replaced: recorded again.
				Config: deploymentConfig(endpoint, lambdaB, usEast1Only),
				Check: f.expect(func() error {
					if len(f.puts) != 2 || f.puts[1]["collectorId"] != lambdaB {
						return fmt.Errorf("PUT bodies %v", f.puts)
					}
					return nil
				}),
			},
			{
				// Changed in Traceforce (another workspace): the plan records ours again.
				PreConfig:          func() { f.set("collectorId", lambdaA) },
				Config:             deploymentConfig(endpoint, lambdaB, usEast1Only),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				// Cleared in Traceforce: the plan records it again.
				PreConfig:          func() { f.set("collectorId", lambdaB); f.set("deployedAt", nil) },
				Config:             deploymentConfig(endpoint, lambdaB, usEast1Only),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				// The deployment alone is removed from the configuration: it is cleared in Traceforce.
				PreConfig: func() { f.set("deployedAt", "2026-10-30T10:00:00Z") },
				Config:    collectorConfig(endpoint, "1.2.0"),
				Check: f.expect(func() error {
					if last := f.calls[len(f.calls)-1]; last != "DELETE deployment" || f.row["deployedAt"] != nil {
						return fmt.Errorf("calls %v, deployedAt %v", f.calls, f.row["deployedAt"])
					}
					return nil
				}),
			},
			{Config: deploymentConfig(endpoint, lambdaB, usEast1Only)},
		},
		// Destroy: the deployment is cleared, then the connection disconnected.
		CheckDestroy: func(*terraform.State) error {
			f.mu.Lock()
			defer f.mu.Unlock()
			if got := strings.Join(f.calls[len(f.calls)-2:], ", "); got != "DELETE deployment, DELETE connection" {
				return fmt.Errorf("destroy called %s", got)
			}
			return nil
		},
	})
}

func TestAccCloudConnectionDeploymentWithoutRegions(t *testing.T) {
	f, endpoint := newFakeAPI(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: deploymentConfig(endpoint, lambdaA, usEast1Only),
			},
			{
				// regions removed from the configuration: the stored ones are kept, no call.
				Config: deploymentConfig(endpoint, lambdaA, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(deployment, "regions.0", "us-east-1"),
					f.expect(func() error {
						if len(f.puts) != 1 {
							return fmt.Errorf("expected no new PUT, got %v", f.puts)
						}
						return nil
					}),
				),
			},
			{
				// A new Lambda without regions: the regions in the state are sent again, so they are kept.
				Config: deploymentConfig(endpoint, lambdaB, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(deployment, "regions.0", "us-east-1"),
					f.expect(func() error {
						if put := f.puts[len(f.puts)-1]; fmt.Sprint(put["regions"]) != "[us-east-1]" || put["collectorId"] != lambdaB {
							return fmt.Errorf("last PUT %v", put)
						}
						return nil
					}),
				),
			},
			{Config: deploymentConfig(endpoint, lambdaB, ""), PlanOnly: true}, // nothing left to plan
		},
	})
}

func TestAccCloudConnectionDeploymentEmptyRegions(t *testing.T) {
	_, endpoint := newFakeAPI(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: deploymentConfig(endpoint, lambdaA, `  regions = []`), ExpectError: regexp.MustCompile(`Empty regions`)},
		},
	})
}

func TestAccCloudConnectionDeploymentOldBackend(t *testing.T) {
	f, endpoint := newFakeAPI(t)
	f.oldBackend = true
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: deploymentConfig(endpoint, lambdaA, usEast1Only), ExpectError: regexp.MustCompile(`Update the Traceforce backend`)},
		},
	})
}

func TestAccCloudConnectionDeploymentNotRecorded(t *testing.T) {
	f, endpoint := newFakeAPI(t)
	f.noDeployedAtOn = true
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: deploymentConfig(endpoint, lambdaA, usEast1Only), ExpectError: regexp.MustCompile(`without\s+deployedAt`)},
		},
	})
}
