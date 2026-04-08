package acctest

import (
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/alikor/terraform-provider-godaddy/internal/client"
)

const (
	EnvAPIKey    = "GODADDY_API_KEY"
	EnvAPISecret = "GODADDY_API_SECRET"
	EnvEndpoint  = "GODADDY_ENDPOINT"
	EnvDomain    = "GODADDY_TEST_DOMAIN"
)

func RequireEnv(t *testing.T, keys ...string) {
	t.Helper()

	for _, key := range keys {
		if os.Getenv(key) == "" {
			t.Fatalf("%s must be set for acceptance tests", key)
		}
	}
}

func Endpoint() string {
	if endpoint := os.Getenv(EnvEndpoint); endpoint != "" {
		return endpoint
	}
	return "ote"
}

func TestDomain() string {
	return os.Getenv(EnvDomain)
}

func DiscoverTestDomain(t *testing.T) string {
	t.Helper()

	c := NewClient()
	ctx := t.Context()

	if domain := os.Getenv(EnvDomain); domain != "" {
		if _, err := c.GetDomain(ctx, domain); err == nil {
			return domain
		}
	}

	domains, err := c.ListDomains(ctx, url.Values{"limit": []string{"1"}})
	if err != nil {
		t.Skipf("skipping acceptance tests because no accessible test domain could be discovered: %v", err)
	}
	if len(domains) == 0 || domains[0].Domain == "" {
		t.Skip("skipping acceptance tests because the account has no accessible domains")
	}

	return domains[0].Domain
}

func ProviderConfig() string {
	return fmt.Sprintf(`
provider "godaddy" {
  api_key    = %q
  api_secret = %q
  endpoint   = %q
}
`, os.Getenv(EnvAPIKey), os.Getenv(EnvAPISecret), Endpoint())
}

func NewClient() *client.Client {
	return client.New(client.Config{
		APIKey:         os.Getenv(EnvAPIKey),
		APISecret:      os.Getenv(EnvAPISecret),
		BaseURL:        client.ResolveBaseURL(Endpoint(), ""),
		RequestTimeout: 30 * time.Second,
		PollInterval:   5 * time.Second,
		MaxRetries:     5,
		RateLimitRPM:   50,
		UserAgent:      "terraform-provider-godaddy/testacc",
	})
}
