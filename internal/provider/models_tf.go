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
	ExposeRegistrantOrganization types.Bool   `tfsdk:"expose_registrant_organization"`
	ExposeWhois                  types.Bool   `tfsdk:"expose_whois"`
	AuthCode                     types.String `tfsdk:"auth_code"`
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
