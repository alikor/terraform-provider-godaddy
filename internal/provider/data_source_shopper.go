package provider

import (
	"context"

	"github.com/alikor/terraform-provider-godaddy/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

var _ datasource.DataSource = (*shopperDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*shopperDataSource)(nil)

type shopperDataSource struct {
	client *client.Client
}

func NewShopperDataSource() datasource.DataSource {
	return &shopperDataSource{}
}

func (d *shopperDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "godaddy_shopper"
}

func (d *shopperDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasourceschema.Schema{
		MarkdownDescription: "Read shopper details and optionally resolve customer ID.",
		Attributes: map[string]datasourceschema.Attribute{
			"shopper_id":          datasourceschema.StringAttribute{Required: true, MarkdownDescription: "Shopper ID to read."},
			"include_customer_id": datasourceschema.BoolAttribute{Optional: true, MarkdownDescription: "Whether to request `customerId`. Defaults to true."},
			"customer_id":         datasourceschema.StringAttribute{Computed: true, MarkdownDescription: "Resolved customer ID."},
			"name_first":          datasourceschema.StringAttribute{Computed: true, MarkdownDescription: "Shopper first name."},
			"name_last":           datasourceschema.StringAttribute{Computed: true, MarkdownDescription: "Shopper last name."},
			"email":               datasourceschema.StringAttribute{Computed: true, MarkdownDescription: "Shopper email."},
		},
	}
}

func (d *shopperDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *shopperDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data shopperDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	includeCustomerID := true
	if !data.IncludeCustomerID.IsNull() && !data.IncludeCustomerID.IsUnknown() {
		includeCustomerID = data.IncludeCustomerID.ValueBool()
	}

	shopper, err := d.client.GetShopper(ctx, data.ShopperID.ValueString(), includeCustomerID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read shopper", err.Error())
		return
	}

	data.CustomerID = stringOrNull(shopper.CustomerID)
	data.NameFirst = stringOrNull(shopper.NameFirst)
	data.NameLast = stringOrNull(shopper.NameLast)
	data.Email = stringOrNull(shopper.Email)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
