---
page_title: "guardium-data-protection_register_va_datasource Resource - terraform-provider-guardium-data-protection"
subcategory: ""
description: |-
  Resource for registering a VA datasource in Guardium Data Protection
---

# guardium-data-protection_register_va_datasource (Resource)

This resource allows you to register a vulnerability assessment (VA) datasource in Guardium Data Protection. Datasources are the database instances that you want to monitor and protect.

## Example Usage

```terraform
# Get authentication token
data "guardium-data-protection_authentication" "access_token" {
  client_id     = "your_client_id"
  client_secret = "your_client_secret"
  username      = "your_username"
  password      = "your_password"
}

# Register a VA datasource
resource "guardium-data-protection_register_va_datasource" "example" {
  # Required parameters
  datasource_name     = "example-datasource"
  datasource_type     = "DB2"
  datasource_hostname = "db.example.com"
  datasource_port     = 50000
  application         = "Example App"
  access_token        = data.guardium-data-protection_authentication.access_token.access_token

  # Optional parameters
  datasource_description = "Example datasource for demonstration"
  datasource_database    = "EXAMPLEDB"
  connection_username    = "db_user"
  connection_password    = "db_password"
  severity_level         = "high"
  
  # Boolean flags
  save_password        = true
  use_ssl              = true
  import_server_ssl_cert = true
}
```

## Argument Reference

The following arguments are supported:

### Required

* `datasource_name` - (Required) Name of the datasource.
* `datasource_type` - (Required) Type of the datasource (e.g., DB2, MSSQL, Oracle, PostgreSQL).
* `datasource_hostname` - (Required) Hostname or IP address of the datasource.
* `datasource_port` - (Required) Port number of the datasource.
* `application` - (Required) Application name associated with the datasource.
* `access_token` - (Required, Sensitive) Access token for authentication with the Guardium Data Protection API.

### Optional

* `datasource_description` - (Optional) Description of the datasource.
* `datasource_database` - (Optional) Database name.
* `connection_username` - (Optional) Username for connecting to the datasource.
* `connection_password` - (Optional, Sensitive) Password for connecting to the datasource.
* `severity_level` - (Optional) Severity level for the datasource.
* `service_name` - (Optional) Service name for the datasource.
* `shared_datasource` - (Optional) Shared datasource configuration.
* `connection_properties` - (Optional) Connection properties for the datasource.
* `compatibility_mode` - (Optional) Compatibility mode for the datasource.
* `custom_url` - (Optional) Custom URL for the datasource.
* `kerberos_config_name` - (Optional) Kerberos configuration name.
* `external_password_type_name` - (Optional) External password type name.
* `cyberark_config_name` - (Optional) CyberArk configuration name.
* `cyberark_object_name` - (Optional) CyberArk object name.
* `hashicorp_config_name` - (Optional) HashiCorp configuration name.
* `hashicorp_path` - (Optional) HashiCorp path.
* `hashicorp_role` - (Optional) HashiCorp role.
* `hashicorp_child_namespace` - (Optional) HashiCorp child namespace.
* `aws_secrets_manager_config_name` - (Optional) AWS Secrets Manager configuration name.
* `region` - (Optional) AWS region.
* `secret_name` - (Optional) Secret name.
* `db_instance_account` - (Optional) DB instance account.
* `db_instance_directory` - (Optional) DB instance directory.

### Boolean Flags

* `save_password` - (Optional) Whether to save the password. Defaults to `false`.
* `use_ssl` - (Optional) Whether to use SSL for the connection. Defaults to `false`.
* `import_server_ssl_cert` - (Optional) Whether to import the server SSL certificate. Defaults to `false`.
* `use_kerberos` - (Optional) Whether to use Kerberos authentication. Defaults to `false`.
* `use_ldap` - (Optional) Whether to use LDAP authentication. Defaults to `false`.
* `use_external_password` - (Optional) Whether to use an external password. Defaults to `false`.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of the datasource, formatted as `{datasource_name}-{datasource_hostname}`.
* `last_registered_time` - Timestamp of when the datasource was last registered.

## Import

Datasources can be imported using the `id` attribute, which is a combination of the datasource name and hostname:

```
$ terraform import guardium-data-protection_register_va_datasource.example example-datasource-db.example.com