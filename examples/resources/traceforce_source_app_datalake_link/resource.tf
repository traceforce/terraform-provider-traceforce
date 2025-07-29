resource "traceforce_project" "production" {
  name           = "production"
  type           = "customer_managed"
  cloud_provider = "GCP"
  native_id      = "my-gcp-project-id"
}

resource "traceforce_datalake" "analytics" {
  project_id            = traceforce_project.production.id
  name                  = "analytics"
  type                  = "bigquery"
  environment_native_id = "my-gcp-project-id"
  region                = "us-central1"
}

resource "traceforce_source_app" "salesforce" {
  project_id = traceforce_project.production.id
  name       = "salesforce-prod"
  type       = "salesforce"
}

resource "traceforce_source_app_datalake_link" "salesforce_to_analytics" {
  source_app_id = traceforce_source_app.salesforce.id
  datalake_id   = traceforce_datalake.analytics.id
}

resource "traceforce_datalake" "warehouse" {
  project_id            = traceforce_project.production.id
  name                  = "warehouse"
  type                  = "bigquery"
  environment_native_id = "my-gcp-project-id"
  region                = "us-central1"
}

resource "traceforce_source_app_datalake_link" "salesforce_to_warehouse" {
  source_app_id = traceforce_source_app.salesforce.id
  datalake_id   = traceforce_datalake.warehouse.id
}

resource "traceforce_source_app" "hubspot" {
  project_id = traceforce_project.production.id
  name       = "hubspot-prod"
  type       = "hubspot"
}

resource "traceforce_source_app_datalake_link" "hubspot_to_analytics" {
  source_app_id = traceforce_source_app.hubspot.id
  datalake_id   = traceforce_datalake.analytics.id
}