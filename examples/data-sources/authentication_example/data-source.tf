terraform {
  required_providers {
    guardium-data-protection = {
      source = "ibm/guardium-data-protection"
    }
  }
}

provider "guardium-data-protection" {
  host = "cm-rr-guardium.dev.fyre.ibm.com"
  port = "8443"
}
#
# data "guardium-data-protection_authentication" "access_token" {
#   	client_secret = ""
#   	username = "admin"
#   	password = ""
# }




data "guardium-data-protection_docdb_parameter_group" "access_token" {
  cluster_identifier = "guardium-docdb"
  region             = "us-east-1"
}

output "example" {
  value = data.guardium-data-protection_docdb_parameter_group.access_token.parameter_group
}