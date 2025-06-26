terraform {
  required_providers {
    traceforce = {
      source = "hashicorp.com/edu/traceforce"
    }
  }
}

provider "traceforce" {
  endpoint = "https://zexktzntymqvltijndxl.supabase.co"
  api_key  = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6InpleGt0em50eW1xdmx0aWpuZHhsIiwicm9sZSI6ImFub24iLCJpYXQiOjE3NTA4MDY4MzksImV4cCI6MjA2NjM4MjgzOX0.s_CNf2JwkPQn6064T79_5gqZ8lyALxwgFSseJIHnWnk"
}

resource "traceforce_connection" "example-aws" {
  name                  = "example"
  environment_type      = "AWS"
  environment_native_id = "9876543210"
  status                = "disconnected"
}

resource "traceforce_connection" "example-gcp" {
  id                    = "a5d1ed31-5400-4fd2-b0d6-b795931e1f21"
  name                  = "test-connection-1"
  environment_type      = "GCP"
  environment_native_id = "test-project-1"
  status                = "connected"
}


data "traceforce_connections" "example" {}

output "connections" {
  value = data.traceforce_connections.example
}

output "connection-aws" {
  value = traceforce_connection.example-aws
}

output "connection-gcp" {
  value = traceforce_connection.example-gcp
}