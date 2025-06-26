terraform {
  required_providers {
    traceforce = {
      source = "hashicorp.com/edu/traceforce"
    }
  }
}

provider "traceforce" {}

data "traceforce_connections" "example" {}

