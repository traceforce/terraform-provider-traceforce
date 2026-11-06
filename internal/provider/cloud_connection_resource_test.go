// Copyright (c) Traceforce, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// These tests run against a fake Traceforce API, so they need no API key.

const (
	fakeConnID  = "3b0c6d1e-1111-4222-8333-444455556666"
	otherConnID = "3b0c6d1e-9999-4222-8333-444455556666"
	fakeAPIKey  = "key-1" // the api_key the fake accepts
)

// fakeAPI stands in for the Traceforce cloud-connection routes (one connection).
type fakeAPI struct {
	mu       sync.Mutex
	logins   []map[string]any
	auths    []string       // Authorization of each cloud-connection call
	row      map[string]any // nil = no connection
	posts    []map[string]any
	deletes  int
	tooOld   bool   // answer 426 on register
	reissue  string // register answers this id instead (the row was replaced in Traceforce)
	existing bool   // the next register finds a row registered before (created: false)

	// Deployment (cloud_connection_deployment_resource_test.go).
	calls          []string // "PUT deployment", "DELETE deployment", "DELETE connection", in order
	puts           []map[string]any
	oldBackend     bool // no deployment routes: a plain 404 without the API's code
	noDeployedAtOn bool // PUT answers without deployedAt
}

func newFakeAPI(t *testing.T) (*fakeAPI, string) {
	f := &fakeAPI{}
	srv := httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(srv.Close)
	return f, srv.URL + "/api/v1"
}

func reply(w http.ResponseWriter, status int, body any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// set changes the stored connection, as Traceforce or another workspace would.
func (f *fakeAPI) set(key string, value any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.row[key] = value
}

func (f *fakeAPI) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)
	path := strings.TrimPrefix(r.URL.Path, "/api/v1")
	auth := r.Header.Get("Authorization")
	notFound := map[string]string{"error": "Connection not found", "code": "connection_not_found"}

	if r.Method == http.MethodPost && path == "/api-keys" {
		f.logins = append(f.logins, body)
		reply(w, 200, map[string]string{"access_token": "tok"})
		return
	}
	if auth != "Bearer tok" && auth != "Bearer "+fakeAPIKey {
		reply(w, 401, map[string]string{"error": "Authentication required"})
		return
	}
	f.auths = append(f.auths, auth)
	conn := "/cloud-connections/"
	if f.row != nil {
		id, _ := f.row["id"].(string)
		conn += id
	}

	switch {
	case r.Method == http.MethodPost && path == "/cloud-connections":
		f.posts = append(f.posts, body)
		if f.tooOld {
			reply(w, 426, map[string]string{"error": "Collector 1.0.0 is older than 1.2.0.", "code": "collector_too_old"})
			return
		}
		created := f.row == nil && !f.existing
		if f.row == nil || f.reissue != "" {
			id := fakeConnID
			if f.reissue != "" {
				id, f.reissue = f.reissue, ""
			}
			f.row = map[string]any{"id": id, "provider": body["provider"], "status": 1, "name": nil, "regions": []string{}}
		}
		f.existing = false
		account, _ := body["account"].(string)
		f.row["account"] = strings.ToLower(account)
		f.row["moduleVersion"], f.row["moduleCommit"] = body["moduleVersion"], body["moduleCommit"]
		if name, ok := body["name"]; ok {
			f.row["name"] = name
		}
		answer := map[string]any{"created": created}
		for k, v := range f.row {
			answer[k] = v
		}
		reply(w, 200, answer)
	case f.row == nil:
		reply(w, 404, notFound)
	case r.Method == http.MethodGet && path == conn:
		reply(w, 200, f.row)
	case f.oldBackend && strings.HasSuffix(path, "/deployment"):
		w.WriteHeader(404)
		_, _ = w.Write([]byte("<html>404: This page could not be found.</html>"))
	case r.Method == http.MethodPut && path == conn+"/deployment":
		f.calls, f.puts = append(f.calls, "PUT deployment"), append(f.puts, body)
		f.row["collectorId"], f.row["identityId"] = body["collectorId"], body["identityId"]
		if regions, ok := body["regions"]; ok {
			f.row["regions"] = regions
		}
		f.row["deployedAt"] = "2026-10-30T10:00:00Z"
		if f.noDeployedAtOn {
			f.row["deployedAt"] = nil
		}
		reply(w, 200, f.row)
	case r.Method == http.MethodDelete && path == conn+"/deployment":
		f.calls = append(f.calls, "DELETE deployment")
		f.row["collectorId"], f.row["identityId"], f.row["deployedAt"] = nil, nil, nil
		reply(w, 200, f.row)
	case r.Method == http.MethodDelete && path == conn:
		f.calls = append(f.calls, "DELETE connection")
		f.deletes++
		f.row = nil
		reply(w, 200, map[string]bool{"disconnected": true})
	default:
		reply(w, 404, notFound)
	}
}

// collectorProvider is the provider block the collector Terraform uses.
func collectorProvider(endpoint string) string {
	return fmt.Sprintf(`
provider "traceforce" {
  endpoint      = %q
  client_id     = "client-1"
  client_secret = "secret-1"
  device_id     = "cloud-collector-aws"
}
`, endpoint)
}

// connectionConfig is the provider block and one connection.
func connectionConfig(endpoint, cloud, account, version string) string {
	return collectorProvider(endpoint) + fmt.Sprintf(`
resource "traceforce_cloud_connection" "test" {
  provider_name  = %q
  account        = %q
  name           = "Test"
  module_version = %q
}
`, cloud, account, version)
}

func collectorConfig(endpoint, version string) string {
	return connectionConfig(endpoint, "aws", "111111111111", version)
}

const conn = "traceforce_cloud_connection.test"

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
					func(*terraform.State) error {
						f.mu.Lock()
						defer f.mu.Unlock()
						login := f.logins[0]
						if login["client_id"] != "client-1" || login["client_secret"] != "secret-1" || login["device_id"] != "cloud-collector-aws" {
							return fmt.Errorf("login sent %v", login)
						}
						return nil
					},
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
		CheckDestroy: func(*terraform.State) error {
			f.mu.Lock()
			defer f.mu.Unlock()
			if f.deletes != 1 {
				return fmt.Errorf("expected 1 DELETE, got %d", f.deletes)
			}
			return nil
		},
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
				PreConfig:          func() { f.mu.Lock(); f.row = nil; f.mu.Unlock() },
				Config:             collectorConfig(endpoint, "1.2.0"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
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

func TestAccCloudConnectionAccountCaseIsNotAChange(t *testing.T) {
	_, endpoint := newFakeAPI(t)
	tenant := "AAAAAAAA-0000-4000-8000-00000000000A" // Traceforce stores it in lowercase
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: connectionConfig(endpoint, "m365", tenant, "1.2.0"),
				Check:  resource.TestCheckResourceAttr(conn, "account", tenant),
			},
			{Config: connectionConfig(endpoint, "m365", tenant, "1.2.0"), PlanOnly: true}, // no replace planned
		},
	})
}

func TestAccCloudConnectionAlreadyRegistered(t *testing.T) {
	f, endpoint := newFakeAPI(t)
	f.existing = true // e.g. connected by an earlier collector: taken over with a warning
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: collectorConfig(endpoint, "1.2.0"), Check: resource.TestCheckResourceAttr(conn, "id", fakeConnID)},
		},
	})
}

func TestAccCloudConnectionReplacedDuringUpdate(t *testing.T) {
	f, endpoint := newFakeAPI(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: collectorConfig(endpoint, "1.2.0")},
			{
				// The row was replaced in Traceforce between refresh and apply.
				PreConfig:   func() { f.mu.Lock(); f.reissue = otherConnID; f.mu.Unlock() },
				Config:      collectorConfig(endpoint, "1.3.0"),
				ExpectError: regexp.MustCompile(`Cloud connection was replaced`),
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
			{Config: connectionConfig(endpoint, "gcp", "p", "1.2.0"), ExpectError: regexp.MustCompile(`provider_name must be one of aws, m365`)},
		},
	})
	if len(f.posts) != 0 {
		t.Fatalf("nothing should be sent, got %d registrations", len(f.posts))
	}
}
