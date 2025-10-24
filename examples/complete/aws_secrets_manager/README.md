# AWS Secrets Manager Configuration Example

This example demonstrates how to create an AWS Secrets Manager configuration in Guardium Data Protection using the Terraform provider.

## Limitation

This resource has a limitation:

**Delete operations are not supported** by the API. When you run `terraform destroy`, the resource will be removed from the Terraform state, but the configuration will remain in Guardium Data Protection.

## Usage

To run this example, you need to execute:

```bash
$ terraform init
$ terraform plan
$ terraform apply
```

Note that this example may create resources which cost money. Run `terraform destroy` when you don't need these resources anymore.

## Requirements

| Name | Version |
|------|---------|
| terraform | >= 0.13.0 |
| guardium-data-protection | >= 1.0.0 |

## Providers

| Name | Version |
|------|---------|
| guardium-data-protection | >= 1.0.0 |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| host | The Guardium Data Protection host | `string` | n/a | yes |
| port | The Guardium Data Protection port | `string` | n/a | yes |
| client_id | The client ID for authentication | `string` | n/a | yes |
| client_secret | The client secret for authentication | `string` | n/a | yes |
| username | The username for authentication | `string` | n/a | yes |
| password | The password for authentication | `string` | n/a | yes |
| aws_config_name | Name of the AWS Secrets Manager configuration | `string` | `"my-aws-config"` | no |
| auth_type | Authentication type for AWS Secrets Manager | `string` | `"Security-Credentials"` | no |
| access_key_id | AWS Access Key ID | `string` | n/a | yes |
| secret_access_key | AWS Secret Access Key | `string` | n/a | yes |
| secret_key_username | Secret Key Username | `string` | n/a | yes |
| secret_key_password | Secret Key Password | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| aws_secrets_manager_id | ID of the created AWS Secrets Manager configuration |
| aws_secrets_manager_name | Name of the created AWS Secrets Manager configuration |