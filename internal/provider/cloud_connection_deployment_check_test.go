// Copyright (c) Traceforce, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestCheckForCloud(t *testing.T) {
	noRegions := types.SetNull(types.StringType)
	someRegions := types.SetValueMust(types.StringType, []attr.Value{types.StringValue("us-east-1")})
	const (
		role     = "arn:aws:iam::111111111111:role/TraceforceIntegrationRole"
		funcApp  = "/subscriptions/00000000-0000-4000-8000-000000000001/resourceGroups/rg/providers/Microsoft.Web/sites/collector"
		objectID = "00000000-0000-4000-8000-0000000000aa"
	)
	cases := []struct {
		name, cloud, collector, identity string
		regions                          types.Set
		wantErrors                       int
	}{
		{"aws ok", "aws", lambdaA, role, someRegions, 0},
		{"aws ok, function alias", "aws", lambdaA + ":live", role, noRegions, 0},
		{"aws: collector is not a Lambda ARN", "aws", "my-function", role, noRegions, 1},
		{"aws: identity is not a role ARN", "aws", lambdaA, "arn:aws:iam::111111111111:user/U", noRegions, 1},
		{"aws: Function App id", "aws", funcApp, objectID, noRegions, 2},
		{"m365 ok", "m365", funcApp, objectID, noRegions, 0},
		{"m365: regions refused", "m365", funcApp, objectID, someRegions, 1},
		{"m365: Lambda ARN and role ARN", "m365", lambdaA, role, noRegions, 2},
		{"unknown cloud: not checked", "gcp", "x", "y", noRegions, 0},
	}
	for _, c := range cases {
		plan := &cloudConnectionDeploymentResourceModel{
			CollectorID: types.StringValue(c.collector),
			IdentityID:  types.StringValue(c.identity),
			Regions:     c.regions,
		}
		if got := checkForCloud(c.cloud, plan).ErrorsCount(); got != c.wantErrors {
			t.Errorf("%s: %d errors, want %d", c.name, got, c.wantErrors)
		}
	}
}
