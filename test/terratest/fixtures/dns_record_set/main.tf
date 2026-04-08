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

resource "godaddy_dns_record_set" "test" {
  domain = var.domain
  type   = "TXT"
  name   = "terratest-codex"

  records = [
    {
      data = "codex-test"
      ttl  = 600
    }
  ]
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
