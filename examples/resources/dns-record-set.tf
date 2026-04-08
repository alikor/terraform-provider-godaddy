resource "godaddy_dns_record_set" "www_a" {
  domain = "example.com"
  type   = "A"
  name   = "www"

  records = [
    {
      data = "203.0.113.10"
      ttl  = 600
    }
  ]
}
