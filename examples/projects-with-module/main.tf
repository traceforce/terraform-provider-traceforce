terraform {
  required_providers {
    traceforce = {
      source = "registry.terraform.io/traceforce/traceforce"
    }
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }
}

provider "traceforce" {}

# Create a project in GCP
resource "traceforce_project" "example-gcp" {
  name           = "example-gcp"
  type           = "Customer Managed"
  cloud_provider = "GCP"
  native_id      = var.google_project_id
}

# Create BigQuery datasets using a module
module "google-bigquery-datasets" {
  source     = "./modules/google-bigquery-datasets"
  project_id = var.google_project_id
  dataset_id = var.dataset_id
  depends_on = [traceforce_project.example-gcp]
}

# Create a datalake connected to the BigQuery datasets
resource "traceforce_datalake" "bigquery-analytics" {
  name       = "bigquery-analytics"
  project_id = traceforce_project.example-gcp.id
  type       = "BigQuery"
  depends_on = [module.google-bigquery-datasets]
}

# Establish post-connection setup
resource "traceforce_post_connection" "post-connection-example-gcp" {
  project_id = traceforce_project.example-gcp.id
  depends_on = [traceforce_datalake.bigquery-analytics]
}

output "project-gcp" {
  value = traceforce_project.example-gcp
}

output "bigquery-dataset" {
  value = module.google-bigquery-datasets.self_link
}

output "datalake" {
  value = traceforce_datalake.bigquery-analytics
}
