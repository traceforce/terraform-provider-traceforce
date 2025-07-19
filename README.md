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
      source = "registry.terraform.io/traceforce/traceforce"
    }
  }
}

provider "traceforce" {}

# Create a project
resource "traceforce_project" "example" {
  name           = "example-project"
  type           = "Customer Managed"
  cloud_provider = "AWS"
  native_id      = "123456789012"
}

# Create a datalake in the project
resource "traceforce_datalake" "analytics" {
  name       = "analytics"
  project_id = traceforce_project.example.id
  type       = "BigQuery"
}

data "traceforce_projects" "all" {}

output "projects" {
  value = data.traceforce_projects.all
}

output "new_project" {
  value = traceforce_project.example
}
```
