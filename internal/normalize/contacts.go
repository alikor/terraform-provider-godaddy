package normalize

import (
	"strings"

	"github.com/alikor/terraform-provider-godaddy/internal/client"
)

func Contact(value client.Contact) client.Contact {
	value.NameFirst = strings.TrimSpace(value.NameFirst)
	value.NameMiddle = strings.TrimSpace(value.NameMiddle)
	value.NameLast = strings.TrimSpace(value.NameLast)
	value.Organization = strings.TrimSpace(value.Organization)
	value.JobTitle = strings.TrimSpace(value.JobTitle)
	value.Email = strings.TrimSpace(value.Email)
	value.Phone = strings.TrimSpace(value.Phone)
	value.Fax = strings.TrimSpace(value.Fax)
	value.AddressMailing.Address1 = strings.TrimSpace(value.AddressMailing.Address1)
	value.AddressMailing.Address2 = strings.TrimSpace(value.AddressMailing.Address2)
	value.AddressMailing.City = strings.TrimSpace(value.AddressMailing.City)
	value.AddressMailing.State = strings.TrimSpace(value.AddressMailing.State)
	value.AddressMailing.PostalCode = strings.TrimSpace(value.AddressMailing.PostalCode)
	value.AddressMailing.Country = strings.ToUpper(strings.TrimSpace(value.AddressMailing.Country))
	return value
}
