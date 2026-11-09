# Log in with an API key ...
provider "traceforce" {
  endpoint = "https://your-traceforce-endpoint.traceforce.ai/v1"
  api_key  = "your-api-key-here"
}

# ... or with one of your API clients (Settings -> API Clients), e.g. the cloud collector:
# provider "traceforce" {
#   endpoint      = "https://your-traceforce-endpoint.traceforce.ai/v1"
#   client_id     = var.traceforce_client_id
#   client_secret = var.traceforce_client_secret
#   device_id     = "cloud-collector-aws"
# }
