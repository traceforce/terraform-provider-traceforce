terraform {
  required_providers {
    traceforce = {
      source = "registry.terraform.io/traceforce/traceforce"
    }
  }
}

provider "traceforce" {}

# Create a project in AWS
resource "traceforce_project" "example-aws" {
  name           = "example-project"
  type           = "Customer Managed"
  cloud_provider = "AWS"
  native_id      = "123456789012"
}

# Create a datalake in the project
resource "traceforce_datalake" "analytics" {
  name       = "analytics"
  project_id = traceforce_project.example-aws.id
  type       = "BigQuery"
}

# Create a source app connected to the datalake
resource "traceforce_source_app" "salesforce" {
  name        = "salesforce-prod"
  datalake_id = traceforce_datalake.analytics.id
  type        = "Salesforce"
}

# Establish post-connection setup
resource "traceforce_post_connection" "post-connection-example-aws" {
  project_id = traceforce_project.example-aws.id
  depends_on = [traceforce_project.example-aws]
}

# Query existing projects
data "traceforce_projects" "all" {}

output "projects" {
  value = data.traceforce_projects.all
}

output "project-aws" {
  value = traceforce_project.example-aws
}

output "datalake" {
  value = traceforce_datalake.analytics
}

output "source-app" {
  value = traceforce_source_app.salesforce
}
