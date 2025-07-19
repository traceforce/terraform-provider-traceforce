# Establish post-connection setup for a project
resource "traceforce_post_connection" "example" {
  project_id = traceforce_project.example.id
  depends_on = [traceforce_project.example]
}
