---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "traceforce_post_connection Resource - traceforce"
subcategory: ""
description: |-
  
---

# traceforce_post_connection (Resource)



## Example Usage

```terraform
resource "traceforce_post_connection" "example" {
  project_id = traceforce_project.example.id

  infrastructure = {
    base = {
      dataplane_identity_identifier   = "dataplane-identity-12345"
      workload_identity_provider_name = "projects/123/locations/global/workloadIdentityPools/traceforce-pool/providers/control-plane-aws"
    }

    bigquery = {
      traceforce_schema        = "traceforce_dataset"
      events_subscription_name = "bigquery-events-subscription"
    }

    salesforce = {
      salesforce_client_secret = "projects/example/secrets/salesforce-secret/versions/latest"
    }
  }

  terraform_url                  = "https://github.com/traceforce/terraform-modules"
  terraform_module_versions      = <<-EOT
  {
    "base_infrastructure": {
      "major": 1,
      "minor": 0
    },
    "datalake_connectors": {
      "bigquery": {
        "major": 1,
        "minor": 0
      }
    },
    "source_connectors": {
      "salesforce": {
        "major": 1,
        "minor": 0
      }
    }
  }
  EOT
  terraform_module_versions_hash = "sha256:abcdef123456..."

  depends_on = [traceforce_project.example]
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `infrastructure` (Attributes) Infrastructure configuration for deployment (see [below for nested schema](#nestedatt--infrastructure))
- `project_id` (String) ID of the project in TraceForce to post-connect.
- `terraform_module_versions` (String) JSON string containing Terraform module versions
- `terraform_module_versions_hash` (String) Hash of the Terraform module versions for integrity verification
- `terraform_url` (String) URL of the Terraform module repository

### Optional

- `deployed_datalake_ids` (List of String) List of datalake IDs that were deployed by terraform
- `deployed_source_app_ids` (List of String) List of source app IDs that were deployed by terraform

<a id="nestedatt--infrastructure"></a>
### Nested Schema for `infrastructure`

Optional:

- `base` (Attributes) Base infrastructure outputs (see [below for nested schema](#nestedatt--infrastructure--base))
- `bigquery` (Attributes) BigQuery datalake infrastructure outputs (see [below for nested schema](#nestedatt--infrastructure--bigquery))
- `salesforce` (Attributes) Salesforce source app infrastructure outputs (see [below for nested schema](#nestedatt--infrastructure--salesforce))

<a id="nestedatt--infrastructure--base"></a>
### Nested Schema for `infrastructure.base`

Required:

- `dataplane_identity_identifier` (String) Dataplane identity identifier for base infrastructure

Optional:

- `workload_identity_provider_name` (String) Workload identity provider name for external authentication


<a id="nestedatt--infrastructure--bigquery"></a>
### Nested Schema for `infrastructure.bigquery`

Required:

- `events_subscription_name` (String) PubSub subscription name for BigQuery events
- `traceforce_schema` (String) BigQuery dataset ID for TraceForce schema


<a id="nestedatt--infrastructure--salesforce"></a>
### Nested Schema for `infrastructure.salesforce`

Required:

- `salesforce_client_id` (String) Salesforce connected app client ID
- `salesforce_client_secret` (String) Secret Manager resource name for Salesforce client secret
- `salesforce_domain` (String) Salesforce domain (e.g., mycompany.my.salesforce.com)
