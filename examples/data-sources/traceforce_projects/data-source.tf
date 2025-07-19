# Get all projects
data "traceforce_projects" "all" {}

# Filter projects by status
data "traceforce_projects" "connected" {
  filter = {
    status = "Connected"
  }
}

# Filter by cloud provider
data "traceforce_projects" "aws_projects" {
  filter = {
    cloud_provider = "AWS"
  }
}

# Use project data in other resources
resource "traceforce_datalake" "from_existing_project" {
  name       = "new-analytics"
  project_id = data.traceforce_projects.all.projects[0].id
  type       = "BigQuery"
}