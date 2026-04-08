data "godaddy_domains" "all" {
  limit    = 25
  includes = ["nameServers"]
}
