# Create a project in AWS
resource "traceforce_project" "production" {
  name           = "production"
  type           = "Customer Managed"
  cloud_provider = "AWS"
  native_id      = "123456789012" # AWS Account ID
}

# Create a project in GCP  
resource "traceforce_project" "staging" {
  name           = "staging"
  type           = "TraceForce Managed"
  cloud_provider = "GCP"
  native_id      = "my-gcp-project-id"
}

# Create a datalake in the project
resource "traceforce_datalake" "analytics" {
  name       = "analytics"
  project_id = traceforce_project.production.id
  type       = "BigQuery"
}

# Create a source app connected to the datalake
resource "traceforce_source_app" "salesforce" {
  name        = "salesforce-prod"
  datalake_id = traceforce_datalake.analytics.id
  type        = "Salesforce"
}