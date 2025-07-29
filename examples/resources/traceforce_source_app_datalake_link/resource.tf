# Create a hosting environment
resource "traceforce_project" "production" {
  name           = "production"
  type           = "Customer Managed"
  cloud_provider = "GCP"
  native_id      = "my-gcp-project-id"
}

# Create a datalake
resource "traceforce_datalake" "analytics" {
  project_id            = traceforce_project.production.id
  name                  = "analytics"
  type                  = "BigQuery"
  environment_native_id = "my-gcp-project-id"
  region                = "us-central1"
}

# Create a source app
resource "traceforce_source_app" "salesforce" {
  project_id = traceforce_project.production.id
  name       = "salesforce-prod"
  type       = "Salesforce"
}

# Link the source app to the datalake
resource "traceforce_source_app_datalake_link" "salesforce_to_analytics" {
  source_app_id = traceforce_source_app.salesforce.id
  datalake_id   = traceforce_datalake.analytics.id
}

# Create another datalake for the same hosting environment
resource "traceforce_datalake" "warehouse" {
  project_id            = traceforce_project.production.id
  name                  = "warehouse"
  type                  = "BigQuery"
  environment_native_id = "my-gcp-project-id"
  region                = "us-central1"
}

# Link the same source app to multiple datalakes
resource "traceforce_source_app_datalake_link" "salesforce_to_warehouse" {
  source_app_id = traceforce_source_app.salesforce.id
  datalake_id   = traceforce_datalake.warehouse.id
}

# Create another source app
resource "traceforce_source_app" "hubspot" {
  project_id = traceforce_project.production.id
  name       = "hubspot-prod"
  type       = "Salesforce" # Note: Adjust type based on available source app types
}

# Link multiple source apps to the same datalake
resource "traceforce_source_app_datalake_link" "hubspot_to_analytics" {
  source_app_id = traceforce_source_app.hubspot.id
  datalake_id   = traceforce_datalake.analytics.id
}