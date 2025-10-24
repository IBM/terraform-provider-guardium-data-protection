variable "host" {
  description = "The Guardium Data Protection host"
  type        = string
}

variable "port" {
  description = "The Guardium Data Protection port"
  type        = string
}

variable "client_id" {
  description = "The client ID for authentication"
  type        = string
}

variable "client_secret" {
  description = "The client secret for authentication"
  type        = string
  sensitive   = true
}

variable "username" {
  description = "The username for authentication"
  type        = string
}

variable "password" {
  description = "The password for authentication"
  type        = string
  sensitive   = true
}

variable "aws_config_name" {
  description = "Name of the AWS Secrets Manager configuration"
  type        = string
  default     = "my-aws-config"
}

variable "auth_type" {
  description = "Authentication type for AWS Secrets Manager"
  type        = string
  default     = "Security-Credentials"
}

variable "access_key_id" {
  description = "AWS Access Key ID"
  type        = string
  sensitive   = true
}

variable "secret_access_key" {
  description = "AWS Secret Access Key"
  type        = string
  sensitive   = true
}

variable "secret_key_username" {
  description = "Secret Key Username"
  type        = string
}

variable "secret_key_password" {
  description = "Secret Key Password"
  type        = string
  sensitive   = true
}