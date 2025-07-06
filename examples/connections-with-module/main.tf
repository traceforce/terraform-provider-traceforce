terraform {
  required_providers {
    traceforce = {
      source = "hashicorp.com/edu/traceforce"
    }
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }
}

provider "traceforce" {}

module "google-bigquery-datasets" {
  source = "./modules/google-bigquery-datasets"
  project_id = var.google_project_id
  dataset_id = var.dataset_id
}

resource "traceforce_connection" "example-gcp" {
  name                  = "example"
  environment_type      = "GCP"
  environment_native_id = module.google-bigquery-datasets.dataset_id
  status                = module.google-bigquery-datasets.self_link
}

output "connection-gcp" {
  value = module.google-bigquery-datasets.self_link
}
