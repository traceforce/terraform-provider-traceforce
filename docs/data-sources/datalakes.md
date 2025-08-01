---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "traceforce_datalakes Data Source - traceforce"
subcategory: ""
description: |-
  
---

# traceforce_datalakes (Data Source)





<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `project_id` (String) Filter datalakes by project ID. If not specified, returns all datalakes.

### Read-Only

- `datalakes` (Attributes List) (see [below for nested schema](#nestedatt--datalakes))

<a id="nestedatt--datalakes"></a>
### Nested Schema for `datalakes`

Read-Only:

- `created_at` (String) Date and time the datalake was created
- `id` (String) System generated ID of the datalake
- `name` (String) Name of the datalake
- `project_id` (String) ID of the project this datalake belongs to
- `status` (String) Status of the datalake. Valid values: pending, deployed, ready, failed.
- `type` (String) Type of datalake. For example, BigQuery, Snowflake, etc.
- `updated_at` (String) Date and time the datalake was last updated
