# Example usage of the RDS PostgreSQL parameter group data source

provider "guardium-data-protection" {
  host = "localhost"
  port = "8443"
}

# Retrieve information about an RDS PostgreSQL parameter group
data "guardium-data-protection_rds_postgres_parameter_group" "example" {
  db_identifier = "my-postgres-db"
  region        = "us-west-2"  # Optional: Specify the AWS region
}

# Output the parameter group details
output "parameter_group_name" {
  value = data.guardium-data-protection_rds_postgres_parameter_group.example.parameter_group
}

output "family_name" {
  value = data.guardium-data-protection_rds_postgres_parameter_group.example.family_name
}

output "description" {
  value = data.guardium-data-protection_rds_postgres_parameter_group.example.description
}