package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type stringModel = types.String
type boolModel = types.Bool
type int64Model = types.Int64

type domainDataSourceModel struct {
	Domain                       types.String `tfsdk:"domain"`
	IncludeAuthCode              types.Bool   `tfsdk:"include_auth_code"`
	IncludeActions               types.Bool   `tfsdk:"include_actions"`
	IncludeDNSSECRecords         types.Bool   `tfsdk:"include_dnssec_records"`
	IncludeRegistryStatusCodes   types.Bool   `tfsdk:"include_registry_status_codes"`
	DomainID                     types.Int64  `tfsdk:"domain_id"`
	Status                       types.String `tfsdk:"status"`
	CreatedAt                    types.String `tfsdk:"created_at"`
	ExpiresAt                    types.String `tfsdk:"expires_at"`
	RenewAuto                    types.Bool   `tfsdk:"renew_auto"`
	RenewDeadline                types.String `tfsdk:"renew_deadline"`
	Locked                       types.Bool   `tfsdk:"locked"`
	Privacy                      types.Bool   `tfsdk:"privacy"`
	TransferProtected            types.Bool   `tfsdk:"transfer_protected"`
	ExpirationProtected          types.Bool   `tfsdk:"expiration_protected"`
	HoldRegistrar                types.Bool   `tfsdk:"hold_registrar"`
	NameServers                  types.List   `tfsdk:"name_servers"`
	Contacts                     types.Object `tfsdk:"contacts"`
	ExposeRegistrantOrganization types.Bool   `tfsdk:"expose_registrant_organization"`
	ExposeWhois                  types.Bool   `tfsdk:"expose_whois"`
	AuthCode                     types.String `tfsdk:"auth_code"`
	Actions                      types.List   `tfsdk:"actions"`
	DNSSECRecords                types.List   `tfsdk:"dnssec_records"`
	RegistryStatusCodes          types.List   `tfsdk:"registry_status_codes"`
	Partial                      types.Bool   `tfsdk:"partial"`
}

type domainsDataSourceModel struct {
	Statuses     types.List   `tfsdk:"statuses"`
	StatusGroups types.List   `tfsdk:"status_groups"`
	Limit        types.Int64  `tfsdk:"limit"`
	Marker       types.String `tfsdk:"marker"`
	Includes     types.List   `tfsdk:"includes"`
	ModifiedDate types.String `tfsdk:"modified_date"`
	Domains      types.List   `tfsdk:"domains"`
}

type dnsRecordSetDataSourceModel struct {
	Domain  types.String `tfsdk:"domain"`
	Type    types.String `tfsdk:"type"`
	Name    types.String `tfsdk:"name"`
	FQDN    types.String `tfsdk:"fqdn"`
	Records types.List   `tfsdk:"records"`
}

type dnsRecordSetResourceModel struct {
	ID      types.String `tfsdk:"id"`
	Domain  types.String `tfsdk:"domain"`
	Type    types.String `tfsdk:"type"`
	Name    types.String `tfsdk:"name"`
	FQDN    types.String `tfsdk:"fqdn"`
	Records types.List   `tfsdk:"records"`
}

type settingsResourceModel struct {
	ID                           types.String `tfsdk:"id"`
	Domain                       types.String `tfsdk:"domain"`
	Locked                       types.Bool   `tfsdk:"locked"`
	RenewAuto                    types.Bool   `tfsdk:"renew_auto"`
	ExposeRegistrantOrganization types.Bool   `tfsdk:"expose_registrant_organization"`
	ExposeWhois                  types.Bool   `tfsdk:"expose_whois"`
	Consent                      types.Object `tfsdk:"consent"`
	Status                       types.String `tfsdk:"status"`
	CreatedAt                    types.String `tfsdk:"created_at"`
	ExpiresAt                    types.String `tfsdk:"expires_at"`
	RenewDeadline                types.String `tfsdk:"renew_deadline"`
	Privacy                      types.Bool   `tfsdk:"privacy"`
	TransferProtected            types.Bool   `tfsdk:"transfer_protected"`
	ExpirationProtected          types.Bool   `tfsdk:"expiration_protected"`
	HoldRegistrar                types.Bool   `tfsdk:"hold_registrar"`
	NameServers                  types.List   `tfsdk:"name_servers"`
}

type nameserversResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Domain       types.String `tfsdk:"domain"`
	NameServers  types.List   `tfsdk:"name_servers"`
	Status       types.String `tfsdk:"status"`
	UpdatedViaV2 types.Bool   `tfsdk:"updated_via_v2"`
}

type contactsResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Domain             types.String `tfsdk:"domain"`
	Registrant         types.Object `tfsdk:"registrant"`
	Admin              types.Object `tfsdk:"admin"`
	Tech               types.Object `tfsdk:"tech"`
	Billing            types.Object `tfsdk:"billing"`
	IdentityDocumentID types.String `tfsdk:"identity_document_id"`
}

type consentModel struct {
	AgreedBy      types.String `tfsdk:"agreed_by"`
	AgreedAt      types.String `tfsdk:"agreed_at"`
	AgreementKeys types.List   `tfsdk:"agreement_keys"`
}

type contactModel struct {
	NameFirst      types.String `tfsdk:"name_first"`
	NameMiddle     types.String `tfsdk:"name_middle"`
	NameLast       types.String `tfsdk:"name_last"`
	Organization   types.String `tfsdk:"organization"`
	JobTitle       types.String `tfsdk:"job_title"`
	Email          types.String `tfsdk:"email"`
	Phone          types.String `tfsdk:"phone"`
	Fax            types.String `tfsdk:"fax"`
	AddressMailing types.Object `tfsdk:"address_mailing"`
}

type mailingAddressModel struct {
	Address1   types.String `tfsdk:"address1"`
	Address2   types.String `tfsdk:"address2"`
	City       types.String `tfsdk:"city"`
	State      types.String `tfsdk:"state"`
	PostalCode types.String `tfsdk:"postal_code"`
	Country    types.String `tfsdk:"country"`
}

type domainAgreementDataSourceModel struct {
	TLDs        types.List `tfsdk:"tlds"`
	Privacy     types.Bool `tfsdk:"privacy"`
	ForTransfer types.Bool `tfsdk:"for_transfer"`
	Agreements  types.List `tfsdk:"agreements"`
}

type shopperDataSourceModel struct {
	ShopperID         types.String `tfsdk:"shopper_id"`
	IncludeCustomerID types.Bool   `tfsdk:"include_customer_id"`
	CustomerID        types.String `tfsdk:"customer_id"`
	NameFirst         types.String `tfsdk:"name_first"`
	NameLast          types.String `tfsdk:"name_last"`
	Email             types.String `tfsdk:"email"`
}

type domainActionsDataSourceModel struct {
	Domain  types.String `tfsdk:"domain"`
	Actions types.List   `tfsdk:"actions"`
}

type forwardingDataSourceModel struct {
	FQDN        types.String `tfsdk:"fqdn"`
	IncludeSubs types.Bool   `tfsdk:"include_subs"`
	Type        types.String `tfsdk:"type"`
	URL         types.String `tfsdk:"url"`
	Mask        types.Object `tfsdk:"mask"`
	Subs        types.List   `tfsdk:"subs"`
}

type forwardingResourceModel struct {
	ID   types.String `tfsdk:"id"`
	FQDN types.String `tfsdk:"fqdn"`
	Type types.String `tfsdk:"type"`
	URL  types.String `tfsdk:"url"`
	Mask types.Object `tfsdk:"mask"`
}

type dnssecResourceModel struct {
	ID      types.String `tfsdk:"id"`
	Domain  types.String `tfsdk:"domain"`
	Records types.List   `tfsdk:"records"`
}

type forwardMaskModel struct {
	Title       types.String `tfsdk:"title"`
	Description types.String `tfsdk:"description"`
	Keywords    types.String `tfsdk:"keywords"`
}
