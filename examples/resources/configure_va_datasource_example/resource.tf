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