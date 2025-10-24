terraform {
  required_providers {
    guardium-data-protection = {
      source  = "hashicorp.com/ibm/guardium-data-protection"
      version = "1.0.0"
    }
  }
}

# Configure terraform to use a local provider build
provider "guardium-data-protection" {
  host = var.host
  port = var.port
}

data "guardium-data-protection_authentication" "auth" {
  client_id     = var.client_id
  client_secret = var.client_secret
  username      = var.username
  password      = var.password
}

resource "guardium-data-protection_aws_secrets_manager" "example" {
  access_token        = data.guardium-data-protection_authentication.auth.access_token
  name                = var.aws_config_name
  auth_type           = var.auth_type
  access_key_id       = var.access_key_id
  secret_access_key   = var.secret_access_key
  secret_key_username = var.secret_key_username
  secret_key_password = var.secret_key_password
}
