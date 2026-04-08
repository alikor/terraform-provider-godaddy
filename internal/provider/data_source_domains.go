package provider

import (
	"context"

	"github.com/alikor/terraform-provider-godaddy/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*domainsDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*domainsDataSource)(nil)

type domainsDataSource struct {
	client *client.Client
}

func NewDomainsDataSource() datasource.DataSource {
	return &domainsDataSource{}
}

func (d *domainsDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "godaddy_domains"
}

func (d *domainsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasourceschema.Schema{
		MarkdownDescription: "List domains for the configured GoDaddy account.",
		Attributes: map[string]datasourceschema.Attribute{
			"statuses":      datasourceschema.ListAttribute{Optional: true, ElementType: types.StringType, MarkdownDescription: "Optional list of statuses."},
			"status_groups": datasourceschema.ListAttribute{Optional: true, ElementType: types.StringType, MarkdownDescription: "Optional list of status groups."},
			"limit":         datasourceschema.Int64Attribute{Optional: true, MarkdownDescription: "Maximum number of domains to return."},
			"marker":        datasourceschema.StringAttribute{Optional: true, MarkdownDescription: "Marker token for pagination."},
			"includes":      datasourceschema.ListAttribute{Optional: true, ElementType: types.StringType, MarkdownDescription: "Optional includes like `contacts` or `nameServers`."},
			"modified_date": datasourceschema.StringAttribute{Optional: true, MarkdownDescription: "Optional modified date filter."},
			"domains": datasourceschema.ListAttribute{
				Computed:            true,
				ElementType:         types.ObjectType{AttrTypes: domainSummaryAttrTypes},
				MarkdownDescription: "Domain summaries returned by GoDaddy.",
			},
		},
	}
}

func (d *domainsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *domainsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data domainsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query, err := buildDomainQuery(data, ctx)
	if err != nil {
		resp.Diagnostics.AddError("Invalid query", err.Error())
		return
	}

	result, err := d.client.ListDomains(ctx, query)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list domains", err.Error())
		return
	}

	data.Domains = domainSummariesToList(result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
