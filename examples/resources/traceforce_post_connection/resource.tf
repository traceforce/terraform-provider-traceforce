resource "traceforce_post_connection" "example" {
  project_id = traceforce_project.example.id

  infrastructure = {
    base = {
      dataplane_identity_identifier     = "dataplane-identity-12345"
      workload_identity_provider_name   = "projects/123/locations/global/workloadIdentityPools/traceforce-pool/providers/control-plane-aws"
      auth_view_generator_function_name = "auth-view-generator-function"
      traceforce_bucket_name            = "traceforce-bucket"
    }

    bigquery = {
      traceforce_schema        = "traceforce_dataset"
      events_subscription_name = "bigquery-events-subscription"
    }

    salesforce = {
      salesforce_client_id     = "3MVG9g9rbsTkKnAXABCDEFGHIJKLMNOPQRSTUVWXYZ"
      salesforce_domain        = "mycompany.my.salesforce.com"
      salesforce_client_secret = "projects/example/secrets/salesforce-secret/versions/latest"
    }
  }

  terraform_url                  = "https://github.com/traceforce/terraform-modules"
  terraform_module_versions      = <<-EOT
  {
    "base_infrastructure": {
      "major": 1,
      "minor": 0
    },
    "datalake_connectors": {
      "bigquery": {
        "major": 1,
        "minor": 0
      }
    },
    "source_connectors": {
      "salesforce": {
        "major": 1,
        "minor": 0
      }
    }
  }
  EOT
  terraform_module_versions_hash = "sha256:abcdef123456..."
  deployed_datalake_ids          = ["datalake-abc123"]
  deployed_source_app_ids        = ["sourceapp-def456"]

  depends_on = [traceforce_project.example]
}