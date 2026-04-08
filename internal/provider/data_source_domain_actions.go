package provider

import (
	"context"

	"github.com/alikor/terraform-provider-godaddy/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*domainActionsDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*domainActionsDataSource)(nil)

type domainActionsDataSource struct {
	client *client.Client
}

func NewDomainActionsDataSource() datasource.DataSource {
	return &domainActionsDataSource{}
}

func (d *domainActionsDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "godaddy_domain_actions"
}

func (d *domainActionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasourceschema.Schema{
		MarkdownDescription: "Read recent v2 actions for a GoDaddy domain.",
		Attributes: map[string]datasourceschema.Attribute{
			"domain": datasourceschema.StringAttribute{Required: true, MarkdownDescription: "Domain name to inspect."},
			"actions": datasourceschema.ListAttribute{
				Computed:            true,
				ElementType:         types.ObjectType{AttrTypes: actionAttrTypes},
				MarkdownDescription: "Recent domain actions.",
			},
		},
	}
}

func (d *domainActionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *domainActionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data domainActionsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domain, err := parseDomain(data.Domain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid domain", err.Error())
		return
	}

	customerID, err := d.client.ResolveCustomerID(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to resolve customer_id", err.Error())
		return
	}

	actions, err := d.client.ListDomainActions(ctx, customerID, domain)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read domain actions", err.Error())
		return
	}

	data.Domain = types.StringValue(domain)
	data.Actions = actionsToList(actions)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
