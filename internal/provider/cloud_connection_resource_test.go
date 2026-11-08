// Copyright (c) Traceforce, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	traceforce "github.com/traceforce/traceforce-go-sdk"
)

// These tests run against the fake Traceforce API (fake_cloud_api_test.go).

func TestAccCloudConnectionLifecycle(t *testing.T) {
	f, endpoint := newFakeAPI(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: collectorConfig(endpoint, "1.2.0"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(conn, "id", fakeConnID),
					resource.TestCheckResourceAttr(conn, "status", "pending"),
					resource.TestCheckResourceAttr(conn, "name", "Test"),
					resource.TestCheckNoResourceAttr(conn, "module_commit"),
					f.expect(func() error {
						login := f.logins[0]
						if login["client_id"] != "client-1" || login["client_secret"] != "secret-1" || login["device_id"] != "cloud-collector-aws" {
							return fmt.Errorf("login sent %v", login)
						}
						return nil
					}),
				),
			},
			{
				// Import by id fills every attribute from Traceforce.
				ResourceName:      conn,
				ImportState:       true,
				ImportStateId:     fakeConnID,
				ImportStateVerify: true,
			},
			{
				// New bundle: registered again, same connection (no replace).
				Config: collectorConfig(endpoint, "1.3.0"),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(conn, plancheck.ResourceActionUpdate),
				}},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(conn, "id", fakeConnID),
					resource.TestCheckResourceAttr(conn, "module_version", "1.3.0"),
				),
			},
			{
				// Status from Traceforce is refreshed.
				PreConfig: func() { f.set("status", 2) },
				Config:    collectorConfig(endpoint, "1.3.0"),
				Check:     resource.TestCheckResourceAttr(conn, "status", "connected"),
			},
		},
		// Destroy: the connection is disconnected once.
		CheckDestroy: f.expect(func() error {
			if f.deletes != 1 {
				return fmt.Errorf("expected 1 DELETE, got %d", f.deletes)
			}
			return nil
		}),
	})
}

func TestAccCloudConnectionGoneIsRecreated(t *testing.T) {
	f, endpoint := newFakeAPI(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: collectorConfig(endpoint, "1.2.0")},
			{
				// Disconnected in Settings: GET answers 404, so the next apply registers it again.
				PreConfig: func() { f.mu.Lock(); f.row = nil; f.mu.Unlock() },
				Config:    collectorConfig(endpoint, "1.2.0"),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(conn, plancheck.ResourceActionCreate),
				}},
				Check: resource.TestCheckResourceAttr(conn, "id", fakeConnID),
			},
		},
	})
}

func TestAccCloudConnectionDriftIsDetected(t *testing.T) {
	f, endpoint := newFakeAPI(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: collectorConfig(endpoint, "1.2.0")},
			{
				// Another workspace registered a different bundle: the plan registers ours again.
				PreConfig:          func() { f.set("moduleVersion", "1.9.0") },
				Config:             collectorConfig(endpoint, "1.2.0"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				// module_commit set in Traceforce but not in the configuration: cleared on the next apply.
				PreConfig:          func() { f.set("moduleVersion", "1.2.0"); f.set("moduleCommit", "abc") },
				Config:             collectorConfig(endpoint, "1.2.0"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccCloudConnectionImportedUppercaseAccount(t *testing.T) {
	f, endpoint := newFakeAPI(t)
	upper := "AAAAAAAA-0000-4000-8000-00000000000A"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// A connection that exists in Traceforce (lowercase), not yet in any state.
				PreConfig: func() {
					f.mu.Lock()
					f.row = map[string]any{"id": fakeConnID, "provider": "m365", "account": strings.ToLower(upper), "status": 2,
						"name": "Test", "moduleVersion": "1.2.0", "moduleCommit": nil, "regions": []string{}}
					f.mu.Unlock()
				},
				Config:             connectionConfig(endpoint, "m365", upper, "1.2.0", "Test"),
				ResourceName:       conn,
				ImportState:        true,
				ImportStateId:      fakeConnID,
				ImportStatePersist: true,
			},
			{
				// The configuration's uppercase tenant after the import: updated in place, not replaced.
				Config: connectionConfig(endpoint, "m365", upper, "1.2.0", "Test"),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(conn, plancheck.ResourceActionUpdate),
				}},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(conn, "id", fakeConnID),
					resource.TestCheckResourceAttr(conn, "account", upper),
					f.expect(func() error { return expectEqual(f.deletes, 0) }),
				),
			},
		},
	})
}

func TestAccCloudConnectionAccountCase(t *testing.T) {
	f, endpoint := newFakeAPI(t)
	upper := "AAAAAAAA-0000-4000-8000-00000000000A" // Traceforce stores it in lowercase
	lower := "aaaaaaaa-0000-4000-8000-00000000000a"
	noDisconnect := f.expect(func() error {
		if f.deletes != 0 {
			return fmt.Errorf("the connection was disconnected %d time(s)", f.deletes)
		}
		return nil
	})
	inPlace := resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
		plancheck.ExpectResourceAction(conn, plancheck.ResourceActionUpdate),
	}}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: connectionConfig(endpoint, "m365", upper, "1.2.0", "Test"),
				Check:  resource.TestCheckResourceAttr(conn, "account", upper),
			},
			{Config: connectionConfig(endpoint, "m365", upper, "1.2.0", "Test"), PlanOnly: true}, // nothing to do
			{
				// A case-only edit of the configuration: also in place.
				Config:           connectionConfig(endpoint, "m365", lower, "1.2.0", "Test"),
				ConfigPlanChecks: inPlace,
				Check:            resource.ComposeAggregateTestCheckFunc(resource.TestCheckResourceAttr(conn, "id", fakeConnID), noDisconnect),
			},
			{
				// Another account: replaced.
				Config: connectionConfig(endpoint, "m365", "bbbbbbbb-0000-4000-8000-00000000000b", "1.2.0", "Test"),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(conn, plancheck.ResourceActionDestroyBeforeCreate),
				}},
			},
		},
	})
}

func TestAccCloudConnectionNameRemoved(t *testing.T) {
	f, endpoint := newFakeAPI(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: collectorConfig(endpoint, "1.2.0")},
			{
				// The collector's upload replaced the name; then name is removed and the bundle changes.
				PreConfig: func() { f.set("name", "collector-name") },
				Config:    connectionConfig(endpoint, "aws", "111111111111", "1.3.0", ""),
				Check: f.expect(func() error {
					last := f.posts[len(f.posts)-1]
					if _, sent := last["name"]; sent {
						return fmt.Errorf("a removed name was sent again: %v", last)
					}
					return expectEqual(f.row["name"], "collector-name")
				}),
			},
			{
				// Set again: sent.
				Config: connectionConfig(endpoint, "aws", "111111111111", "1.3.0", "Prod"),
				Check:  f.expect(func() error { return expectEqual(f.row["name"], "Prod") }),
			},
		},
	})
}

func TestAccCloudConnectionAlreadyRegistered(t *testing.T) {
	f, endpoint := newFakeAPI(t)
	f.existing = true // e.g. connected by an earlier collector: taken over (with the takenOver warning)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: collectorConfig(endpoint, "1.2.0"), Check: resource.TestCheckResourceAttr(conn, "id", fakeConnID)},
		},
	})
}

func TestTakenOverWarnsOnlyForAnExistingConnection(t *testing.T) {
	existing := takenOver(&traceforce.CloudConnection{ID: fakeConnID, Provider: "aws", Account: "111111111111", Created: false})
	if existing.WarningsCount() != 1 || existing.Warnings()[0].Summary() != "Cloud connection already registered" {
		t.Fatalf("an existing connection should warn once, got %v", existing)
	}
	if created := takenOver(&traceforce.CloudConnection{ID: fakeConnID, Created: true}); len(created) != 0 {
		t.Fatalf("a new connection (or one that came back from disconnect) should not warn, got %v", created)
	}
}

func TestRegisterFailedMessages(t *testing.T) {
	var tooOld, other diag.Diagnostics
	registerFailed(&tooOld, "1.0.0", &traceforce.APIError{StatusCode: 426, Code: traceforce.CodeCollectorTooOld})
	registerFailed(&other, "1.0.0", errors.New("HTTP 500: boom"))
	if tooOld.Errors()[0].Summary() != "Collector bundle too old" || other.Errors()[0].Summary() != "Error registering cloud connection" {
		t.Fatalf("got %v / %v", tooOld, other)
	}
}

func TestAccCloudConnectionReplacedDuringUpdate(t *testing.T) {
	f, endpoint := newFakeAPI(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: collectorConfig(endpoint, "1.2.0")},
			{
				// The row was replaced in Traceforce between refresh and apply: an error, and the new id is saved.
				PreConfig:   func() { f.mu.Lock(); f.reissue = otherConnID; f.mu.Unlock() },
				Config:      collectorConfig(endpoint, "1.3.0"),
				ExpectError: regexp.MustCompile(`Cloud connection was registered again`),
			},
			{
				// The next apply finds the new connection: nothing to register (so no takeover warning).
				Config: collectorConfig(endpoint, "1.3.0"),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(conn, plancheck.ResourceActionNoop),
				}},
				Check: resource.TestCheckResourceAttr(conn, "id", otherConnID),
			},
		},
	})
}

func TestAccCloudConnectionOldBundle(t *testing.T) {
	f, endpoint := newFakeAPI(t)
	f.tooOld = true
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: collectorConfig(endpoint, "1.0.0"), ExpectError: regexp.MustCompile(`Collector bundle too old`)},
		},
	})
}

func TestAccCloudConnectionOldBundleOnUpdate(t *testing.T) {
	f, endpoint := newFakeAPI(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: collectorConfig(endpoint, "1.2.0")},
			{
				PreConfig:   func() { f.mu.Lock(); f.tooOld = true; f.mu.Unlock() },
				Config:      collectorConfig(endpoint, "1.0.0"),
				ExpectError: regexp.MustCompile(`Collector bundle too old`),
			},
		},
	})
}

func TestAccCloudConnectionUnknownCloud(t *testing.T) {
	f, endpoint := newFakeAPI(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: connectionConfig(endpoint, "gcp", "p", "1.2.0", "Test"), ExpectError: regexp.MustCompile(`provider_name must be one of aws, m365`)},
		},
	})
	if len(f.posts) != 0 {
		t.Fatalf("nothing should be sent, got %d registrations", len(f.posts))
	}
}

func expectEqual(got, want any) error {
	if got != want {
		return fmt.Errorf("got %v, want %v", got, want)
	}
	return nil
}

func TestAccCloudConnectionNameIsNotRefreshed(t *testing.T) {
	f, endpoint := newFakeAPI(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: collectorConfig(endpoint, "1.2.0")},
			{
				// The collector's upload renamed the connection: not a change for this configuration.
				PreConfig: func() { f.set("name", "collector-name") },
				Config:    collectorConfig(endpoint, "1.2.0"),
				PlanOnly:  true,
			},
		},
	})
}

func TestAccCloudConnectionEmptyModuleCommit(t *testing.T) {
	f, endpoint := newFakeAPI(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: collectorProvider(endpoint) + `
resource "traceforce_cloud_connection" "test" {
  provider_name  = "aws"
  account        = "111111111111"
  module_version = "1.2.0"
  module_commit  = ""
}
`,
				ExpectError: regexp.MustCompile(`Empty module_commit`),
			},
		},
	})
	if len(f.posts) != 0 {
		t.Fatalf("nothing should be sent, got %d registrations", len(f.posts))
	}
}
