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