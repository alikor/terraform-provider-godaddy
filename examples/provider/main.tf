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
  endpoint   = "production"
}
