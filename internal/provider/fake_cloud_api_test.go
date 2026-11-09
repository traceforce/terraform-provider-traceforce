// Copyright (c) Traceforce, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// A fake Traceforce API for the cloud-connection tests (no API key needed).
// It holds one connection and answers like the nextjs routes.

const (
	fakeConnID  = "3b0c6d1e-1111-4222-8333-444455556666"
	otherConnID = "3b0c6d1e-9999-4222-8333-444455556666"
	fakeAPIKey  = "key-1" // the api_key the fake accepts
)

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

// expect checks what the fake API saw, under its lock.
func (f *fakeAPI) expect(check func() error) func(*terraform.State) error {
	return func(*terraform.State) error {
		f.mu.Lock()
		defer f.mu.Unlock()
		return check()
	}
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
	case r.Method == http.MethodDelete && path == conn:
		f.deletes++
		f.row = nil
		reply(w, 200, map[string]bool{"disconnected": true})
	default:
		reply(w, 404, notFound)
	}
}

// usedLogin says how the fake API was called: "client <device id>" or "api_key".
func (f *fakeAPI) usedLogin() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.logins) > 0 && len(f.auths) > 0 && f.auths[0] == "Bearer tok" {
		return fmt.Sprintf("client %v", f.logins[0]["device_id"])
	}
	if len(f.auths) > 0 && f.auths[0] == "Bearer "+fakeAPIKey {
		return "api_key"
	}
	return "none"
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

// connectionConfig is the provider block and one connection; name "" leaves name out.
func connectionConfig(endpoint, cloud, account, version, name string) string {
	nameLine := ""
	if name != "" {
		nameLine = fmt.Sprintf("  name           = %q\n", name)
	}
	return collectorProvider(endpoint) + fmt.Sprintf(`
resource "traceforce_cloud_connection" "test" {
  provider_name  = %q
  account        = %q
%s  module_version = %q
}
`, cloud, account, nameLine, version)
}

func collectorConfig(endpoint, version string) string {
	return connectionConfig(endpoint, "aws", "111111111111", version, "Test")
}

const conn = "traceforce_cloud_connection.test"
