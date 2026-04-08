package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/alikor/terraform-provider-godaddy/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*domainDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*domainDataSource)(nil)

type domainDataSource struct {
	client *client.Client
}

func NewDomainDataSource() datasource.DataSource {
	return &domainDataSource{}
}

func (d *domainDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "godaddy_domain"
}

func (d *domainDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasourceschema.Schema{
		MarkdownDescription: "Read details for a single GoDaddy domain.",
		Attributes: map[string]datasourceschema.Attribute{
			"domain":                        datasourceschema.StringAttribute{Required: true, MarkdownDescription: "Domain name to read."},
			"include_auth_code":             datasourceschema.BoolAttribute{Optional: true, MarkdownDescription: "Whether to request the sensitive domain auth code via the v2 domain detail endpoint."},
			"include_actions":               datasourceschema.BoolAttribute{Optional: true, MarkdownDescription: "Whether to include recent asynchronous domain actions from the v2 domain detail endpoint."},
			"include_dnssec_records":        datasourceschema.BoolAttribute{Optional: true, MarkdownDescription: "Whether to include DNSSEC records from the v2 domain detail endpoint."},
			"include_registry_status_codes": datasourceschema.BoolAttribute{Optional: true, MarkdownDescription: "Whether to include registry status codes from the v2 domain detail endpoint."},
			"domain_id":                     datasourceschema.Int64Attribute{Computed: true, MarkdownDescription: "GoDaddy domain identifier."},
			"status":                        datasourceschema.StringAttribute{Computed: true, MarkdownDescription: "Domain status."},
			"created_at":                    datasourceschema.StringAttribute{Computed: true, MarkdownDescription: "Creation timestamp."},
			"expires_at":                    datasourceschema.StringAttribute{Computed: true, MarkdownDescription: "Expiration timestamp."},
			"renew_auto":                    datasourceschema.BoolAttribute{Computed: true, MarkdownDescription: "Whether auto-renew is enabled."},
			"renew_deadline":                datasourceschema.StringAttribute{Computed: true, MarkdownDescription: "Renewal deadline timestamp."},
			"locked":                        datasourceschema.BoolAttribute{Computed: true, MarkdownDescription: "Whether the domain is locked."},
			"privacy":                       datasourceschema.BoolAttribute{Computed: true, MarkdownDescription: "Whether privacy is enabled."},
			"transfer_protected":            datasourceschema.BoolAttribute{Computed: true, MarkdownDescription: "Whether transfer protection is enabled."},
			"expiration_protected":          datasourceschema.BoolAttribute{Computed: true, MarkdownDescription: "Whether expiration protection is enabled."},
			"hold_registrar":                datasourceschema.BoolAttribute{Computed: true, MarkdownDescription: "Whether registrar hold is enabled."},
			"name_servers":                  datasourceschema.ListAttribute{Computed: true, ElementType: types.StringType, MarkdownDescription: "Authoritative nameservers."},
			"contacts": datasourceschema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Current domain contact set when returned by GoDaddy.",
				Attributes: map[string]datasourceschema.Attribute{
					"registrant": domainContactAttribute("Registrant contact."),
					"admin":      domainContactAttribute("Admin contact."),
					"tech":       domainContactAttribute("Tech contact."),
					"billing":    domainContactAttribute("Billing contact."),
				},
			},
			"expose_registrant_organization": datasourceschema.BoolAttribute{Computed: true, MarkdownDescription: "WHOIS organization exposure flag."},
			"expose_whois":                   datasourceschema.BoolAttribute{Computed: true, MarkdownDescription: "WHOIS exposure flag."},
			"auth_code":                      datasourceschema.StringAttribute{Computed: true, Sensitive: true, MarkdownDescription: "Auth code returned by the v2 domain detail endpoint when requested."},
			"actions":                        datasourceschema.ListAttribute{Computed: true, ElementType: types.ObjectType{AttrTypes: actionAttrTypes}, MarkdownDescription: "Recent domain actions when requested."},
			"dnssec_records":                 datasourceschema.ListAttribute{Computed: true, ElementType: types.ObjectType{AttrTypes: dnssecAttrTypes}, MarkdownDescription: "DNSSEC records when requested."},
			"registry_status_codes":          datasourceschema.ListAttribute{Computed: true, ElementType: types.StringType, MarkdownDescription: "Registry status codes when requested."},
			"partial":                        datasourceschema.BoolAttribute{Computed: true, MarkdownDescription: "Whether the v2 response was partial due to optional includes being unavailable."},
		},
	}
}

func (d *domainDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, diags := configuredClient(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	d.client = client
}

func (d *domainDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data domainDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain, err := parseDomain(data.Domain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid domain", err.Error())
		return
	}

	includes := requestedDomainIncludes(data)

	var (
		result  *client.Domain
		partial bool
		errRead error
	)
	if len(includes) == 0 {
		result, errRead = d.client.GetDomain(ctx, domain)
	} else {
		customerID, err := d.client.ResolveCustomerID(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Unable to resolve customer_id", err.Error())
			return
		}

		result, partial, errRead = d.client.GetDomainV2(ctx, customerID, domain, includes)
		if partial {
			resp.Diagnostics.AddWarning("Partial domain response", partialDomainWarning(includes))
		}
	}
	if errRead != nil {
		resp.Diagnostics.AddError("Unable to read domain", errRead.Error())
		return
	}

	applyDomainResult(&data, domain, result, partial)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func domainContactAttribute(description string) datasourceschema.Attribute {
	return datasourceschema.SingleNestedAttribute{
		Computed:            true,
		MarkdownDescription: description,
		Attributes: map[string]datasourceschema.Attribute{
			"name_first":   datasourceschema.StringAttribute{Computed: true, MarkdownDescription: "First name."},
			"name_middle":  datasourceschema.StringAttribute{Computed: true, MarkdownDescription: "Middle name."},
			"name_last":    datasourceschema.StringAttribute{Computed: true, MarkdownDescription: "Last name."},
			"organization": datasourceschema.StringAttribute{Computed: true, MarkdownDescription: "Organization name."},
			"job_title":    datasourceschema.StringAttribute{Computed: true, MarkdownDescription: "Job title."},
			"email":        datasourceschema.StringAttribute{Computed: true, MarkdownDescription: "Email address."},
			"phone":        datasourceschema.StringAttribute{Computed: true, MarkdownDescription: "Phone number."},
			"fax":          datasourceschema.StringAttribute{Computed: true, MarkdownDescription: "Fax number."},
			"address_mailing": datasourceschema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Mailing address.",
				Attributes: map[string]datasourceschema.Attribute{
					"address1":    datasourceschema.StringAttribute{Computed: true, MarkdownDescription: "Address line 1."},
					"address2":    datasourceschema.StringAttribute{Computed: true, MarkdownDescription: "Address line 2."},
					"city":        datasourceschema.StringAttribute{Computed: true, MarkdownDescription: "City."},
					"state":       datasourceschema.StringAttribute{Computed: true, MarkdownDescription: "State or province."},
					"postal_code": datasourceschema.StringAttribute{Computed: true, MarkdownDescription: "Postal code."},
					"country":     datasourceschema.StringAttribute{Computed: true, MarkdownDescription: "ISO 3166-1 alpha-2 country code."},
				},
			},
		},
	}
}

func requestedDomainIncludes(data domainDataSourceModel) []string {
	includes := make([]string, 0, 4)
	if data.IncludeAuthCode.ValueBool() {
		includes = append(includes, "authCode")
	}
	if data.IncludeActions.ValueBool() {
		includes = append(includes, "actions")
	}
	if data.IncludeDNSSECRecords.ValueBool() {
		includes = append(includes, "dnssecRecords")
	}
	if data.IncludeRegistryStatusCodes.ValueBool() {
		includes = append(includes, "registryStatusCodes")
	}
	return includes
}

func partialDomainWarning(includes []string) string {
	return fmt.Sprintf("GoDaddy returned a partial v2 domain response. Optional sections may be unavailable for these requested includes: %s.", strings.Join(includes, ", "))
}

func applyDomainResult(data *domainDataSourceModel, domain string, result *client.Domain, partial bool) {
	data.Domain = types.StringValue(domain)
	data.DomainID = types.Int64Value(result.DomainID)
	data.Status = stringOrNull(result.Status)
	data.CreatedAt = stringOrNull(result.CreatedAt)
	data.ExpiresAt = stringOrNull(result.ExpiresAt)
	data.RenewAuto = types.BoolValue(result.RenewAuto)
	data.RenewDeadline = stringOrNull(result.RenewDeadline)
	data.Locked = types.BoolValue(result.Locked)
	data.Privacy = types.BoolValue(result.Privacy)
	data.TransferProtected = types.BoolValue(result.TransferProtected)
	data.ExpirationProtected = types.BoolValue(result.ExpirationProtected)
	data.HoldRegistrar = types.BoolValue(result.HoldRegistrar)
	data.NameServers = toStringList(result.NameServers)
	data.Contacts = domainContactsObjectFromAPI(result.Contacts)
	data.ExposeRegistrantOrganization = types.BoolValue(result.ExposeRegistrantOrganization)
	data.ExposeWhois = types.BoolValue(result.ExposeWhois)
	data.AuthCode = types.StringNull()
	data.Actions = types.ListNull(types.ObjectType{AttrTypes: actionAttrTypes})
	data.DNSSECRecords = types.ListNull(types.ObjectType{AttrTypes: dnssecAttrTypes})
	data.RegistryStatusCodes = types.ListNull(types.StringType)

	if data.IncludeAuthCode.ValueBool() {
		data.AuthCode = stringOrNull(result.AuthCode)
	}
	if data.IncludeActions.ValueBool() {
		data.Actions = actionsToList(result.Actions)
	}
	if data.IncludeDNSSECRecords.ValueBool() {
		data.DNSSECRecords = dnssecRecordsToList(result.DNSSECRecords)
	}
	if data.IncludeRegistryStatusCodes.ValueBool() {
		data.RegistryStatusCodes = toStringList(result.RegistryStatusCodes)
	}

	data.Partial = types.BoolValue(partial)
}
