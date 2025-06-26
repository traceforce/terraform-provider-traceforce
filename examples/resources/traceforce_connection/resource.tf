# Manage example order.
resource "traceforce_connection" "example" {
  name                  = "example"
  environment_type      = "AWS"
  environment_native_id = "9876543210"
  status                = "disconnected"
}
