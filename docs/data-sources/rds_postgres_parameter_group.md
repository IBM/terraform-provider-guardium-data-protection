---
page_title: "guardium-data-protection_rds_postgres_parameter_group Data Source - terraform-provider-guardium-data-protection"
subcategory: ""
description: |-
  Data source for AWS RDS PostgreSQL parameter group
---

# guardium-data-protection_rds_postgres_parameter_group (Data Source)

This data source provides information about an AWS RDS PostgreSQL parameter group. It retrieves details about the parameter group associated with a specified RDS PostgreSQL database instance.

## Example Usage

```terraform
data "guardium-data-protection_rds_postgres_parameter_group" "example" {
  db_identifier = "my-postgres-db"
  region        = "us-west-2"  # Optional: Specify the AWS region
}
```

## Argument Reference

* `db_identifier` - (Required) The identifier of the RDS PostgreSQL database instance.
* `region` - (Optional) The AWS region where the RDS PostgreSQL database instance is located. If not specified, the provider's default region will be used.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `parameter_group` - The name of the parameter group associated with the RDS PostgreSQL database instance.
* `family_name` - The family name of the parameter group (e.g., postgres13).
* `description` - The description of the parameter group.
* `id` - The identifier of the data source (same as `db_identifier`).