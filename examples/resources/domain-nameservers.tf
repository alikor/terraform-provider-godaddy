resource "godaddy_domain_nameservers" "example" {
  domain = "example.com"

  name_servers = [
    "ns1.example-dns.net",
    "ns2.example-dns.net",
  ]
}
