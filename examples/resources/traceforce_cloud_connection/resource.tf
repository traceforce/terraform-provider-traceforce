# The provider logs in with one of your API clients (Settings → API Clients).
provider "traceforce" {
  client_id     = var.traceforce_client_id
  client_secret = var.traceforce_client_secret
  device_id     = "cloud-collector-aws"
}

resource "traceforce_cloud_connection" "this" {
  provider_name  = "aws"
  account        = "123456789012"
  name           = "Production"
  module_version = "1.2.0"
}
