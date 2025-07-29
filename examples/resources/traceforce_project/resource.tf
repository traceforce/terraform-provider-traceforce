resource "traceforce_project" "production" {
  name           = "production"
  type           = "customer_managed"
  cloud_provider = "AWS"
  native_id      = "123456789012"
}

resource "traceforce_project" "staging" {
  name           = "staging"
  type           = "traceforce_managed"
  cloud_provider = "GCP"
  native_id      = "my-gcp-project-id"
}

resource "traceforce_datalake" "analytics" {
  name       = "analytics"
  project_id = traceforce_project.production.id
  type       = "bigquery"
}

resource "traceforce_source_app" "salesforce" {
  name        = "salesforce-prod"
  datalake_id = traceforce_datalake.analytics.id
  type        = "salesforce"
}