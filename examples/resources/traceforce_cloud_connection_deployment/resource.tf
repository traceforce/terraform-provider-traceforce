resource "traceforce_cloud_connection_deployment" "this" {
  connection_id = traceforce_cloud_connection.this.id
  collector_id  = aws_lambda_function.collector.arn
  identity_id   = aws_iam_role.collector.arn
  regions       = ["us-east-1"]
}
