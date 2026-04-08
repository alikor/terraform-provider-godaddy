package provider

import (
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestParseDurationString(t *testing.T) {
	t.Parallel()

	got, err := parseDurationString("45")
	if err != nil {
		t.Fatalf("parseDurationString() returned error: %v", err)
	}
	if got != 45*time.Second {
		t.Fatalf("parseDurationString() = %v, want %v", got, 45*time.Second)
	}
}

func TestExpandClientConfigUsesEnvFallbacks(t *testing.T) {
	t.Setenv("GODADDY_API_KEY", "key-from-env")
	t.Setenv("GODADDY_API_SECRET", "secret-from-env")
	t.Setenv("GODADDY_ENDPOINT", "ote")

	cfg, err := expandClientConfig("test", providerConfigModel{
		RequestTimeout: types.StringValue("30s"),
		PollInterval:   types.StringValue("5s"),
	})
	if err != nil {
		t.Fatalf("expandClientConfig() returned error: %v", err)
	}
	if cfg.APIKey != "key-from-env" {
		t.Fatalf("APIKey = %q, want key-from-env", cfg.APIKey)
	}
	if cfg.APISecret != "secret-from-env" {
		t.Fatalf("APISecret = %q, want secret-from-env", cfg.APISecret)
	}
	if cfg.BaseURL != "https://api.ote-godaddy.com" {
		t.Fatalf("BaseURL = %q, want ote base URL", cfg.BaseURL)
	}
}

func TestFirstNonEmpty(t *testing.T) {
	t.Parallel()

	got := firstNonEmpty("", " ", "value", "later")
	if got != "value" {
		t.Fatalf("firstNonEmpty() = %q, want value", got)
	}
}

func TestExpandClientConfigDefaults(t *testing.T) {
	_ = os.Unsetenv("GODADDY_API_KEY")
	_ = os.Unsetenv("GODADDY_API_SECRET")

	cfg, err := expandClientConfig("test", providerConfigModel{
		APIKey:         types.StringValue("key"),
		APISecret:      types.StringValue("secret"),
		RequestTimeout: types.StringValue("30s"),
		PollInterval:   types.StringValue("5s"),
	})
	if err != nil {
		t.Fatalf("expandClientConfig() returned error: %v", err)
	}
	if cfg.MaxRetries != 5 {
		t.Fatalf("MaxRetries = %d, want 5", cfg.MaxRetries)
	}
	if cfg.RateLimitRPM != 50 {
		t.Fatalf("RateLimitRPM = %d, want 50", cfg.RateLimitRPM)
	}
}
