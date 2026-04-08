package provider

import (
	"context"

	"github.com/alikor/terraform-provider-godaddy/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*domainForwardingDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*domainForwardingDataSource)(nil)

type domainForwardingDataSource struct {
	client *client.Client
}

func NewDomainForwardingDataSource() datasource.DataSource {
	return &domainForwardingDataSource{}
}

func (d *domainForwardingDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "godaddy_domain_forwarding"
}

func (d *domainForwardingDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasourceschema.Schema{
		MarkdownDescription: "Read GoDaddy URL forwarding for one FQDN.",
		Attributes: map[string]datasourceschema.Attribute{
			"fqdn":         datasourceschema.StringAttribute{Required: true, MarkdownDescription: "Forwarded FQDN."},
			"include_subs": datasourceschema.BoolAttribute{Optional: true, MarkdownDescription: "Reserved for later support."},
			"type":         datasourceschema.StringAttribute{Computed: true, MarkdownDescription: "Forward type."},
			"url":          datasourceschema.StringAttribute{Computed: true, MarkdownDescription: "Destination URL."},
			"subs":         datasourceschema.ListAttribute{Computed: true, ElementType: types.StringType, MarkdownDescription: "Forwarded subdomains, if returned by the API."},
			"mask": datasourceschema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Optional forwarding mask configuration.",
				Attributes: map[string]datasourceschema.Attribute{
					"title":       datasourceschema.StringAttribute{Computed: true, MarkdownDescription: "Mask title."},
					"description": datasourceschema.StringAttribute{Computed: true, MarkdownDescription: "Mask description."},
					"keywords":    datasourceschema.StringAttribute{Computed: true, MarkdownDescription: "Mask keywords."},
				},
			},
		},
	}
}

func (d *domainForwardingDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *domainForwardingDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data forwardingDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fqdn, err := parseFQDN(data.FQDN.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid fqdn", err.Error())
		return
	}

	customerID, err := d.client.ResolveCustomerID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to resolve customer_id", err.Error())
		return
	}

	forwarding, err := d.client.GetDomainForwarding(ctx, customerID, fqdn)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read domain forwarding", err.Error())
		return
	}

	data.FQDN = types.StringValue(fqdn)
	data.Type = stringOrNull(forwarding.Type)
	data.URL = stringOrNull(forwarding.URL)
	data.Mask = forwardMaskObjectFromAPI(forwarding.Mask)
	data.Subs = toStringList(forwarding.Subs)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
