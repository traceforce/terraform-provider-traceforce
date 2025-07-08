terraform {
  required_providers {
    traceforce = {
      source = "registry.terraform.io/traceforce/traceforce"
    }
  }
}

provider "traceforce" {}

resource "traceforce_connection" "example-aws" {
  name                  = "example"
  environment_type      = "AWS"
  environment_native_id = "9876543210"
  status                = "disconnected"
}

resource "traceforce_post_connection" "post-connection-example-aws" {
  name                  = "example"
  environment_type      = "AWS"
  environment_native_id = "9876543211"
  depends_on            = [traceforce_connection.example-aws]
}

data "traceforce_connections" "example" {}

output "connections" {
  value = data.traceforce_connections.example
}

output "connection-aws" {
  value = traceforce_connection.example-aws
}

output "post-connection-aws" {
  value = traceforce_post_connection.post-connection-example-aws
}
