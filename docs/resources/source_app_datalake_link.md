---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "traceforce_source_app_datalake_link Resource - traceforce"
subcategory: ""
description: |-
  Links a source app to a datalake within the same hosting environment.
---

# traceforce_source_app_datalake_link (Resource)

Links a source app to a datalake within the same hosting environment.

## Example Usage

```terraform
resource "traceforce_project" "production" {
  name           = "production"
  type           = "customer_managed"
  cloud_provider = "GCP"
  native_id      = "my-gcp-project-id"
}

resource "traceforce_datalake" "analytics" {
  project_id            = traceforce_project.production.id
  name                  = "analytics"
  type                  = "bigquery"
  environment_native_id = "my-gcp-project-id"
  region                = "us-central1"
}

resource "traceforce_source_app" "salesforce" {
  project_id = traceforce_project.production.id
  name       = "salesforce-prod"
  type       = "salesforce"
}

resource "traceforce_source_app_datalake_link" "salesforce_to_analytics" {
  source_app_id = traceforce_source_app.salesforce.id
  datalake_id   = traceforce_datalake.analytics.id
}

resource "traceforce_datalake" "warehouse" {
  project_id            = traceforce_project.production.id
  name                  = "warehouse"
  type                  = "bigquery"
  environment_native_id = "my-gcp-project-id"
  region                = "us-central1"
}

resource "traceforce_source_app_datalake_link" "salesforce_to_warehouse" {
  source_app_id = traceforce_source_app.salesforce.id
  datalake_id   = traceforce_datalake.warehouse.id
}

resource "traceforce_source_app" "hubspot" {
  project_id = traceforce_project.production.id
  name       = "hubspot-prod"
  type       = "hubspot"
}

resource "traceforce_source_app_datalake_link" "hubspot_to_analytics" {
  source_app_id = traceforce_source_app.hubspot.id
  datalake_id   = traceforce_datalake.analytics.id
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `datalake_id` (String) ID of the datalake to link.
- `source_app_id` (String) ID of the source app to link.

### Read-Only

- `created_at` (String) Date and time the link was created
- `hosting_environment_id` (String) ID of the hosting environment (derived from linked resources).
- `id` (String) System generated ID of the link
- `updated_at` (String) Date and time the link was last updated
