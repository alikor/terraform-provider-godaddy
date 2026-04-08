package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/alikor/terraform-provider-godaddy/internal/client"
	"github.com/alikor/terraform-provider-godaddy/internal/normalize"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

var _ datasource.DataSource = (*dnsRecordSetDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*dnsRecordSetDataSource)(nil)

type dnsRecordSetDataSource struct {
	client *client.Client
}

func NewDNSRecordSetDataSource() datasource.DataSource {
	return &dnsRecordSetDataSource{}
}

func (d *dnsRecordSetDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "godaddy_dns_record_set"
}

func (d *dnsRecordSetDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	dnsRecordAttrs := map[string]datasourceschema.Attribute{
		"data":     datasourceschema.StringAttribute{Computed: true, MarkdownDescription: "Record target/value."},
		"ttl":      datasourceschema.Int64Attribute{Computed: true, MarkdownDescription: "Time to live in seconds."},
		"priority": datasourceschema.Int64Attribute{Computed: true, MarkdownDescription: "MX/SRV priority."},
		"weight":   datasourceschema.Int64Attribute{Computed: true, MarkdownDescription: "SRV weight."},
		"port":     datasourceschema.Int64Attribute{Computed: true, MarkdownDescription: "SRV port."},
		"protocol": datasourceschema.StringAttribute{Computed: true, MarkdownDescription: "SRV protocol."},
		"service":  datasourceschema.StringAttribute{Computed: true, MarkdownDescription: "SRV service."},
	}

	resp.Schema = datasourceschema.Schema{
		MarkdownDescription: "Read a DNS RRset without managing it.",
		Attributes: map[string]datasourceschema.Attribute{
			"domain": datasourceschema.StringAttribute{Required: true, MarkdownDescription: "Domain name."},
			"type":   datasourceschema.StringAttribute{Required: true, MarkdownDescription: "RRset type."},
			"name":   datasourceschema.StringAttribute{Optional: true, MarkdownDescription: "RRset name. Defaults to `@`."},
			"fqdn":   datasourceschema.StringAttribute{Computed: true, MarkdownDescription: "Fully qualified domain name for the RRset."},
			"records": datasourceschema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Normalized records in the set.",
				NestedObject: datasourceschema.NestedAttributeObject{
					Attributes: dnsRecordAttrs,
				},
			},
		},
	}
}

func (d *dnsRecordSetDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *dnsRecordSetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data dnsRecordSetDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain, err := parseDomain(data.Domain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid domain", err.Error())
		return
	}

	recordType := normalize.RecordType(data.Type.ValueString())
	recordName := normalize.RecordName(data.Name.ValueString())
	records, err := d.client.GetDNSRecordSet(ctx, domain, recordType, recordName)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read DNS record set", err.Error())
		return
	}

	data.Domain = stringOrNull(domain)
	data.Type = stringOrNull(recordType)
	data.Name = stringOrNull(recordName)
	data.FQDN = stringOrNull(recordSetFQDN(domain, recordName))
	data.Records = recordsToList(records)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func recordSetFQDN(domain, name string) string {
	if name == "@" || strings.TrimSpace(name) == "" {
		return domain
	}
	return fmt.Sprintf("%s.%s", name, domain)
}
