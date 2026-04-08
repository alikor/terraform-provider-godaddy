data "godaddy_domain_forwarding" "blog" {
  fqdn         = "blog.example.com"
  include_subs = true
}
