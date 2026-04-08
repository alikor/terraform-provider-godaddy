data "godaddy_domain_agreements" "example" {
  tlds    = ["com"]
  privacy = true
}
