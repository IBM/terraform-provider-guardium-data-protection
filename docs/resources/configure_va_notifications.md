---
page_title: "guardium-data-protection_configure_va_notifications Resource - terraform-provider-guardium-data-protection"
subcategory: ""
description: |-
  Resource for configuring vulnerability assessment notifications for a datasource in Guardium Data Protection
---

# guardium-data-protection_configure_va_notifications (Resource)

This resource allows you to configure vulnerability assessment (VA) notifications for a datasource in Guardium Data Protection. You can set up email notifications to be sent when vulnerabilities are detected, based on severity level.

## Example Usage

```terraform
# Get authentication token
data "guardium-data-protection_authentication" "access_token" {
  client_id     = "your_client_id"
  client_secret = "your_client_secret"
  username      = "your_username"
  password      = "your_password"
}

# Configure vulnerability assessment notifications for a datasource
resource "guardium-data-protection_configure_va_notifications" "example" {
  # Required parameters
  datasource_name      = "example-datasource"
  notification_type    = "email"
  notification_emails  = ["admin@example.com", "security@example.com"]
  notification_severity = "high"
  access_token         = data.guardium-data-protection_authentication.access_token.access_token

  # Optional parameters
  enabled              = true
}
```

## Argument Reference

The following arguments are supported:

### Required

* `datasource_name` - (Required) Name of the datasource to configure notifications for.
* `notification_type` - (Required) Type of notification. Currently only "email" is supported.
* `notification_emails` - (Required) List of email addresses to send notifications to.
* `notification_severity` - (Required) Severity level for notifications (e.g., high, medium, low).
* `access_token` - (Required, Sensitive) Access token for authentication with the Guardium Data Protection API.

### Optional

* `enabled` - (Optional) Whether notifications are enabled. Defaults to `true`.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of the notifications configuration, formatted as `va-notifications-{datasource_name}`.
* `last_configured_time` - Timestamp of when the notifications were last configured.

## Import

Notifications configurations can be imported using the `id` attribute, which is a combination of "va-notifications-" and the datasource name:

```
$ terraform import guardium-data-protection_configure_va_notifications.example va-notifications-example-datasource