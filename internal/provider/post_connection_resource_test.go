// Copyright (c) Traceforce, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPostConnectionResource(t *testing.T) {
	// Generate unique names with Z prefix for parallel execution
	projectId := "z-project-" + uuid.New().String()
	resourceName := "traceforce_post_connection.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing - basic post connection
			{
				Config: testAccPostConnectionResourceConfig(projectId),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "project_id", projectId),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "status"),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
					resource.TestCheckResourceAttrSet(resourceName, "updated_at"),
				),
			},
			// Import State testing
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccPostConnectionResourceWithBigQuery(t *testing.T) {
	// Generate unique names with Z prefix for parallel execution
	projectId := "z-project-" + uuid.New().String()
	resourceName := "traceforce_post_connection.test"
	traceforceSchema := "z_traceforce_dataset_" + uuid.New().String()
	eventsSubscription := "z-events-subscription-" + uuid.New().String()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing - with BigQuery configuration
			{
				Config: testAccPostConnectionResourceConfigWithBigQuery(projectId, traceforceSchema, eventsSubscription),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "project_id", projectId),
					resource.TestCheckResourceAttr(resourceName, "infrastructure.bigquery.traceforce_schema", traceforceSchema),
					resource.TestCheckResourceAttr(resourceName, "infrastructure.bigquery.events_subscription_name", eventsSubscription),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "status"),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
					resource.TestCheckResourceAttrSet(resourceName, "updated_at"),
				),
			},
			// Import State testing
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccPostConnectionResourceWithSalesforce(t *testing.T) {
	// Generate unique names with Z prefix for parallel execution
	projectId := "z-project-" + uuid.New().String()
	resourceName := "traceforce_post_connection.test"
	clientId := "test_client_id_" + uuid.New().String()
	domain := "test-domain-" + uuid.New().String() + ".my.salesforce.com"
	secretMountPath := "projects/test-project/secrets/salesforce-secret/versions/latest"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing - with Salesforce configuration
			{
				Config: testAccPostConnectionResourceConfigWithSalesforce(projectId, clientId, domain, secretMountPath),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "project_id", projectId),
					resource.TestCheckResourceAttr(resourceName, "infrastructure.salesforce.salesforce_client_id", clientId),
					resource.TestCheckResourceAttr(resourceName, "infrastructure.salesforce.salesforce_domain", domain),
					resource.TestCheckResourceAttr(resourceName, "infrastructure.salesforce.salesforce_client_secret", secretMountPath),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "status"),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
					resource.TestCheckResourceAttrSet(resourceName, "updated_at"),
				),
			},
			// Import State testing
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccPostConnectionResourceWithBoth(t *testing.T) {
	// Generate unique names with Z prefix for parallel execution
	projectId := "z-project-" + uuid.New().String()
	resourceName := "traceforce_post_connection.test"
	traceforceSchema := "z_traceforce_dataset_" + uuid.New().String()
	eventsSubscription := "z-events-subscription-" + uuid.New().String()
	clientId := "test_client_id_" + uuid.New().String()
	domain := "test-domain-" + uuid.New().String() + ".my.salesforce.com"
	secretMountPath := "projects/test-project/secrets/salesforce-secret/versions/latest"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing - with both BigQuery and Salesforce configuration
			{
				Config: testAccPostConnectionResourceConfigWithBoth(projectId, traceforceSchema, eventsSubscription, clientId, domain, secretMountPath),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "project_id", projectId),
					resource.TestCheckResourceAttr(resourceName, "infrastructure.bigquery.traceforce_schema", traceforceSchema),
					resource.TestCheckResourceAttr(resourceName, "infrastructure.bigquery.events_subscription_name", eventsSubscription),
					resource.TestCheckResourceAttr(resourceName, "infrastructure.salesforce.salesforce_client_id", clientId),
					resource.TestCheckResourceAttr(resourceName, "infrastructure.salesforce.salesforce_domain", domain),
					resource.TestCheckResourceAttr(resourceName, "infrastructure.salesforce.salesforce_client_secret", secretMountPath),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "status"),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
					resource.TestCheckResourceAttrSet(resourceName, "updated_at"),
				),
			},
			// Import State testing
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccPostConnectionResourceWithBase(t *testing.T) {
	// Generate unique names with Z prefix for parallel execution
	projectId := "z-project-" + uuid.New().String()
	resourceName := "traceforce_post_connection.test"
	dataplaneIdentifier := "z-dataplane-" + uuid.New().String()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing - with base infrastructure configuration
			{
				Config: testAccPostConnectionResourceConfigWithBase(projectId, dataplaneIdentifier),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "project_id", projectId),
					resource.TestCheckResourceAttr(resourceName, "infrastructure.base.dataplane_identity_identifier", dataplaneIdentifier),
					resource.TestCheckResourceAttr(resourceName, "infrastructure.base.workload_identity_provider_name", "projects/123/locations/global/workloadIdentityPools/test-pool/providers/test-provider"),
					resource.TestCheckResourceAttr(resourceName, "infrastructure.base.auth_view_generator_function_name", "test-auth-view-generator-function"),
					resource.TestCheckResourceAttr(resourceName, "infrastructure.base.auth_view_generator_function_url", "https://test-function-url.cloudfunctions.net/auth-view-generator"),
					resource.TestCheckResourceAttr(resourceName, "infrastructure.base.traceforce_bucket_name", "test-traceforce-bucket"),
					resource.TestCheckResourceAttrSet(resourceName, "terraform_url"),
					resource.TestCheckResourceAttrSet(resourceName, "terraform_module_versions"),
					resource.TestCheckResourceAttrSet(resourceName, "terraform_module_versions_hash"),
				),
			},
			// Import State testing
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// testAccPostConnectionResourceConfig returns a basic configuration for post_connection resource.
func testAccPostConnectionResourceConfig(projectId string) string {
	return fmt.Sprintf(`
%s

resource "traceforce_post_connection" "test" {
  project_id = "%s"
  
  infrastructure = {
    # Empty infrastructure for basic setup
  }
  
  terraform_url = "https://github.com/traceforce/terraform-modules"
  terraform_module_versions = "{\"base\": \"v1.0.0\"}"
  terraform_module_versions_hash = "abc123def456"
  deployed_datalake_ids = []
  deployed_source_app_ids = []
}
`, providerConfig, projectId)
}

// testAccPostConnectionResourceConfigWithBigQuery returns configuration with BigQuery infrastructure.
func testAccPostConnectionResourceConfigWithBigQuery(projectId, traceforceSchema, eventsSubscription string) string {
	return fmt.Sprintf(`
%s

resource "traceforce_post_connection" "test" {
  project_id = "%s"
  
  infrastructure = {
    bigquery = {
      traceforce_schema        = "%s"
      events_subscription_name = "%s"
    }
  }
  
  terraform_url = "https://github.com/traceforce/terraform-modules"
  terraform_module_versions = "{\"bigquery\": \"v1.0.0\"}"
  terraform_module_versions_hash = "def456ghi789"
  deployed_datalake_ids = ["datalake-1"]
  deployed_source_app_ids = []
}
`, providerConfig, projectId, traceforceSchema, eventsSubscription)
}

// testAccPostConnectionResourceConfigWithSalesforce returns configuration with Salesforce infrastructure.
func testAccPostConnectionResourceConfigWithSalesforce(projectId, clientId, domain, secretMountPath string) string {
	return fmt.Sprintf(`
%s

resource "traceforce_post_connection" "test" {
  project_id = "%s"
  
  infrastructure = {
    salesforce = {
      salesforce_client_id     = "%s"
      salesforce_domain        = "%s"
      salesforce_client_secret = "%s"
    }
  }
  
  terraform_url = "https://github.com/traceforce/terraform-modules"
  terraform_module_versions = "{\"salesforce\": \"v1.0.0\"}"
  terraform_module_versions_hash = "ghi789jkl012"
  deployed_datalake_ids = []
  deployed_source_app_ids = ["source-app-1"]
}
`, providerConfig, projectId, clientId, domain, secretMountPath)
}

// testAccPostConnectionResourceConfigWithBoth returns configuration with both BigQuery and Salesforce infrastructure.
func testAccPostConnectionResourceConfigWithBoth(projectId, traceforceSchema, eventsSubscription, clientId, domain, secretMountPath string) string {
	return fmt.Sprintf(`
%s

resource "traceforce_post_connection" "test" {
  project_id = "%s"
  
  infrastructure = {
    bigquery = {
      traceforce_schema        = "%s"
      events_subscription_name = "%s"
    }
    
    salesforce = {
      salesforce_client_id     = "%s"
      salesforce_domain        = "%s"
      salesforce_client_secret = "%s"
    }
  }
  
  terraform_url = "https://github.com/traceforce/terraform-modules"
  terraform_module_versions = "{\"bigquery\": \"v1.0.0\", \"salesforce\": \"v1.0.0\"}"
  terraform_module_versions_hash = "jkl012mno345"
  deployed_datalake_ids = ["datalake-1", "datalake-2"]
  deployed_source_app_ids = ["source-app-1", "source-app-2"]
}
`, providerConfig, projectId, traceforceSchema, eventsSubscription, clientId, domain, secretMountPath)
}

// testAccPostConnectionResourceConfigWithBase returns configuration with base infrastructure.
func testAccPostConnectionResourceConfigWithBase(projectId, dataplaneIdentifier string) string {
	return fmt.Sprintf(`
%s

resource "traceforce_post_connection" "test" {
  project_id = "%s"
  
  infrastructure = {
    base = {
      dataplane_identity_identifier = "%s"
      workload_identity_provider_name = "projects/123/locations/global/workloadIdentityPools/test-pool/providers/test-provider"
      auth_view_generator_function_name = "test-auth-view-generator-function"
      auth_view_generator_function_url = "https://test-function-url.cloudfunctions.net/auth-view-generator"
      traceforce_bucket_name = "test-traceforce-bucket"
    }
  }
  
  terraform_url = "https://github.com/traceforce/terraform-modules"
  terraform_module_versions = "{\"base\": \"v1.0.0\"}"
  terraform_module_versions_hash = "abc123def456"
  deployed_datalake_ids = []
  deployed_source_app_ids = []
}
`, providerConfig, projectId, dataplaneIdentifier)
}
