---
page_title: "guardium-data-protection_rds_reboot Resource - terraform-provider-guardium-data-protection"
subcategory: ""
description: |-
  Resource for rebooting an AWS RDS instance
---

# guardium-data-protection_rds_reboot (Resource)

This resource allows you to reboot an AWS RDS instance. It can be used to apply parameter changes or perform maintenance operations that require a database restart.

## Example Usage

```terraform
resource "guardium-data-protection_rds_reboot" "example" {
  db_instance_identifier = "my-postgres-db"
  region                 = "us-west-2"
  force_failover         = false
}
```

## Argument Reference

* `db_instance_identifier` - (Required) The identifier of the RDS instance to reboot.
* `region` - (Optional) The AWS region where the RDS instance is located. If not specified, the provider's default region will be used.
* `force_failover` - (Optional) When true, the reboot is conducted through a MultiAZ failover. Default is false.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `last_reboot_time` - The timestamp of when the reboot operation was performed.
* `id` - The identifier of the resource (same as `db_instance_identifier`).

## Import

RDS reboot resources can be imported using the `db_instance_identifier`:

```
$ terraform import guardium-data-protection_rds_reboot.example my-postgres-db
```

## Notes

* This resource will trigger a reboot operation each time it is created or updated.
* The reboot operation will wait for the RDS instance to become available again before completing.
* For Multi-AZ deployments, you can use the `force_failover` option to perform a failover during the reboot.