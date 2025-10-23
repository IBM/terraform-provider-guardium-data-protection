---
page_title: "guardium-data-protection_aws_secrets_manager Resource - terraform-provider-guardium-data-protection"
subcategory: ""
description: |-
  AWS Secrets Manager configuration for Guardium Data Protection
---

# guardium-data-protection_aws_secrets_manager (Resource)

AWS Secrets Manager configuration for Guardium Data Protection. This resource allows you to create, update, and delete AWS Secrets Manager configurations in Guardium Data Protection.

~> **Note:** The resource uses different API methods for different operations:
* `POST` for creating new configurations
* `PUT` for updating existing configurations
* `DELETE` for deleting configurations

## Example Usage

```terraform
# First, get the authentication token
data "guardium-data-protection_authentication" "auth" {
  client_id     = "admin"
  client_secret = "admin"
  username      = "admin"
  password      = "admin"
}

# Create a new AWS Secrets Manager configuration
resource "guardium-data-protection_aws_secrets_manager" "example" {
  access_token       = data.guardium-data-protection_authentication.auth.access_token
  name               = "my-aws-config"
  auth_type          = "Security-Credentials"
  # Example credentials - replace with your own
  access_key_id      = "AKIAIOSFODNN7EXAMPLE"  # Example format, not a real key
  secret_access_key  = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"  # Example format, not a real key
  secret_key_username = "username"
  secret_key_password = "password"
}
```

## Argument Reference

* `access_token` - (Required) Access token for authentication.
* `name` - (Required) Name of the AWS Secrets Manager configuration. Can be any valid name for new configurations, or an existing name to update an existing configuration.
* `auth_type` - (Required) Authentication type (e.g., Security-Credentials, IAM-Role, IAM-Instance-Profile).
* `access_key_id` - (Required) AWS Access Key ID.
* `secret_access_key` - (Required) AWS Secret Access Key.
* `secret_key_username` - (Required) Secret Key Username.
* `secret_key_password` - (Required) Secret Key Password.
* `ca_path` - (Optional) Path to CA certificate.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of the AWS Secrets Manager configuration (same as `name`).

## Import

AWS Secrets Manager configurations can be imported using the `name`, e.g.,

```
$ terraform import guardium-data-protection_aws_secrets_manager.example my-aws-config