package provider

import (
	"context"

	"github.com/alikor/terraform-provider-godaddy/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var _ provider.Provider = (*godaddyProvider)(nil)

type godaddyProvider struct {
	version string
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &godaddyProvider{version: version}
	}
}

func (p *godaddyProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "godaddy"
	resp.Version = p.version
}

func (p *godaddyProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = providerSchema()
}

func (p *godaddyProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data providerConfigModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	config, err := expandClientConfig(p.version, data)
	if err != nil {
		resp.Diagnostics.AddError("Invalid provider configuration", err.Error())
		return
	}

	if config.APIKey == "" {
		resp.Diagnostics.AddAttributeError(path.Root("api_key"), "Missing API key", "Set `api_key` or `GODADDY_API_KEY`.")
	}
	if config.APISecret == "" {
		resp.Diagnostics.AddAttributeError(path.Root("api_secret"), "Missing API secret", "Set `api_secret` or `GODADDY_API_SECRET`.")
	}
	if resp.Diagnostics.HasError() {
		return
	}

	if config.RateLimitRPM < 1 || config.RateLimitRPM > 60 {
		resp.Diagnostics.AddAttributeError(path.Root("rate_limit_rpm"), "Invalid rate limit", "`rate_limit_rpm` must be between 1 and 60.")
		return
	}

	if config.BaseURL == "" {
		resp.Diagnostics.AddError("Invalid provider configuration", "base URL resolution produced an empty value")
		return
	}

	providerData := configuredProvider{
		client: client.New(config),
	}

	resp.DataSourceData = providerData
	resp.ResourceData = providerData
}

func (p *godaddyProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewDomainDataSource,
		NewDomainsDataSource,
		NewDNSRecordSetDataSource,
		NewDomainAgreementsDataSource,
		NewDomainActionsDataSource,
		NewDomainForwardingDataSource,
		NewShopperDataSource,
	}
}

func (p *godaddyProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewDNSRecordSetResource,
		NewDomainSettingsResource,
		NewDomainNameserversResource,
		NewDomainContactsResource,
		NewDomainForwardingResource,
		NewDomainDNSSECRecordsResource,
	}
}

func configuredClient(data any) (*client.Client, diag.Diagnostics) {
	var diags diag.Diagnostics

	providerData, ok := data.(configuredProvider)
	if !ok {
		diags.AddError("Unexpected provider data type", "Expected configuredProvider.")
		return nil, diags
	}

	return providerData.client, diags
}

var _ schema.Schema
