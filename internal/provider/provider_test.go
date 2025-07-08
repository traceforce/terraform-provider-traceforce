// Copyright (c) Traceforce, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

const (
	// providerConfig is a shared configuration to combine with the actual
	// test configuration so the Traceforce client is properly configured.
	// It is also possible to use the TRACEFORCE_ environment variables instead,
	// such as updating the Makefile and running the testing through that tool.
	providerConfig = `
provider "traceforce" {
  api_key  = "eyJhbGciOiJIUzI1NiIsImtpZCI6ImFRMUxOVzFFY3hCT1hhRzQiLCJ0eXAiOiJKV1QifQ.eyJpc3MiOiJodHRwczovL3pleGt0em50eW1xdmx0aWpuZHhsLnN1cGFiYXNlLmNvL2F1dGgvdjEiLCJzdWIiOiJmNmYzNGE0Ni1jYmFlLTRkZjctOWQzNS0wNzY2ZTM4ZjZhZjIiLCJhdWQiOiJhdXRoZW50aWNhdGVkIiwiZXhwIjoxNzUyMDAyNjYxLCJpYXQiOjE3NTE5OTkwNjEsImVtYWlsIjoieGlhQHRyYWNlZm9yY2UuYWkiLCJwaG9uZSI6IiIsImFwcF9tZXRhZGF0YSI6eyJwcm92aWRlciI6InNzbzpiOGMwYTM2MS0xNWY2LTRjZjctYWQ1Yy02NjYyMWJlMDViNDUiLCJwcm92aWRlcnMiOlsic3NvOmI4YzBhMzYxLTE1ZjYtNGNmNy1hZDVjLTY2NjIxYmUwNWI0NSJdfSwidXNlcl9tZXRhZGF0YSI6eyJjdXN0b21fY2xhaW1zIjp7fSwiZW1haWwiOiJ4aWFAdHJhY2Vmb3JjZS5haSIsImVtYWlsX3ZlcmlmaWVkIjp0cnVlLCJpc3MiOiJodHRwczovL2FjY291bnRzLmdvb2dsZS5jb20vby9zYW1sMj9pZHBpZD1DMDB0M2x5cHYiLCJwaG9uZV92ZXJpZmllZCI6ZmFsc2UsInN1YiI6InhpYUB0cmFjZWZvcmNlLmFpIn0sInJvbGUiOiJhdXRoZW50aWNhdGVkIiwiYWFsIjoiYWFsMSIsImFtciI6W3sibWV0aG9kIjoic3NvL3NhbWwiLCJ0aW1lc3RhbXAiOjE3NTE5OTkwNjEsInByb3ZpZGVyIjoiYjhjMGEzNjEtMTVmNi00Y2Y3LWFkNWMtNjY2MjFiZTA1YjQ1In1dLCJzZXNzaW9uX2lkIjoiZjNlM2Q2YWMtYWY4ZS00MzgzLTk1NGMtMDRhZTdlZGEyNTI3IiwiaXNfYW5vbnltb3VzIjpmYWxzZX0.oGet1JhJNwl2m1BCFHOtEU9UQeAtLukTXjZugH1ZOp4"
}
`
)

var (
	// testAccProtoV6ProviderFactories are used to instantiate a provider during
	// acceptance testing. The factory function will be invoked for every Terraform
	// CLI command executed to create a provider server to which the CLI can
	// reattach.
	testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"traceforce": providerserver.NewProtocol6WithError(New("test")()),
	}
)
