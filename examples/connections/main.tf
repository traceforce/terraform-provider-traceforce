terraform {
  required_providers {
    traceforce = {
      source = "hashicorp.com/edu/traceforce"
    }
  }
}

provider "traceforce" {
  api_key  = "eyJhbGciOiJIUzI1NiIsImtpZCI6ImFRMUxOVzFFY3hCT1hhRzQiLCJ0eXAiOiJKV1QifQ.eyJpc3MiOiJodHRwczovL3pleGt0em50eW1xdmx0aWpuZHhsLnN1cGFiYXNlLmNvL2F1dGgvdjEiLCJzdWIiOiJlZDliNDJiNi05OTFmLTQzOWUtOTRlMy0zZDMxZWZjNWJiMWYiLCJhdWQiOiJhdXRoZW50aWNhdGVkIiwiZXhwIjoxNzUxMjM0NTg0LCJpYXQiOjE3NTEyMzA5ODQsImVtYWlsIjoieGlhQHRyYWNlZm9yY2UuYWkiLCJwaG9uZSI6IiIsImFwcF9tZXRhZGF0YSI6eyJwcm92aWRlciI6ImVtYWlsIiwicHJvdmlkZXJzIjpbImVtYWlsIl19LCJ1c2VyX21ldGFkYXRhIjp7ImVtYWlsIjoieGlhQHRyYWNlZm9yY2UuYWkiLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwicGhvbmVfdmVyaWZpZWQiOmZhbHNlLCJzdWIiOiJlZDliNDJiNi05OTFmLTQzOWUtOTRlMy0zZDMxZWZjNWJiMWYifSwicm9sZSI6ImF1dGhlbnRpY2F0ZWQiLCJhYWwiOiJhYWwxIiwiYW1yIjpbeyJtZXRob2QiOiJwYXNzd29yZCIsInRpbWVzdGFtcCI6MTc1MTIzMDk4NH1dLCJzZXNzaW9uX2lkIjoiMDQyZGU5NjktNjAzZS00ZmRkLTk4ZmItNzE0MDhlY2UzMjYzIiwiaXNfYW5vbnltb3VzIjpmYWxzZX0.zT_BowQNqq-ByWrZTNZOIzOyNzPh1FzlLFgpS5c_BvE"
}

resource "traceforce_connection" "example-aws" {
  name                  = "example"
  environment_type      = "AWS"
  environment_native_id = "9876543210"
  status                = "disconnected"
}

data "traceforce_connections" "example" {}

output "connections" {
  value = data.traceforce_connections.example
}

output "connection-aws" {
  value = traceforce_connection.example-aws
}
