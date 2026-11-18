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
	existing bool   // the next register finds a row registered before (created: false)
}

func newFakeAPI(t *testing.T) (*fakeAPI, string) {
	f := &fakeAPI{}
	// https: the SDK logs in with an API client only over https. This process trusts the
	// test certificate until the test ends (these tests do not run in parallel).
	srv := httptest.NewTLSServer(http.HandlerFunc(f.serve))
	old := http.DefaultTransport
	http.DefaultTransport = srv.Client().Transport
	t.Cleanup(func() { http.DefaultTransport = old; srv.Close() })
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

	// Login: a plain token, or a cloud's collector token (device_id cloud-collector-<cloud>).
	if r.Method == http.MethodPost && path == "/api-keys" {
		f.logins = append(f.logins, body)
		token := "tok"
		if device, _ := body["device_id"].(string); device != "" {
			token += "-" + strings.TrimPrefix(device, "cloud-collector-")
		}
		reply(w, 200, map[string]string{"access_token": token})
		return
	}
	if !strings.HasPrefix(auth, "Bearer tok") && auth != "Bearer "+fakeAPIKey {
		reply(w, 401, map[string]string{"error": "Authentication required"})
		return
	}
	f.auths = append(f.auths, auth)
	conn := "/cloud-connections/"
	cloud, _ := body["provider"].(string) // POST: the cloud of the body
	if f.row != nil {
		id, _ := f.row["id"].(string)
		conn += id
		if r.Method != http.MethodPost || path != "/cloud-connections" {
			cloud, _ = f.row["provider"].(string)
		}
	}
	// Like Traceforce: a client token must be the collector token of the connection's cloud.
	if strings.HasPrefix(auth, "Bearer tok") && cloud != "" && auth != "Bearer tok-"+cloud {
		reply(w, 403, map[string]string{"error": "Only the cloud collector may call this"})
		return
	}

	switch {
	case r.Method == http.MethodPost && path == "/cloud-connections":
		f.posts = append(f.posts, body)
		if f.tooOld {
			reply(w, 426, map[string]string{"error": "Collector 1.0.0 is older than 1.2.0.", "code": "collector_too_old"})
			return
		}
		created := f.row == nil && !f.existing
		if f.row == nil {
			f.row = map[string]any{"id": fakeConnID, "provider": body["provider"], "status": "pending", "name": nil, "regions": []string{}}
		}
		f.existing = false
		account, _ := body["account"].(string)
		f.row["account"] = strings.ToLower(account)
		f.row["module_version"], f.row["module_commit"] = body["module_version"], body["module_commit"]
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
	if len(f.logins) > 0 && len(f.auths) > 0 && strings.HasPrefix(f.auths[0], "Bearer tok-") {
		return "client cloud-collector-" + strings.TrimPrefix(f.auths[0], "Bearer tok-")
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
