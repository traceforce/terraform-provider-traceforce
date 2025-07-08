# Terraform Provider Traceforce

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.23

## Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:

```shell
go install
```

This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

## Using the provider
First get the API key from https://www.traceforce.co and set `TRACEFORCE_API_KEY=<your_api_key>`.

```terraform
terraform {
  required_providers {
    traceforce = {
      source = "registry.terraform.io/hashicorp/traceforce"
    }
  }
}

provider "traceforce" {}

resource "traceforce_connection" "example" {
  name                  = "example"
  environment_type      = "AWS"
  environment_native_id = "9876543210"
  status                = "disconnected"
}

data "traceforce_connections" "example" {}

output "connections" {
  value = data.traceforce_connections.example
}

output "new_connection" {
  value = traceforce_connection.example
}
```
