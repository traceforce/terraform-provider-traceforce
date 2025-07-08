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

resource "traceforce_connection" "example-gcp" {
  name                  = "example"
  environment_type      = "GCP"
  environment_native_id = var.google_project_id
  status                = "connecting"
}

module "google-bigquery-datasets" {
  source     = "./modules/google-bigquery-datasets"
  project_id = var.google_project_id
  dataset_id = var.dataset_id
  depends_on = [traceforce_connection.example-gcp]
}

resource "traceforce_post_connection" "post-connection-example-gcp" {
  name                  = "example"
  environment_type      = "GCP"
  environment_native_id = module.google-bigquery-datasets.dataset_id
  depends_on            = [module.google-bigquery-datasets]
}

output "connection-gcp" {
  value = module.google-bigquery-datasets.self_link
}
