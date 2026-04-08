resource "godaddy_domain_contacts" "example" {
  domain = "example.com"

  registrant {
    name_first = "Jane"
    name_last  = "Doe"
    email      = "jane@example.com"
    phone      = "+1.4805550100"

    address_mailing {
      address1    = "123 Main St"
      city        = "Tempe"
      state       = "AZ"
      postal_code = "85281"
      country     = "US"
    }
  }

  admin {
    name_first = "Jane"
    name_last  = "Doe"
    email      = "jane@example.com"
    phone      = "+1.4805550100"

    address_mailing {
      address1    = "123 Main St"
      city        = "Tempe"
      state       = "AZ"
      postal_code = "85281"
      country     = "US"
    }
  }

  tech {
    name_first = "Jane"
    name_last  = "Doe"
    email      = "jane@example.com"
    phone      = "+1.4805550100"

    address_mailing {
      address1    = "123 Main St"
      city        = "Tempe"
      state       = "AZ"
      postal_code = "85281"
      country     = "US"
    }
  }

  billing {
    name_first = "Jane"
    name_last  = "Doe"
    email      = "jane@example.com"
    phone      = "+1.4805550100"

    address_mailing {
      address1    = "123 Main St"
      city        = "Tempe"
      state       = "AZ"
      postal_code = "85281"
      country     = "US"
    }
  }
}
