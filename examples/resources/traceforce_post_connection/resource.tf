# Manage example order.
resource "traceforce_post_connection" "example" {
  name                  = "example"
  environment_type      = "AWS"
  environment_native_id = "9876543210"
  depends_on            = [traceforce_connection.example-aws]
}
