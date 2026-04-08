package provider

import (
	"context"
	"strings"

	"github.com/alikor/terraform-provider-godaddy/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*domainAgreementsDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*domainAgreementsDataSource)(nil)

type domainAgreementsDataSource struct {
	client *client.Client
}

func NewDomainAgreementsDataSource() datasource.DataSource {
	return &domainAgreementsDataSource{}
}

func (d *domainAgreementsDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "godaddy_domain_agreements"
}

func (d *domainAgreementsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasourceschema.Schema{
		MarkdownDescription: "Read legal agreements required for domain operations.",
		Attributes: map[string]datasourceschema.Attribute{
			"tlds":         datasourceschema.ListAttribute{Required: true, ElementType: types.StringType, MarkdownDescription: "List of TLDs."},
			"privacy":      datasourceschema.BoolAttribute{Required: true, MarkdownDescription: "Whether privacy is requested."},
			"for_transfer": datasourceschema.BoolAttribute{Optional: true, MarkdownDescription: "Whether the agreements are for transfer."},
			"agreements": datasourceschema.ListAttribute{
				Computed:            true,
				ElementType:         types.ObjectType{AttrTypes: agreementAttrTypes},
				MarkdownDescription: "Agreement documents returned by GoDaddy.",
			},
		},
	}
}

func (d *domainAgreementsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *domainAgreementsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data domainAgreementDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tlds, err := stringsFromList(ctx, data.TLDs)
	if err != nil {
		resp.Diagnostics.AddError("Invalid TLD list", err.Error())
		return
	}

	query := makeQuery()
	query.Set("tlds", strings.Join(tlds, ","))
	query.Set("privacy", boolString(data.Privacy.ValueBool()))
	if !data.ForTransfer.IsNull() && !data.ForTransfer.IsUnknown() {
		query.Set("forTransfer", boolString(data.ForTransfer.ValueBool()))
	}

	agreements, err := d.client.GetAgreements(ctx, query)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read agreements", err.Error())
		return
	}

	data.Agreements = agreementsToList(agreements)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
