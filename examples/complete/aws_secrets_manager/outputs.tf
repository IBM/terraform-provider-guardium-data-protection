output "aws_secrets_manager_id" {
  description = "ID of the created AWS Secrets Manager configuration"
  value       = guardium-data-protection_aws_secrets_manager.example.id
}

output "aws_secrets_manager_name" {
  description = "Name of the created AWS Secrets Manager configuration"
  value       = guardium-data-protection_aws_secrets_manager.example.name
}