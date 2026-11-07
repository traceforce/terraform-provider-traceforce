// Copyright (c) Traceforce, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// These tests run against the fake Traceforce API (fake_cloud_api_test.go).

const deployment = "traceforce_cloud_connection_deployment.test"

const (
	lambdaA = "arn:aws:lambda:us-east-1:111111111111:function:A"
	lambdaB = "arn:aws:lambda:us-east-1:111111111111:function:B"
)

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

func regionsLine(regions ...string) string {
	return fmt.Sprintf("  regions = [%s]", `"`+strings.Join(regions, `", "`)+`"`)
}

// lastPut is the body of the newest PUT (call under the fake's lock).
func (f *fakeAPI) lastPut() map[string]any { return f.puts[len(f.puts)-1] }

func planAction(action plancheck.ResourceActionType) resource.ConfigPlanChecks {
	return resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(deployment, action)}}
}

func TestAccCloudConnectionDeploymentLifecycle(t *testing.T) {
	f, endpoint := newFakeAPI(t)
	east := regionsLine("us-east-1")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: deploymentConfig(endpoint, lambdaA, east),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(deployment, "deployed_at", "2026-10-30T10:00:00Z"),
					resource.TestCheckResourceAttr(deployment, "regions.#", "1"),
					f.expect(func() error {
						if len(f.puts) != 1 || f.puts[0]["collectorId"] != lambdaA || fmt.Sprint(f.puts[0]["regions"]) != "[us-east-1]" {
							return fmt.Errorf("PUT bodies %v", f.puts)
						}
						return nil
					}),
				),
			},
			{Config: deploymentConfig(endpoint, lambdaA, east), PlanOnly: true}, // nothing to do
			{
				// The Lambda is replaced: recorded again (in place).
				Config:           deploymentConfig(endpoint, lambdaB, east),
				ConfigPlanChecks: planAction(plancheck.ResourceActionUpdate),
				Check: f.expect(func() error {
					if len(f.puts) != 2 || f.lastPut()["collectorId"] != lambdaB {
						return fmt.Errorf("PUT bodies %v", f.puts)
					}
					return nil
				}),
			},
			{
				// The collector changed elsewhere: the plan records ours again.
				PreConfig:          func() { f.set("collectorId", lambdaA) },
				Config:             deploymentConfig(endpoint, lambdaB, east),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				// The identity changed elsewhere: too.
				PreConfig:          func() { f.set("collectorId", lambdaB); f.set("identityId", "arn:aws:iam::111111111111:role/Other") },
				Config:             deploymentConfig(endpoint, lambdaB, east),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				// Cleared in Traceforce: the next apply records it again.
				PreConfig: func() {
					f.set("identityId", "arn:aws:iam::111111111111:role/R")
					f.set("deployedAt", nil)
				},
				Config:           deploymentConfig(endpoint, lambdaB, east),
				ConfigPlanChecks: planAction(plancheck.ResourceActionCreate),
				Check:            resource.TestCheckResourceAttr(deployment, "deployed_at", "2026-10-30T10:00:00Z"),
			},
			{
				// The deployment alone is removed from the configuration: it is cleared in Traceforce.
				Config: collectorConfig(endpoint, "1.2.0"),
				Check: f.expect(func() error {
					if last := f.calls[len(f.calls)-1]; last != "DELETE deployment" || f.row["deployedAt"] != nil || f.row["collectorId"] != nil {
						return fmt.Errorf("calls %v, row %v", f.calls, f.row)
					}
					return nil
				}),
			},
			{Config: deploymentConfig(endpoint, lambdaB, east)},
		},
		// Destroy: the deployment is cleared, then the connection disconnected.
		CheckDestroy: f.expect(func() error {
			if got := strings.Join(f.calls[len(f.calls)-2:], ", "); got != "DELETE deployment, DELETE connection" {
				return fmt.Errorf("destroy called %s", got)
			}
			return nil
		}),
	})
}

func TestAccCloudConnectionDeploymentWithoutRegions(t *testing.T) {
	f, endpoint := newFakeAPI(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// First apply with no regions (every Microsoft 365 deployment): no regions sent.
				Config: deploymentConfig(endpoint, lambdaA, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr(deployment, "regions"),
					resource.TestCheckResourceAttr(deployment, "deployed_at", "2026-10-30T10:00:00Z"),
					f.expect(func() error {
						if _, sent := f.lastPut()["regions"]; sent {
							return fmt.Errorf("regions sent: %v", f.lastPut())
						}
						return nil
					}),
				),
			},
			{Config: deploymentConfig(endpoint, lambdaA, ""), PlanOnly: true}, // nothing to do
		},
	})
}

func TestAccCloudConnectionDeploymentRegionsRemoved(t *testing.T) {
	f, endpoint := newFakeAPI(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: deploymentConfig(endpoint, lambdaA, regionsLine("us-east-1"))},
			{
				// regions removed: one update with no regions; Traceforce keeps its regions.
				Config:           deploymentConfig(endpoint, lambdaA, ""),
				ConfigPlanChecks: planAction(plancheck.ResourceActionUpdate),
				Check: f.expect(func() error {
					if _, sent := f.lastPut()["regions"]; sent {
						return fmt.Errorf("regions sent: %v", f.lastPut())
					}
					return expectEqual(fmt.Sprint(f.row["regions"]), "[us-east-1]")
				}),
			},
			{Config: deploymentConfig(endpoint, lambdaA, ""), PlanOnly: true}, // then stable
			{
				// A collector upload changes the stored regions; a new Lambda (no regions set) does not send old ones.
				PreConfig: func() { f.set("regions", []string{"us-east-1", "eu-west-1"}) },
				Config:    deploymentConfig(endpoint, lambdaB, ""),
				Check: f.expect(func() error {
					if _, sent := f.lastPut()["regions"]; sent {
						return fmt.Errorf("regions sent: %v", f.lastPut())
					}
					return expectEqual(fmt.Sprint(f.row["regions"]), "[us-east-1 eu-west-1]")
				}),
			},
		},
	})
}

func TestAccCloudConnectionDeploymentRegionsOrder(t *testing.T) {
	f, endpoint := newFakeAPI(t)
	f.sortsRegions = true // the backend answers the regions in another order than sent
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: deploymentConfig(endpoint, lambdaA, regionsLine("us-west-2", "us-east-1")),
				Check:  resource.TestCheckResourceAttr(deployment, "regions.#", "2"),
			},
			// Another order in the configuration is not a change.
			{Config: deploymentConfig(endpoint, lambdaA, regionsLine("us-east-1", "us-west-2")), PlanOnly: true},
		},
	})
}

func TestAccCloudConnectionDeploymentNewConnectionReplaces(t *testing.T) {
	_, endpoint := newFakeAPI(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: deploymentConfig(endpoint, lambdaA, "")},
			{
				// Another connection id: the deployment is replaced (the fake has no such connection, so the new PUT fails).
				Config: collectorConfig(endpoint, "1.2.0") + fmt.Sprintf(`
resource "traceforce_cloud_connection_deployment" "test" {
  connection_id = %q
  collector_id  = %q
  identity_id   = "arn:aws:iam::111111111111:role/R"
}
`, otherConnID, lambdaA),
				ConfigPlanChecks: planAction(plancheck.ResourceActionDestroyBeforeCreate),
				ExpectError:      regexp.MustCompile(`Error recording cloud connection deployment`),
			},
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
			{Config: deploymentConfig(endpoint, lambdaA, ""), ExpectError: regexp.MustCompile(`Update the Traceforce\s+backend`)},
		},
	})
}

func TestAccCloudConnectionDeploymentNotRecorded(t *testing.T) {
	f, endpoint := newFakeAPI(t)
	f.putOmitsDeployedAt = true
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: deploymentConfig(endpoint, lambdaA, ""), ExpectError: regexp.MustCompile(`without\s+deployedAt`)},
		},
	})
}
