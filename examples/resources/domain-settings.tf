resource "godaddy_domain_settings" "example" {
  domain     = "example.com"
  locked     = true
  renew_auto = true
}
