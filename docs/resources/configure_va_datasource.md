---
page_title: "guardium-data-protection_configure_va_datasource Resource - terraform-provider-guardium-data-protection"
subcategory: ""
description: |-
  Resource for configuring vulnerability assessment for a datasource in Guardium Data Protection
---

# guardium-data-protection_configure_va_datasource (Resource)

This resource allows you to configure vulnerability assessment (VA) for a datasource in Guardium Data Protection. You can set up the assessment schedule, including frequency, day, and time.

## Example Usage

```terraform
# Get authentication token
data "guardium-data-protection_authentication" "access_token" {
  client_id     = "your_client_id"
  client_secret = "your_client_secret"
  username      = "your_username"
  password      = "your_password"
}

# Configure vulnerability assessment for a datasource
resource "guardium-data-protection_configure_va_datasource" "example" {
  # Required parameters
  datasource_name     = "example-datasource"
  assessment_schedule = "weekly"
  assessment_day      = "Monday"
  assessment_time     = "23:00"
  access_token        = data.guardium-data-protection_authentication.access_token.access_token

  # Optional parameters
  enabled             = true
}
```

## Argument Reference

The following arguments are supported:

### Required

* `datasource_name` - (Required) Name of the datasource to configure VA for.
* `assessment_schedule` - (Required) Schedule frequency for vulnerability assessment (e.g., daily, weekly, monthly).
* `assessment_day` - (Required) Day for vulnerability assessment (e.g., Monday for weekly, 1 for monthly).
* `assessment_time` - (Required) Time for vulnerability assessment in 24-hour format (e.g., 23:00).
* `access_token` - (Required, Sensitive) Access token for authentication with the Guardium Data Protection API.

### Optional

* `enabled` - (Optional) Whether vulnerability assessment is enabled. Defaults to `true`.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of the VA configuration, formatted as `va-config-{datasource_name}`.
* `last_configured_time` - Timestamp of when the VA was last configured.

## Import

VA configurations can be imported using the `id` attribute, which is a combination of "va-config-" and the datasource name:

```
$ terraform import guardium-data-protection_configure_va_datasource.example va-config-example-datasource