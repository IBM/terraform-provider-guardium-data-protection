data "guardium-data-protection_authentication" "auth" {
  client_id     = "admin"
  client_secret = "admin"
  username      = "admin"
  password      = "admin"
}

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