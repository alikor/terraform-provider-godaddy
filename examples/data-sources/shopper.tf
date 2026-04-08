data "godaddy_shopper" "current" {
  shopper_id          = "123456789"
  include_customer_id = true
}
