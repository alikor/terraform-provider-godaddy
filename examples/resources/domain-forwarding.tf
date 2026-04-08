resource "godaddy_domain_forwarding" "blog" {
  fqdn = "blog.example.com"
  type = "REDIRECT_PERMANENT"
  url  = "https://www.example.com/blog"
}
