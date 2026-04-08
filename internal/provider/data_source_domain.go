package provider

import (
	"context"

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
			"domain":                         datasourceschema.StringAttribute{Required: true, MarkdownDescription: "Domain name to read."},
			"include_auth_code":              datasourceschema.BoolAttribute{Optional: true, MarkdownDescription: "Reserved for later v2 support."},
			"include_actions":                datasourceschema.BoolAttribute{Optional: true, MarkdownDescription: "Reserved for later v2 support."},
			"include_dnssec_records":         datasourceschema.BoolAttribute{Optional: true, MarkdownDescription: "Reserved for later v2 support."},
			"include_registry_status_codes":  datasourceschema.BoolAttribute{Optional: true, MarkdownDescription: "Reserved for later v2 support."},
			"domain_id":                      datasourceschema.Int64Attribute{Computed: true, MarkdownDescription: "GoDaddy domain identifier."},
			"status":                         datasourceschema.StringAttribute{Computed: true, MarkdownDescription: "Domain status."},
			"created_at":                     datasourceschema.StringAttribute{Computed: true, MarkdownDescription: "Creation timestamp."},
			"expires_at":                     datasourceschema.StringAttribute{Computed: true, MarkdownDescription: "Expiration timestamp."},
			"renew_auto":                     datasourceschema.BoolAttribute{Computed: true, MarkdownDescription: "Whether auto-renew is enabled."},
			"renew_deadline":                 datasourceschema.StringAttribute{Computed: true, MarkdownDescription: "Renewal deadline timestamp."},
			"locked":                         datasourceschema.BoolAttribute{Computed: true, MarkdownDescription: "Whether the domain is locked."},
			"privacy":                        datasourceschema.BoolAttribute{Computed: true, MarkdownDescription: "Whether privacy is enabled."},
			"transfer_protected":             datasourceschema.BoolAttribute{Computed: true, MarkdownDescription: "Whether transfer protection is enabled."},
			"expiration_protected":           datasourceschema.BoolAttribute{Computed: true, MarkdownDescription: "Whether expiration protection is enabled."},
			"hold_registrar":                 datasourceschema.BoolAttribute{Computed: true, MarkdownDescription: "Whether registrar hold is enabled."},
			"name_servers":                   datasourceschema.ListAttribute{Computed: true, ElementType: types.StringType, MarkdownDescription: "Authoritative nameservers."},
			"expose_registrant_organization": datasourceschema.BoolAttribute{Computed: true, MarkdownDescription: "WHOIS organization exposure flag."},
			"expose_whois":                   datasourceschema.BoolAttribute{Computed: true, MarkdownDescription: "WHOIS exposure flag."},
			"auth_code":                      datasourceschema.StringAttribute{Computed: true, Sensitive: true, MarkdownDescription: "Reserved for later v2 support."},
			"registry_status_codes":          datasourceschema.ListAttribute{Computed: true, ElementType: types.StringType, MarkdownDescription: "Reserved for later v2 support."},
			"partial":                        datasourceschema.BoolAttribute{Computed: true, MarkdownDescription: "Whether the response was partial. Always false for the current v1 implementation."},
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

	if data.IncludeActions.ValueBool() || data.IncludeAuthCode.ValueBool() || data.IncludeDNSSECRecords.ValueBool() || data.IncludeRegistryStatusCodes.ValueBool() {
		resp.Diagnostics.AddError("Advanced includes are not implemented yet", "The current implementation supports the v1 read path only.")
		return
	}

	domain, err := parseDomain(data.Domain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid domain", err.Error())
		return
	}

	result, err := d.client.GetDomain(ctx, domain)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read domain", err.Error())
		return
	}

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
	data.ExposeRegistrantOrganization = types.BoolValue(result.ExposeRegistrantOrganization)
	data.ExposeWhois = types.BoolValue(result.ExposeWhois)
	data.AuthCode = types.StringNull()
	data.RegistryStatusCodes = toStringList(result.RegistryStatusCodes)
	data.Partial = types.BoolValue(false)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
