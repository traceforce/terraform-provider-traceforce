terraform {
  required_providers {
    traceforce = {
      source = "registry.terraform.io/traceforce/traceforce"
    }
  }
}

provider "traceforce" {
  # Configure with environment variables:
  # TRACEFORCE_API_KEY
  # TRACEFORCE_ENDPOINT (optional)
}

data "traceforce_projects" "example" {}

