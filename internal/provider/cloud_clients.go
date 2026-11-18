// Copyright (c) Traceforce, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	traceforce "github.com/traceforce/traceforce-go-sdk"
)

// cloudClients gives the cloud-connection resources the API client for a
// cloud. Traceforce only lets a cloud's own collector token reach that cloud's
// connections, so with an API client login (client_id) each cloud gets its own
// login, made on first use and kept for the rest of the Terraform operation.
// Each resource names its cloud from its own configuration or state: nothing
// is remembered between operations. With an api_key there is one client.
type cloudClients struct {
	mu       sync.Mutex
	plain    *traceforce.Client // the provider's client; nil until the provider is configured
	creds    traceforce.Credentials
	endpoint string
	options  *traceforce.ClientOptions
	byCloud  map[string]*traceforce.Client
}

// configure is called by the provider's Configure with its own client and, for
// an API client login, the credentials to log in per cloud (empty for an api_key).
func (c *cloudClients) configure(plain *traceforce.Client, creds traceforce.Credentials, endpoint string, options *traceforce.ClientOptions) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.plain, c.creds, c.endpoint, c.options = plain, creds, endpoint, options
	c.byCloud = map[string]*traceforce.Client{}
}

// forCloud is the API client for one cloud ("aws", "m365").
func (c *cloudClients) forCloud(cloud string) (*traceforce.Client, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.plain == nil {
		return nil, errors.New("the Traceforce provider is not configured")
	}
	if c.creds.ClientID == "" {
		return c.plain, nil // api_key login
	}
	if cloud == "" {
		return nil, fmt.Errorf("the cloud of this connection is not known; expected one of %s", strings.Join(cloudProviders, ", "))
	}
	if client, ok := c.byCloud[cloud]; ok {
		return client, nil
	}
	creds := c.creds
	creds.Cloud = cloud
	client, err := traceforce.NewClientWithCredentials(creds, c.endpoint, c.options)
	if err != nil {
		return nil, err
	}
	c.byCloud[cloud] = client
	return client, nil
}
