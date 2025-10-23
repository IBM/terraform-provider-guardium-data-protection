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