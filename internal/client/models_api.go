package client

type Domain struct {
	Domain                       string          `json:"domain,omitempty"`
	DomainID                     int64           `json:"domainId,omitempty"`
	Status                       string          `json:"status,omitempty"`
	CreatedAt                    string          `json:"createdAt,omitempty"`
	ExpiresAt                    string          `json:"expires,omitempty"`
	RenewAuto                    bool            `json:"renewAuto,omitempty"`
	RenewDeadline                string          `json:"renewDeadline,omitempty"`
	Locked                       bool            `json:"locked,omitempty"`
	Privacy                      bool            `json:"privacy,omitempty"`
	TransferProtected            bool            `json:"transferProtected,omitempty"`
	ExpirationProtected          bool            `json:"expirationProtected,omitempty"`
	HoldRegistrar                bool            `json:"holdRegistrar,omitempty"`
	NameServers                  []string        `json:"nameServers,omitempty"`
	ExposeRegistrantOrganization bool            `json:"exposeRegistrantOrganization,omitempty"`
	ExposeWhois                  bool            `json:"exposeWhois,omitempty"`
	AuthCode                     string          `json:"authCode,omitempty"`
	Contacts                     *DomainContacts `json:"contacts,omitempty"`
	RegistryStatusCodes          []string        `json:"registryStatusCodes,omitempty"`
	Actions                      []DomainAction  `json:"actions,omitempty"`
	DNSSECRecords                []DNSSECRecord  `json:"dnssecRecords,omitempty"`
}

type DomainSummary struct {
	Domain      string   `json:"domain,omitempty"`
	DomainID    int64    `json:"domainId,omitempty"`
	Status      string   `json:"status,omitempty"`
	CreatedAt   string   `json:"createdAt,omitempty"`
	ExpiresAt   string   `json:"expires,omitempty"`
	NameServers []string `json:"nameServers,omitempty"`
}

type DNSRecord struct {
	Data     string `json:"data,omitempty"`
	TTL      int64  `json:"ttl,omitempty"`
	Priority int64  `json:"priority,omitempty"`
	Weight   int64  `json:"weight,omitempty"`
	Port     int64  `json:"port,omitempty"`
	Protocol string `json:"protocol,omitempty"`
	Service  string `json:"service,omitempty"`
}

type Consent struct {
	AgreedAt      string   `json:"agreedAt,omitempty"`
	AgreedBy      string   `json:"agreedBy,omitempty"`
	AgreementKeys []string `json:"agreementKeys,omitempty"`
}

type Agreement struct {
	AgreementKey string `json:"agreementKey,omitempty"`
	Title        string `json:"title,omitempty"`
	Content      string `json:"content,omitempty"`
	URL          string `json:"url,omitempty"`
}

type Shopper struct {
	ShopperID  string `json:"shopperId,omitempty"`
	CustomerID string `json:"customerId,omitempty"`
	NameFirst  string `json:"nameFirst,omitempty"`
	NameLast   string `json:"nameLast,omitempty"`
	Email      string `json:"email,omitempty"`
}

type DomainContacts struct {
	Registrant Contact `json:"contactRegistrant,omitempty"`
	Admin      Contact `json:"contactAdmin,omitempty"`
	Tech       Contact `json:"contactTech,omitempty"`
	Billing    Contact `json:"contactBilling,omitempty"`
}

type Contact struct {
	NameFirst      string         `json:"nameFirst,omitempty"`
	NameMiddle     string         `json:"nameMiddle,omitempty"`
	NameLast       string         `json:"nameLast,omitempty"`
	Organization   string         `json:"organization,omitempty"`
	JobTitle       string         `json:"jobTitle,omitempty"`
	Email          string         `json:"email,omitempty"`
	Phone          string         `json:"phone,omitempty"`
	Fax            string         `json:"fax,omitempty"`
	AddressMailing MailingAddress `json:"addressMailing,omitempty"`
}

type MailingAddress struct {
	Address1   string `json:"address1,omitempty"`
	Address2   string `json:"address2,omitempty"`
	City       string `json:"city,omitempty"`
	State      string `json:"state,omitempty"`
	PostalCode string `json:"postalCode,omitempty"`
	Country    string `json:"country,omitempty"`
}

type DomainAction struct {
	Type        string        `json:"type,omitempty"`
	Origination string        `json:"origination,omitempty"`
	CreatedAt   string        `json:"createdAt,omitempty"`
	StartedAt   string        `json:"startedAt,omitempty"`
	CompletedAt string        `json:"completedAt,omitempty"`
	ModifiedAt  string        `json:"modifiedAt,omitempty"`
	Status      string        `json:"status,omitempty"`
	RequestID   string        `json:"requestId,omitempty"`
	Reason      *ActionReason `json:"reason,omitempty"`
}

type ActionReason struct {
	Code    string          `json:"code,omitempty"`
	Message string          `json:"message,omitempty"`
	Fields  []APIErrorField `json:"fields,omitempty"`
}

type DNSSECRecord struct {
	KeyTag           int64  `json:"keyTag,omitempty"`
	Algorithm        string `json:"algorithm,omitempty"`
	DigestType       string `json:"digestType,omitempty"`
	Digest           string `json:"digest,omitempty"`
	Flags            string `json:"flags,omitempty"`
	PublicKey        string `json:"publicKey,omitempty"`
	MaxSignatureLife int64  `json:"maxSignatureLife,omitempty"`
}
