terraform {
  required_providers {
    godaddy = {
      source = "alikor/godaddy"
    }
  }
}

provider "godaddy" {
  api_key    = var.godaddy_api_key
  api_secret = var.godaddy_api_secret
  endpoint   = var.godaddy_endpoint
}

data "godaddy_domain" "test" {
  domain = var.domain
}

output "domain_status" {
  value = data.godaddy_domain.test.status
}

variable "godaddy_api_key" {
  type      = string
  sensitive = true
  default   = ""
}

variable "godaddy_api_secret" {
  type      = string
  sensitive = true
  default   = ""
}

variable "godaddy_endpoint" {
  type    = string
  default = "production"
}

variable "domain" {
  type    = string
  default = ""
}
