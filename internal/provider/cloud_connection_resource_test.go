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

const fakeConnID = "3b0c6d1e-1111-4222-8333-444455556666"

// fakeAPI stands in for the Traceforce cloud-connection routes (one connection).
type fakeAPI struct {
	mu      sync.Mutex
	logins  []map[string]any
	row     map[string]any // nil = no connection
	deletes int
	tooOld  bool // answer 426 on register
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

func (f *fakeAPI) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)
	path := strings.TrimPrefix(r.URL.Path, "/api/v1")
	conn := "/cloud-connections/" + fakeConnID

	switch {
	case r.Method == http.MethodPost && path == "/api-keys":
		f.logins = append(f.logins, body)
		reply(w, 200, map[string]string{"access_token": "tok"})
	case r.Header.Get("Authorization") != "Bearer tok":
		reply(w, 401, map[string]string{"error": "Authentication required"})
	case r.Method == http.MethodPost && path == "/cloud-connections":
		if f.tooOld {
			reply(w, 426, map[string]string{"error": "Collector 1.0.0 is older than 1.2.0."})
			return
		}
		if f.row == nil {
			f.row = map[string]any{"id": fakeConnID, "status": 1}
		}
		f.row["account_ref"], f.row["module_version"] = body["account"], body["moduleVersion"]
		reply(w, 200, map[string]string{"id": fakeConnID})
	case f.row == nil:
		reply(w, 404, map[string]string{"error": "Connection not found"})
	case r.Method == http.MethodGet && path == conn:
		reply(w, 200, f.row)
	case r.Method == http.MethodDelete && path == conn:
		f.deletes++
		f.row = nil
		reply(w, 200, map[string]bool{"disconnected": true})
	default:
		reply(w, 404, map[string]string{"error": "not found"})
	}
}

// collectorConfig is the provider block and the connection, as the collector Terraform uses them.
func collectorConfig(endpoint, version string) string {
	return fmt.Sprintf(`
provider "traceforce" {
  endpoint      = %q
  client_id     = "client-1"
  client_secret = "secret-1"
  device_id     = "cloud-collector-aws"
}

resource "traceforce_cloud_connection" "test" {
  provider_name  = "aws"
  account        = "111111111111"
  name           = "Test"
  module_version = %q
}
`, endpoint, version)
}

func TestAccCloudConnectionLifecycle(t *testing.T) {
	f, endpoint := newFakeAPI(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: collectorConfig(endpoint, "1.2.0"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("traceforce_cloud_connection.test", "id", fakeConnID),
					resource.TestCheckResourceAttr("traceforce_cloud_connection.test", "status", "1"),
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
				// New bundle: registered again, same connection (no replace).
				Config: collectorConfig(endpoint, "1.3.0"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("traceforce_cloud_connection.test", "id", fakeConnID),
					resource.TestCheckResourceAttr("traceforce_cloud_connection.test", "module_version", "1.3.0"),
				),
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

func TestAccCloudConnectionOldBundle(t *testing.T) {
	f, endpoint := newFakeAPI(t)
	f.tooOld = true
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: collectorConfig(endpoint, "1.0.0"), ExpectError: regexp.MustCompile(`older than 1\.2\.0`)},
		},
	})
}

func TestAccProviderNeedsALogin(t *testing.T) {
	t.Setenv("TRACEFORCE_API_KEY", "")
	t.Setenv("TRACEFORCE_CLIENT_ID", "")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "traceforce" {}

resource "traceforce_cloud_connection" "test" {
  provider_name  = "aws"
  account        = "111111111111"
  module_version = "1.2.0"
}
`,
				ExpectError: regexp.MustCompile(`Missing Traceforce API key`),
			},
		},
	})
}
