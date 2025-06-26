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

Then commit the changes to `go.mod` and `go.sum`.

## Using the provider

```terraform
terraform {
  required_providers {
    traceforce = {
      source = "registry.terraform.io/hashicorp/traceforce"
    }
  }
}

provider "traceforce" {
  endpoint = "your_traceforce_endpoint"
  api_key  = "your_traceforce_api_key"
}

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

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `make generate`.

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```shell
make testacc
```
