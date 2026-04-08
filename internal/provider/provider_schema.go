package provider

import (
	providerschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
)

func providerSchema() providerschema.Schema {
	return providerschema.Schema{
		MarkdownDescription: "Manage GoDaddy domains, DNS, and selected account-backed domain operations.",
		Attributes: map[string]providerschema.Attribute{
			"api_key": providerschema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "GoDaddy API key. Falls back to `GODADDY_API_KEY`.",
			},
			"api_secret": providerschema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "GoDaddy API secret. Falls back to `GODADDY_API_SECRET`.",
			},
			"endpoint": providerschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Endpoint selector: `production` or `ote`. Defaults to `production`.",
			},
			"base_url": providerschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional base URL override, primarily for tests.",
			},
			"shopper_id": providerschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional shopper context. Falls back to `GODADDY_SHOPPER_ID`.",
			},
			"customer_id": providerschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional customer ID for v2 APIs. Falls back to `GODADDY_CUSTOMER_ID`.",
			},
			"app_key": providerschema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Optional app key for subscriptions surfaces.",
			},
			"market_id": providerschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional market ID for agreements and subscriptions APIs.",
			},
			"request_timeout": providerschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Request timeout as a duration like `30s` or integer seconds.",
			},
			"poll_interval": providerschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Polling interval as a duration like `5s` or integer seconds.",
			},
			"max_retries": providerschema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Maximum number of retries for retriable requests.",
			},
			"rate_limit_rpm": providerschema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Per-endpoint rate limit in requests per minute, clamped to 1-60.",
			},
			"user_agent_suffix": providerschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional suffix appended to the provider user agent.",
			},
			"debug_http": providerschema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Enable redacted HTTP request logging.",
			},
		},
	}
}
