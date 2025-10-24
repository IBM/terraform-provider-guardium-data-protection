# Example usage of the RDS reboot resource

provider "guardium-data-protection" {
  host = "localhost"
  port = "8443"
}

# Resource to reboot an RDS instance
resource "guardium-data-protection_rds_reboot" "example" {
  db_instance_identifier = "my-postgres-db"
  region                 = "us-west-2" # Optional: Specify the AWS region
  force_failover         = false       # Optional: Whether to force a failover during reboot (for Multi-AZ)
}

# Output the last reboot time
output "last_reboot_time" {
  value = guardium-data-protection_rds_reboot.example.last_reboot_time
}