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
  base_url   = var.godaddy_base_url
}

resource "godaddy_dns_record_set" "test" {
  domain = var.domain
  type   = "TXT"
  name   = var.record_name

  records = [
    {
      data = var.record_value
      ttl  = 600
    }
  ]
}

variable "godaddy_api_key" {
  type      = string
  sensitive = true
}

variable "godaddy_api_secret" {
  type      = string
  sensitive = true
}

variable "godaddy_base_url" {
  type = string
}

variable "domain" {
  type = string
}

variable "record_name" {
  type = string
}

variable "record_value" {
  type = string
}
