data "godaddy_domain" "example" {
  domain                        = "example.com"
  include_actions               = true
  include_dnssec_records        = true
  include_registry_status_codes = true
}
