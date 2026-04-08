package provider

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/alikor/terraform-provider-godaddy/internal/client"
)

type configuredProvider struct {
	client *client.Client
}

type providerConfigModel struct {
	APIKey          stringModel `tfsdk:"api_key"`
	APISecret       stringModel `tfsdk:"api_secret"`
	Endpoint        stringModel `tfsdk:"endpoint"`
	BaseURL         stringModel `tfsdk:"base_url"`
	ShopperID       stringModel `tfsdk:"shopper_id"`
	CustomerID      stringModel `tfsdk:"customer_id"`
	AppKey          stringModel `tfsdk:"app_key"`
	MarketID        stringModel `tfsdk:"market_id"`
	RequestTimeout  stringModel `tfsdk:"request_timeout"`
	PollInterval    stringModel `tfsdk:"poll_interval"`
	MaxRetries      int64Model  `tfsdk:"max_retries"`
	RateLimitRPM    int64Model  `tfsdk:"rate_limit_rpm"`
	UserAgentSuffix stringModel `tfsdk:"user_agent_suffix"`
	DebugHTTP       boolModel   `tfsdk:"debug_http"`
}

func expandClientConfig(version string, model providerConfigModel) (client.Config, error) {
	apiKey := model.APIKey.ValueString()
	if apiKey == "" {
		apiKey = os.Getenv("GODADDY_API_KEY")
	}

	apiSecret := model.APISecret.ValueString()
	if apiSecret == "" {
		apiSecret = os.Getenv("GODADDY_API_SECRET")
	}

	endpoint := firstNonEmpty(model.Endpoint.ValueString(), os.Getenv("GODADDY_ENDPOINT"), "production")
	baseURL := firstNonEmpty(model.BaseURL.ValueString(), os.Getenv("GODADDY_BASE_URL"))
	shopperID := firstNonEmpty(model.ShopperID.ValueString(), os.Getenv("GODADDY_SHOPPER_ID"))
	customerID := firstNonEmpty(model.CustomerID.ValueString(), os.Getenv("GODADDY_CUSTOMER_ID"))
	appKey := firstNonEmpty(model.AppKey.ValueString(), os.Getenv("GODADDY_APP_KEY"))
	marketID := firstNonEmpty(model.MarketID.ValueString(), os.Getenv("GODADDY_MARKET_ID"), "en-US")
	requestTimeout, err := parseDurationString(firstNonEmpty(model.RequestTimeout.ValueString(), "30s"))
	if err != nil {
		return client.Config{}, fmt.Errorf("invalid request_timeout: %w", err)
	}

	pollInterval, err := parseDurationString(firstNonEmpty(model.PollInterval.ValueString(), "5s"))
	if err != nil {
		return client.Config{}, fmt.Errorf("invalid poll_interval: %w", err)
	}

	maxRetries := int(model.MaxRetries.ValueInt64())
	if maxRetries == 0 {
		maxRetries = 5
	}

	rateLimitRPM := int(model.RateLimitRPM.ValueInt64())
	if rateLimitRPM == 0 {
		rateLimitRPM = 50
	}

	userAgent := fmt.Sprintf("terraform-provider-godaddy/%s", version)
	if suffix := strings.TrimSpace(model.UserAgentSuffix.ValueString()); suffix != "" {
		userAgent = fmt.Sprintf("%s (%s)", userAgent, suffix)
	}

	return client.Config{
		APIKey:         apiKey,
		APISecret:      apiSecret,
		BaseURL:        client.ResolveBaseURL(endpoint, baseURL),
		ShopperID:      shopperID,
		CustomerID:     customerID,
		AppKey:         appKey,
		MarketID:       marketID,
		RequestTimeout: requestTimeout,
		PollInterval:   pollInterval,
		MaxRetries:     maxRetries,
		RateLimitRPM:   rateLimitRPM,
		UserAgent:      userAgent,
		DebugHTTP:      model.DebugHTTP.ValueBool(),
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func parseDurationString(value string) (time.Duration, error) {
	if value == "" {
		return 0, nil
	}
	if seconds, err := strconv.Atoi(value); err == nil {
		return time.Duration(seconds) * time.Second, nil
	}
	return time.ParseDuration(value)
}
