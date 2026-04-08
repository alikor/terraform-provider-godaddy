resource "godaddy_domain_dnssec_records" "example" {
  domain = "example.com"

  records = [
    {
      key_tag     = 12345
      algorithm   = "RSASHA256"
      digest_type = "SHA256"
      digest      = "ABCDEF0123456789"
      flags       = "KSK"
    }
  ]
}
